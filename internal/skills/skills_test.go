package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, root, name, text string) string {
	t.Helper()
	path := filepath.Join(root, ".agents", "skills", name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScanOnlyProjectMetadataAndViewOnDemand(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	writeSkill(t, home, "private", "---\nname: private\ndescription: Home only\n---\nPrivate body")
	c, _, err := Scan(root)
	if err != nil || len(c.Entries) != 0 {
		t.Fatalf("missing directory: %v, %v", c, err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".agents", "skills", "_support"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, problems, err := Scan(root); err != nil || len(problems) != 0 {
		t.Fatalf("directory without SKILL.md is not a skill: %q, %v", problems, err)
	}
	path := writeSkill(t, root, "review", "---\nname: review\ndescription: >-\n  Review code\n  carefully.\n---\nBODY_BEFORE")
	writeSkill(t, root, "alpha", "---\nname: alpha\ndescription: 'A: quoted description'\nlicense: MIT\n---\nALPHA_BODY")
	c, _, err = Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	prompt := c.Prompt()
	if len(c.Entries) != 2 || c.Entries[0].Name != "alpha" || !strings.Contains(prompt, "Review code carefully.") || strings.Contains(prompt, "BODY") || strings.Contains(prompt, "Home only") {
		t.Fatalf("catalog: %s", prompt)
	}
	if err := os.WriteFile(path, []byte("---\nname: review\ndescription: Changed\n---\nBODY_AFTER"), 0600); err != nil {
		t.Fatal(err)
	}
	if c.Prompt() != prompt {
		t.Fatal("catalog changed during session")
	}
	body, err := c.View("review")
	if err != nil || !strings.Contains(body, "BODY_AFTER") {
		t.Fatalf("lazy body: %q, %v", body, err)
	}
	for _, name := range []string{"../private", "private", "review/SKILL.md"} {
		if _, err := c.View(name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestScanSkipsInvalidSkillsAndNamesThem(t *testing.T) {
	for _, tc := range []struct{ text, reason string }{
		{"no front matter", "missing YAML front matter"},
		{"---\nname: review\n---\nBody", "YAML strings"},
		{"---\nname: wrong\ndescription: Description\n---\nBody", "must match the directory name"},
		{"---\nname: review\nname: review\ndescription: Description\n---\nBody", "invalid YAML"},
		{"---\nname: review\ndescription: [not, text]\n---\nBody", "YAML strings"},
		{"---\nname: review\ndescription: \" \"\n---\nBody", "must not be empty"},
		{"---\nname: review\ndescription: " + strings.Repeat("a", 1025) + "\n---\nBody", "1025 characters"},
		{"---\nname: review\ndescription: " + strings.Repeat("长", 1025) + "\n---\nBody", "1025 characters"},
		{"---\nname: review\ndescription: \xff\n---\nBody", "UTF-8"},
	} {
		t.Run(tc.text[:min(len(tc.text), 40)], func(t *testing.T) {
			root := t.TempDir()
			writeSkill(t, root, "alpha", "---\nname: alpha\ndescription: Still loads\n---\nBody")
			writeSkill(t, root, "review", tc.text)
			c, problems, err := Scan(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(c.Entries) != 1 || c.Entries[0].Name != "alpha" {
				t.Fatalf("entries = %+v", c.Entries)
			}
			if len(problems) != 1 || !strings.HasPrefix(problems[0], "review: ") || !strings.Contains(problems[0], tc.reason) {
				t.Fatalf("problems = %q, want review and %q", problems, tc.reason)
			}
		})
	}
}

func TestScanCountsDescriptionCharacters(t *testing.T) {
	root := t.TempDir()
	// 1,024 characters but 3,072 bytes: within the Agent Skills limit.
	writeSkill(t, root, "chinese", "---\nname: chinese\ndescription: "+strings.Repeat("长", 1024)+"\n---\nBody")
	c, problems, err := Scan(root)
	if err != nil || len(problems) != 0 || len(c.Entries) != 1 {
		t.Fatalf("entries=%+v problems=%q err=%v", c, problems, err)
	}
}

func TestScanReportsLinkedSkillsAndFullCatalog(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeSkill(t, outside, "linked", "---\nname: linked\ndescription: Outside\n---\nBody")
	if err := os.MkdirAll(filepath.Join(root, ".agents", "skills"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, ".agents", "skills", "linked"), filepath.Join(root, ".agents", "skills", "linked")); err != nil {
		t.Fatal(err)
	}
	// Plain files beside the skills are supporting material, not skills.
	if err := os.WriteFile(filepath.Join(root, ".agents", "skills", "README.md"), []byte("notes"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := range 40 {
		name := fmt.Sprintf("s%02d", i)
		writeSkill(t, root, name, "---\nname: "+name+"\ndescription: "+strings.Repeat("x", 1000)+"\n---\nBody")
	}
	c, problems, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Entries) == 0 || len(c.Entries) == 40 || c.Entries[0].Name != "s00" {
		t.Fatalf("%d entries", len(c.Entries))
	}
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, "linked: skipped because the skill directory is a symbolic link") || !strings.Contains(joined, "s39: skipped because the skill catalog is full") || strings.Contains(joined, "README") {
		t.Fatalf("problems = %q", problems)
	}
	if len(problems) != 1+40-len(c.Entries) {
		t.Fatalf("%d problems for %d entries", len(problems), len(c.Entries))
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSkillFilesStayInsideProjectAndBounded(t *testing.T) {
	root := t.TempDir()
	path := writeSkill(t, root, "review", "---\nname: review\ndescription: Review code\n---\nBody")
	c, _, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{strings.Repeat("x", 65537), "\xff", ""} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := c.View("review"); err == nil {
			t.Fatal("accepted invalid body")
		}
	}
	outside := filepath.Join(t.TempDir(), "private.md")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err := c.View("review"); err == nil {
		t.Fatal("followed file symlink")
	}
	if c, problems, err := Scan(root); err != nil || len(c.Entries) != 0 || len(problems) != 1 || !strings.HasPrefix(problems[0], "review: ") {
		t.Fatalf("scanned file symlink: %+v, %q, %v", c, problems, err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".agents")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(outside), filepath.Join(root, ".agents")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.View("review"); err == nil {
		t.Fatal("followed root symlink")
	}
	if _, _, err := Scan(root); err == nil {
		t.Fatal("scanned root symlink")
	}
}
