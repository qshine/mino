// Package eval runs the Agent against declared cases and grades the outcome
// with deterministic checks. Replay mode drives the real loop with a scripted
// local model; live mode uses the configured service.
package eval

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/qshine/mino/internal/skills"
	"gopkg.in/yaml.v3"
)

const (
	maxCaseBytes    = 64 << 10
	maxCases        = 256
	maxGraders      = 32
	maxSteps        = 16
	maxSeededFiles  = 64
	maxSeedBytes    = 64 << 10
	maxTextBytes    = 16 << 10
	maxPatternBytes = 1 << 10
	MaxRuns         = 32
)

// Case is one declared interaction: the prompt, the environment it runs in,
// the scripted model answers for replay mode, and the checks that decide a pass.
type Case struct {
	Name, Description, Prompt, Instructions string
	Approve                                 Approval
	Runs                                    int
	Workspace                               Workspace
	Script                                  []Step
	Graders                                 []Grader
}

// Approval answers tool confirmations without a terminal. Denials are the
// normal way to test what the Agent does when the user says no.
type Approval struct {
	All   bool
	Tools []string
}

func (a Approval) Allows(tool string) bool { return a.All || slices.Contains(a.Tools, tool) }

func (a *Approval) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Value {
		case "all":
			*a = Approval{All: true}
			return nil
		case "none":
			*a = Approval{}
			return nil
		}
	case yaml.SequenceNode:
		var tools []string
		if err := node.Decode(&tools); err == nil {
			for _, tool := range tools {
				if strings.TrimSpace(tool) == "" {
					return errors.New("approve lists non-empty tool names")
				}
			}
			*a = Approval{Tools: tools}
			return nil
		}
	}
	return errors.New("approve must be all, none, or a list of tool names")
}

type Workspace struct {
	Files map[string]string `yaml:"files"`
}

// Step is one scripted model response: either final text or one tool call.
type Step struct {
	Text                string
	Call                *ScriptCall
	ExpectInputContains string
}

type ScriptCall struct {
	Tool      string
	Arguments json.RawMessage
}

func (s *Step) UnmarshalYAML(node *yaml.Node) error {
	var raw struct {
		Text *string `yaml:"text"`
		Call *struct {
			Tool      string         `yaml:"tool"`
			Arguments map[string]any `yaml:"arguments"`
		} `yaml:"call"`
		ExpectInputContains string `yaml:"expect_input_contains"`
	}
	if err := strictDecode(node, &raw, "text", "call", "expect_input_contains"); err != nil {
		return err
	}
	if (raw.Text == nil) == (raw.Call == nil) {
		return errors.New("each model step needs exactly one of text or call")
	}
	*s = Step{ExpectInputContains: raw.ExpectInputContains}
	if raw.Text != nil {
		s.Text = *raw.Text
		return nil
	}
	if raw.Call.Arguments == nil {
		raw.Call.Arguments = map[string]any{}
	}
	arguments, err := json.Marshal(raw.Call.Arguments)
	if err != nil {
		return fmt.Errorf("call arguments must be a JSON object: %w", err)
	}
	s.Call = &ScriptCall{Tool: raw.Call.Tool, Arguments: arguments}
	return nil
}

// Grader is one deterministic check. Kind selects the check; the other fields
// hold its parameters.
type Grader struct {
	Kind    string
	Text    string
	Pattern *regexp.Regexp
	Tool    string
	Tools   []string
	Call    int
	Status  string
	Path    string
	Limit   int
}

func (g Grader) String() string {
	switch g.Kind {
	case "final_text_contains", "final_text_matches":
		return g.Kind + " " + fmt.Sprintf("%q", g.Text)
	case "tool_called", "tool_not_called":
		return g.Kind + " " + g.Tool
	case "tool_order":
		return g.Kind + " [" + strings.Join(g.Tools, " ") + "]"
	case "result_status":
		return fmt.Sprintf("%s call %d = %s", g.Kind, g.Call, g.Status)
	case "file_exists":
		return g.Kind + " " + g.Path
	case "file_contains":
		return fmt.Sprintf("%s %s %q", g.Kind, g.Path, g.Text)
	case "max_requests", "max_tool_calls":
		return fmt.Sprintf("%s %d", g.Kind, g.Limit)
	}
	return g.Kind
}

