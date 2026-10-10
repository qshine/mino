package skills

import (
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
	c, err := Scan(root)
	if err != nil || len(c.Entries) != 0 {
		t.Fatalf("missing directory: %v, %v", c, err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".agents", "skills", "_support"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Scan(root); err != nil {
		t.Fatalf("directory without SKILL.md is not a skill: %v", err)
	}
	path := writeSkill(t, root, "review", "---\nname: review\ndescription: >-\n  Review code\n  carefully.\n---\nBODY_BEFORE")
	writeSkill(t, root, "alpha", "---\nname: alpha\ndescription: 'A: quoted description'\nlicense: MIT\n---\nALPHA_BODY")
	c, err = Scan(root)
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

func TestScanRejectsInvalidMetadata(t *testing.T) {
	for _, text := range []string{
		"no front matter", "---\nname: review\n---\nBody",
		"---\nname: wrong\ndescription: Description\n---\nBody",
		"---\nname: review\nname: review\ndescription: Description\n---\nBody",
		"---\nname: review\ndescription: [not, text]\n---\nBody",
		"---\nname: review\ndescription: " + strings.Repeat("a", 1025) + "\n---\nBody",
		"---\nname: review\ndescription: \xff\n---\nBody",
	} {
		t.Run(text[:min(len(text), 40)], func(t *testing.T) {
			root := t.TempDir()
			writeSkill(t, root, "review", text)
			if _, err := Scan(root); err == nil {
				t.Fatal("accepted invalid metadata")
			}
		})
	}
}

func TestSkillFilesStayInsideProjectAndBounded(t *testing.T) {
	root := t.TempDir()
	path := writeSkill(t, root, "review", "---\nname: review\ndescription: Review code\n---\nBody")
	c, err := Scan(root)
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
	if _, err := Scan(root); err == nil {
		t.Fatal("scanned file symlink")
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
	if _, err := Scan(root); err == nil {
		t.Fatal("scanned root symlink")
	}
}
