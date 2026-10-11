package mino

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/qshine/mino/internal/gateway"
	"github.com/qshine/mino/internal/mcp"
)

const mcpUsage = `Usage:
  mino mcp add NAME URL [--bearer] [--header NAME]...
  mino mcp add NAME [--env NAME]... [--cwd DIR] -- COMMAND [ARG]...
  mino mcp list
  mino mcp remove NAME

Servers are saved in ~/.mino/mcp.json (mode 0600) and connect when chat starts.
Header and environment values are entered without echo and never shown again.
--bearer stores an Authorization: Bearer header from a hidden token prompt.`

// runMCP manages ~/.mino/mcp.json. It never connects to a server: connection
// status is reported when chat starts.
func runMCP(ctx context.Context, cli *gateway.CLI, args []string, output io.Writer) error {
	directory, err := userDirectory()
	if err != nil {
		return err
	}
	path := filepath.Join(directory, "mcp.json")
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		_, err := fmt.Fprintln(output, mcpUsage)
		return err
	}
	config, err := mcp.Load(path)
	if err != nil {
		return err
	}
	switch {
	case args[0] == "list" && len(args) == 1:
		return listMCP(config, output)
	case args[0] == "remove" && len(args) == 2:
		if _, ok := config.Servers[args[1]]; !ok {
			return fmt.Errorf("No MCP server named %q", args[1])
		}
		delete(config.Servers, args[1])
		if err := mcp.Save(path, config); err != nil {
			return err
		}
		_, err := fmt.Fprintf(output, "Removed MCP server %s.\n", args[1])
		return err
	case args[0] == "add" && len(args) >= 3:
		name := args[1]
		if !mcp.ValidName(name) {
			return errors.New("Server names use lowercase letters, digits, and single hyphens (at most 32 characters)")
		}
		if _, ok := config.Servers[name]; ok {
			return fmt.Errorf("MCP server %s already exists; remove it first", name)
		}
		server, err := parseAdd(ctx, cli, args[2:])
		if err != nil {
			return err
		}
		config.Servers[name] = server
		if err := mcp.Save(path, config); err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "Added MCP server %s (%s). It connects the next time chat starts; each tool call asks for approval.\n", name, gateway.Text(server.Destination()))
		return err
	}
	return errors.New("Unknown mcp command or unexpected arguments. Run mino mcp help for usage.")
}

func parseAdd(ctx context.Context, cli *gateway.CLI, args []string) (mcp.Server, error) {
	var headers, env []string
	bearer, cwd, rest := false, "", []string(nil)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--bearer":
			bearer = true
		case "--header", "--env", "--cwd":
			if i+1 == len(args) {
				return mcp.Server{}, fmt.Errorf("%s requires a value", args[i])
			}
			switch args[i] {
			case "--header":
				headers = append(headers, args[i+1])
			case "--env":
				env = append(env, args[i+1])
			default:
				cwd = args[i+1]
			}
			i++
		case "--":
			rest = append(rest, args[i:]...)
			i = len(args)
		default:
			rest = append(rest, args[i])
		}
	}
	server := mcp.Server{}
	if len(rest) > 0 && rest[0] == "--" {
		if len(rest) < 2 || bearer || len(headers) > 0 {
			return mcp.Server{}, errors.New("A stdio server needs a command after -- and accepts --env and --cwd, not headers")
		}
		command, err := exec.LookPath(rest[1])
		if err == nil {
			command, err = filepath.Abs(command)
		}
		if err != nil {
			return mcp.Server{}, fmt.Errorf("Cannot find command %q", rest[1])
		}
		if cwd == "" {
			if cwd, err = os.UserHomeDir(); err != nil {
				return mcp.Server{}, err
			}
		}
		if cwd, err = filepath.Abs(cwd); err != nil {
			return mcp.Server{}, err
		}
		if info, err := os.Stat(cwd); err != nil || !info.IsDir() {
			return mcp.Server{}, errors.New("--cwd must be an existing directory")
		}
		server = mcp.Server{Type: "stdio", Command: filepath.Clean(command), Args: rest[2:], Dir: filepath.Clean(cwd)}
	} else {
		if len(rest) != 1 || len(env) > 0 || cwd != "" {
			return mcp.Server{}, errors.New("A remote server takes one URL and accepts --bearer and --header")
		}
		if err := mcp.ValidateURL(rest[0]); err != nil {
			return mcp.Server{}, err
		}
		server = mcp.Server{Type: "http", URL: rest[0]}
	}
	if bearer && slices.ContainsFunc(headers, func(h string) bool { return strings.EqualFold(h, "Authorization") }) {
		return mcp.Server{}, errors.New("Use either --bearer or --header Authorization")
	}
	for _, name := range headers {
		if !mcp.ValidHeaderName(name) {
			return mcp.Server{}, fmt.Errorf("Invalid or reserved header name %q", name)
		}
	}
	for _, name := range env {
		if !mcp.ValidEnvName(name) {
			return mcp.Server{}, fmt.Errorf("Invalid or reserved environment variable %q", name)
		}
	}
	if (bearer || len(headers) > 0 || len(env) > 0) && !cli.Interactive() {
		return mcp.Server{}, errors.New("Entering header or environment values requires an interactive terminal")
	}
	secret := func(label string) (string, error) {
		return cli.PromptConfig(ctx, label, "", true, func(value string) error {
			if strings.ContainsAny(value, "\x00\r\n") || len(value) > 8<<10 {
				return errors.New("Value must be a single line of at most 8 KiB")
			}
			return nil
		})
	}
	if bearer {
		token, err := secret("Bearer token")
		if err != nil {
			return mcp.Server{}, err
		}
		server.Headers = map[string]string{"Authorization": "Bearer " + token}
	}
	for _, name := range headers {
		value, err := secret("Value for header " + name)
		if err != nil {
			return mcp.Server{}, err
		}
		if server.Headers == nil {
			server.Headers = map[string]string{}
		}
		server.Headers[name] = value
	}
	for _, name := range env {
		value, err := secret("Value for environment variable " + name)
		if err != nil {
			return mcp.Server{}, err
		}
		if server.Env == nil {
			server.Env = map[string]string{}
		}
		server.Env[name] = value
	}
	return server, server.Validate()
}

// listMCP shows names of secret settings, never their values.
func listMCP(config mcp.Config, output io.Writer) error {
	if len(config.Servers) == 0 {
		_, err := fmt.Fprintln(output, "No MCP servers configured. Add one with mino mcp add.")
		return err
	}
	names := make([]string, 0, len(config.Servers))
	for name := range config.Servers {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		s := config.Servers[name]
		line := fmt.Sprintf("%s  %s  %s", name, s.Type, s.Destination())
		if s.Type == "stdio" {
			line += "  " + strings.Join(s.Args, " ") + "  cwd=" + s.Dir
		}
		if keys := sortedKeys(s.Headers); len(keys) > 0 {
			line += "  headers=" + strings.Join(keys, ",") + " (values hidden)"
		}
		if keys := sortedKeys(s.Env); len(keys) > 0 {
			line += "  env=" + strings.Join(keys, ",") + " (values hidden)"
		}
		if _, err := fmt.Fprintln(output, gateway.Text(line)); err != nil {
			return err
		}
	}
	return nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