func (g *Grader) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode || len(node.Content) != 2 {
		return errors.New("each grader is a map with exactly one key")
	}
	kind, value := node.Content[0].Value, node.Content[1]
	*g = Grader{Kind: kind}
	switch kind {
	case "final_text_contains":
		return value.Decode(&g.Text)
	case "final_text_matches":
		if err := value.Decode(&g.Text); err != nil {
			return err
		}
		if len(g.Text) > maxPatternBytes {
			return errors.New("final_text_matches pattern exceeds 1 KiB")
		}
		pattern, err := regexp.Compile(g.Text)
		if err != nil {
			return fmt.Errorf("final_text_matches: %w", err)
		}
		g.Pattern = pattern
		return nil
	case "tool_called", "tool_not_called":
		return value.Decode(&g.Tool)
	case "tool_order":
		return value.Decode(&g.Tools)
	case "result_status":
		var raw struct {
			Call   int    `yaml:"call"`
			Status string `yaml:"status"`
		}
		if err := strictDecode(value, &raw, "call", "status"); err != nil {
			return err
		}
		g.Call, g.Status = raw.Call, raw.Status
		return nil
	case "file_exists":
		return value.Decode(&g.Path)
	case "file_contains":
		var raw struct {
			Path string `yaml:"path"`
			Text string `yaml:"text"`
		}
		if err := strictDecode(value, &raw, "path", "text"); err != nil {
			return err
		}
		g.Path, g.Text = raw.Path, raw.Text
		return nil
	case "max_requests", "max_tool_calls":
		return value.Decode(&g.Limit)
	}
	return fmt.Errorf("unknown grader %q", kind)
}

// strictDecode rejects unknown keys so a misspelled field fails loudly instead
// of silently weakening a case.
func strictDecode(node *yaml.Node, target any, allowed ...string) error {
	if node.Kind != yaml.MappingNode {
		return errors.New("expected a map")
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if !slices.Contains(allowed, node.Content[i].Value) {
			return fmt.Errorf("unknown field %q", node.Content[i].Value)
		}
	}
	return node.Decode(target)
}

// Load reads one case file or every *.yaml file directly inside a directory.
func Load(path string) ([]Case, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read eval cases: %w", err)
	}
	var files []string
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("Cannot read eval cases: %w", err)
		}
		for _, entry := range entries {
			ext := filepath.Ext(entry.Name())
			if entry.Type().IsRegular() && (ext == ".yaml" || ext == ".yml") {
				files = append(files, filepath.Join(path, entry.Name()))
			}
		}
		slices.Sort(files)
	} else {
		files = []string{path}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("No eval cases found in %s (expected *.yaml files)", path)
	}
	if len(files) > maxCases {
		return nil, fmt.Errorf("Too many eval cases in %s (limit %d)", path, maxCases)
	}
	cases := make([]Case, 0, len(files))
	for _, file := range files {
		c, err := loadFile(file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		cases = append(cases, c)
	}
	return cases, nil
}

