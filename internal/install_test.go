package mino

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func installerFixture(t *testing.T) (script, home, assets string) {
	t.Helper()
	script, err := filepath.Abs(filepath.Join("..", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	home = filepath.Join(t.TempDir(), "user home")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	assets = t.TempDir()
	commands := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("ZDOTDIR", "")
	t.Setenv("PATH", commands+":/usr/bin:/bin:/usr/sbin:/sbin")
	t.Setenv("MINO_TEST_ASSETS", assets)
	t.Setenv("MINO_TEST_ARCH", "arm64")
	t.Setenv("MINO_TEST_OS", "Darwin")
	t.Setenv("MINO_TEST_DOWNLOAD_FAIL", "")
	t.Setenv("MINO_TEST_LATEST_FAIL", "")
	t.Setenv("MINO_TEST_LATEST_URL", "https://github.com/qshine/mino/releases/tag/v0.1.0")
	t.Setenv("MINO_TEST_MISSING_ASSET", "")
	mocks := map[string]string{
		"uname":   "#!/bin/bash\nif [[ $1 == -s ]]; then echo \"$MINO_TEST_OS\"; else echo \"$MINO_TEST_ARCH\"; fi\n",
		"sw_vers": "#!/bin/bash\necho 13.0\n",
		"gh":      "#!/bin/bash\necho 'Installation must not require GitHub CLI or login' >&2\nexit 1\n",
		"curl": `#!/bin/bash
set -eu
output=''
url=''
while [[ $# -gt 0 ]]; do
  case $1 in
    --proto|--tlsv1.2|--fail|--silent|--show-error|--location|--head)
      if [[ $1 == --proto ]]; then shift; [[ $1 == '=https' ]] || exit 2; fi
      shift ;;
    --output) output=$2; shift 2 ;;
    --write-out) [[ $2 == '%{url_effective}' ]] || exit 2; shift 2 ;;
    https://github.com/qshine/mino/releases/*) url=$1; shift ;;
    *) exit 2 ;;
  esac
done
if [[ $url == https://github.com/qshine/mino/releases/latest ]]; then
  [[ -z $MINO_TEST_LATEST_FAIL ]] || exit 22
  printf '%s' "$MINO_TEST_LATEST_URL"
else
  [[ $url == https://github.com/qshine/mino/releases/download/v0.1.0/* ]] || exit 22
  [[ -z $MINO_TEST_DOWNLOAD_FAIL ]] || exit 22
  filename=${url##*/}
  if [[ $filename == checksums.txt && -n $MINO_TEST_MISSING_ASSET ]]; then exit 22; fi
  cp "$MINO_TEST_ASSETS/$filename" "$output"
fi
`,
	}
	for name, content := range mocks {
		if err := os.WriteFile(filepath.Join(commands, name), []byte(content), 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeInstallerAssets(t, assets, "0.1.0")
	return script, home, assets
}

func writeInstallerAssets(t *testing.T, directory, binaryVersion string) {
	t.Helper()
	var checksums strings.Builder
	for _, arch := range []string{"arm64", "amd64"} {
		var buffer bytes.Buffer
		gz := gzip.NewWriter(&buffer)
		tarWriter := tar.NewWriter(gz)
		body := []byte("#!/bin/bash\nprintf 'mino " + binaryVersion + "\\n'\n")
		if err := tarWriter.WriteHeader(&tar.Header{Name: "mino", Mode: 0755, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(body); err != nil {
			t.Fatal(err)
		}
		if err := tarWriter.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		asset := "mino_0.1.0_darwin_" + arch + ".tar.gz"
		if err := os.WriteFile(filepath.Join(directory, asset), buffer.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&checksums, "%x  %s\n", sha256.Sum256(buffer.Bytes()), asset)
	}
	if err := os.WriteFile(filepath.Join(directory, "checksums.txt"), []byte(checksums.String()), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestInstallerInstallsBothArchitecturesAndPreservesSettings(t *testing.T) {
	for _, arch := range []string{"arm64", "x86_64"} {
		t.Run(arch, func(t *testing.T) {
			script, home, _ := installerFixture(t)
			t.Setenv("MINO_TEST_ARCH", arch)
			configPath := filepath.Join(home, ".mino", "config.json")
			writeTestConfig(t, configPath, `{"api_key":"keep-private-settings"}`)
			for range 2 {
				output, err := exec.Command("/bin/bash", script).CombinedOutput()
				if err != nil {
					t.Fatalf("installer: %v\n%s", err, output)
				}
				if strings.Contains(string(output), "keep-private-settings") {
					t.Fatal("installer exposed settings")
				}
			}
			output, err := exec.Command(filepath.Join(home, ".mino", "bin", "mino"), "version").Output()
			if err != nil || string(output) != "mino 0.1.0\n" {
				t.Fatalf("installed version = %q, error = %v", output, err)
			}
			data, err := os.ReadFile(configPath)
			if err != nil || string(data) != `{"api_key":"keep-private-settings"}` {
				t.Fatal("installer modified configuration")
			}
			profile, err := os.ReadFile(filepath.Join(home, ".zshrc"))
			if err != nil || strings.Count(string(profile), `export PATH="$HOME/.mino/bin:$PATH"`) != 1 {
				t.Fatal("installer did not add the shell path exactly once")
			}
			info, err := os.Stat(filepath.Join(home, ".mino"))
			if err != nil || info.Mode().Perm() != 0700 {
				t.Fatal("user directory permissions are not private")
			}
		})
	}
}

func TestInstallerFailureKeepsExistingBinary(t *testing.T) {
	for _, failure := range []string{"checksum", "download", "latest lookup", "invalid latest URL", "missing asset", "version mismatch", "unsupported OS", "invalid version"} {
		t.Run(failure, func(t *testing.T) {
			script, home, assets := installerFixture(t)
			binDir := filepath.Join(home, ".mino", "bin")
			if err := os.MkdirAll(binDir, 0700); err != nil {
				t.Fatal(err)
			}
			binary := filepath.Join(binDir, "mino")
			if err := os.WriteFile(binary, []byte("previous executable"), 0700); err != nil {
				t.Fatal(err)
			}
			args := []string{script}
			switch failure {
			case "checksum":
				if err := os.WriteFile(filepath.Join(assets, "checksums.txt"), []byte("invalid checksum"), 0600); err != nil {
					t.Fatal(err)
				}
			case "download":
				t.Setenv("MINO_TEST_DOWNLOAD_FAIL", "1")
			case "latest lookup":
				t.Setenv("MINO_TEST_LATEST_FAIL", "1")
			case "invalid latest URL":
				t.Setenv("MINO_TEST_LATEST_URL", "https://github.com/qshine/mino/releases")
			case "missing asset":
				t.Setenv("MINO_TEST_MISSING_ASSET", "1")
			case "version mismatch":
				writeInstallerAssets(t, assets, "0.0.1")
			case "unsupported OS":
				t.Setenv("MINO_TEST_OS", "Linux")
			case "invalid version":
				args = append(args, "../../bad")
			}
			if output, err := exec.Command("/bin/bash", args...).CombinedOutput(); err == nil {
				t.Fatalf("installer accepted %s: %s", failure, output)
			}
			data, err := os.ReadFile(binary)
			if err != nil || string(data) != "previous executable" {
				t.Fatal("failed update replaced the old executable")
			}
			entries, err := os.ReadDir(binDir)
			if err != nil || len(entries) != 1 {
				t.Fatal("failed update left a partial executable")
			}
		})
	}
}

func TestInstallerRejectsSymlinkDirectory(t *testing.T) {
	script, home, _ := installerFixture(t)
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(home, ".mino")); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("/bin/bash", script).CombinedOutput(); err == nil {
		t.Fatalf("accepted symlink directory: %s", output)
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 0 {
		t.Fatal("installer wrote through a directory symlink")
	}
}

func TestFirstInstallationPreparesHomeWithoutInventingSettings(t *testing.T) {
	script, home, _ := installerFixture(t)
	t.Setenv("SHELL", "/bin/bash")
	output, err := exec.Command("/bin/bash", script, "0.1.0").CombinedOutput()
	if err != nil {
		t.Fatalf("first installation: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(home, ".mino", "config.json")); !os.IsNotExist(err) {
		t.Fatal("installer created configuration before collecting user settings")
	}
	if profile, err := os.ReadFile(filepath.Join(home, ".bash_profile")); err != nil || !strings.Contains(string(profile), `export PATH="$HOME/.mino/bin:$PATH"`) {
		t.Fatal("Bash profile was not configured")
	}
}

// A pinned version must remain installable even when latest-release discovery fails.
func TestInstallerPinnedVersionDoesNotRequireLatestLookup(t *testing.T) {
	script, home, _ := installerFixture(t)
	t.Setenv("MINO_TEST_LATEST_FAIL", "1")
	output, err := exec.Command("/bin/bash", script, "v0.1.0").CombinedOutput()
	if err != nil {
		t.Fatalf("pinned installation: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(home, ".mino", "bin", "mino")); err != nil {
		t.Fatal(err)
	}
}