func loadFile(path string) (Case, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Case{}, err
	}
	if !info.Mode().IsRegular() {
		return Case{}, errors.New("case must be a regular file")
	}
	if info.Size() > maxCaseBytes {
		return Case{}, errors.New("case file exceeds 64 KiB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Case{}, err
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return Parse(stem, data)
}

// Parse decodes and validates one case document. name is the file stem, which
// the declared name must match so reports and filters agree with the file.
func Parse(name string, data []byte) (Case, error) {
	var raw struct {
		Name         string    `yaml:"name"`
		Description  string    `yaml:"description"`
		Prompt       string    `yaml:"prompt"`
		Instructions string    `yaml:"instructions"`
		Approve      *Approval `yaml:"approve"`
		Runs         int       `yaml:"runs"`
		Workspace    Workspace `yaml:"workspace"`
		Model        []Step    `yaml:"model"`
		Graders      []Grader  `yaml:"graders"`
	}
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil {
		return Case{}, fmt.Errorf("invalid case: %w", err)
	}
	c := Case{Name: raw.Name, Description: raw.Description, Prompt: raw.Prompt, Instructions: raw.Instructions, Approve: Approval{All: true}, Runs: raw.Runs, Workspace: raw.Workspace, Script: raw.Model, Graders: raw.Graders}
	if raw.Approve != nil {
		c.Approve = *raw.Approve
	}
	if err := c.validate(name); err != nil {
		return Case{}, err
	}
	return c, nil
}

func validText(text string, limit int) bool {
	return utf8.ValidString(text) && !strings.ContainsRune(text, 0) && len(text) <= limit
}

func validRelativePath(path string) bool {
	return path != "" && validText(path, 1024) && filepath.IsLocal(path) && filepath.Clean(path) == path && !filepath.IsAbs(path)
}

func (c Case) validate(stem string) error {
	if !skills.ValidName(c.Name) {
		return errors.New("name must use lowercase letters, digits, and single hyphens (at most 64 bytes)")
	}
	if c.Name != stem {
		return fmt.Errorf("name %q must match the file name %q", c.Name, stem)
	}
	if strings.TrimSpace(c.Prompt) == "" || !validText(c.Prompt, maxTextBytes) {
		return errors.New("prompt must be non-empty UTF-8 text of at most 16 KiB")
	}
	if !validText(c.Instructions, maxTextBytes) || !validText(c.Description, 1<<10) {
		return errors.New("instructions (16 KiB) and description (1 KiB) must be UTF-8 text within their limits")
	}
	if c.Runs < 0 || c.Runs > MaxRuns {
		return fmt.Errorf("runs must be between 1 and %d", MaxRuns)
	}
	if len(c.Workspace.Files) > maxSeededFiles {
		return fmt.Errorf("workspace seeds at most %d files", maxSeededFiles)
	}
	for path, content := range c.Workspace.Files {
		if !validRelativePath(path) {
			return fmt.Errorf("workspace file %q must be a clean relative path inside the workspace", path)
		}
		if !validText(content, maxSeedBytes) {
			return fmt.Errorf("workspace file %q must be UTF-8 text of at most 64 KiB", path)
		}
	}
	if len(c.Script) > maxSteps {
		return fmt.Errorf("model script has at most %d steps", maxSteps)
	}
	for i, step := range c.Script {
		if step.Call != nil && (strings.TrimSpace(step.Call.Tool) == "" || len(step.Call.Arguments) > maxTextBytes) {
			return fmt.Errorf("model step %d needs a tool name and arguments of at most 16 KiB", i+1)
		}
		if step.Call == nil && !validText(step.Text, maxTextBytes) || strings.TrimSpace(step.Text) == "" && step.Call == nil {
			return fmt.Errorf("model step %d text must be non-empty UTF-8 of at most 16 KiB", i+1)
		}
		if !validText(step.ExpectInputContains, maxPatternBytes) {
			return fmt.Errorf("model step %d expect_input_contains exceeds 1 KiB", i+1)
		}
	}
	if len(c.Graders) == 0 || len(c.Graders) > maxGraders {
		return fmt.Errorf("graders must list between 1 and %d checks", maxGraders)
	}
	for i, g := range c.Graders {
		if err := g.validate(); err != nil {
			return fmt.Errorf("grader %d: %w", i+1, err)
		}
	}
	return nil
}

func (g Grader) validate() error {
	switch g.Kind {
	case "final_text_contains":
		if g.Text == "" || !validText(g.Text, maxTextBytes) {
			return errors.New("final_text_contains needs non-empty text")
		}
	case "final_text_matches":
		if g.Pattern == nil {
			return errors.New("final_text_matches needs a pattern")
		}
	case "tool_called", "tool_not_called":
		if strings.TrimSpace(g.Tool) == "" {
			return errors.New(g.Kind + " needs a tool name")
		}
	case "tool_order":
		if len(g.Tools) == 0 || slices.Contains(g.Tools, "") {
			return errors.New("tool_order needs a non-empty list of tool names")
		}
	case "result_status":
		if g.Call < 1 || strings.TrimSpace(g.Status) == "" {
			return errors.New("result_status needs call (1-based) and status")
		}
	case "file_exists":
		if !validRelativePath(g.Path) {
			return errors.New("file_exists needs a clean relative path")
		}
	case "file_contains":
		if !validRelativePath(g.Path) || g.Text == "" || !validText(g.Text, maxTextBytes) {
			return errors.New("file_contains needs a clean relative path and non-empty text")
		}
	case "max_requests", "max_tool_calls":
		if g.Limit < 0 {
			return errors.New(g.Kind + " must not be negative")
		}
	default:
		return fmt.Errorf("unknown grader %q", g.Kind)
	}
	return nil
}
