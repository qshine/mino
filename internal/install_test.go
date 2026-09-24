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
	t.Setenv("MINO_TEST_EMPTY_TAG_ASSETS", "")
	t.Setenv("MINO_TEST_ASSET_LIST_FAIL", "")
	t.Setenv("MINO_TEST_MISSING_ASSET", "")
	mocks := map[string]string{
		"uname":   "#!/bin/bash\nif [[ $1 == -s ]]; then echo \"$MINO_TEST_OS\"; else echo \"$MINO_TEST_ARCH\"; fi\n",
		"sw_vers": "#!/bin/bash\necho 13.0\n",
		"gh": `#!/bin/bash
set -eu
if [[ $1 == api ]]; then
  case "$*" in
    *releases/latest*) echo v0.1.0 ;;
    *releases/tags/v0.1.0*) echo 101 ;;
    *releases/101/assets*)
      [[ -z $MINO_TEST_ASSET_LIST_FAIL ]] || exit 1
      printf '201\tmino_0.1.0_darwin_arm64.tar.gz\n202\tmino_0.1.0_darwin_amd64.tar.gz\n'
      if [[ -z $MINO_TEST_MISSING_ASSET ]]; then printf '203\tchecksums.txt\n'; fi
      ;;
    *releases/assets/*)
      [[ -z $MINO_TEST_DOWNLOAD_FAIL ]] || exit 1
      case "$*" in
        *releases/assets/201*) filename=mino_0.1.0_darwin_arm64.tar.gz ;;
        *releases/assets/202*) filename=mino_0.1.0_darwin_amd64.tar.gz ;;
        *releases/assets/203*) filename=checksums.txt ;;
        *) exit 2 ;;
      esac
      cat "$MINO_TEST_ASSETS/$filename"
      ;;
    *) exit 2 ;;
  esac
  exit 0
fi
[[ $1 == release && $2 == download ]] || exit 2
[[ -z $MINO_TEST_EMPTY_TAG_ASSETS ]] || { echo "no assets to download" >&2; exit 1; }
[[ -z $MINO_TEST_DOWNLOAD_FAIL ]] || exit 1
shift 3
assets=()
while [[ $# -gt 0 ]]; do
  case $1 in
    --repo) shift 2 ;;
    --pattern) assets+=("$2"); shift 2 ;;
    --dir) destination=$2; shift 2 ;;
    *) exit 2 ;;
  esac
done
for asset in "${assets[@]}"; do
  cp "$MINO_TEST_ASSETS/$asset" "$destination/"
done
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
	for _, failure := range []string{"checksum", "download", "asset list", "missing asset", "version mismatch", "unsupported OS", "invalid version"} {
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
			case "asset list":
				t.Setenv("MINO_TEST_ASSET_LIST_FAIL", "1")
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

// GitHub can return assets from the dedicated endpoint while the release-by-tag
// response has an empty embedded assets list. The download must still succeed.
func TestInstallerDownloadsWhenTagMetadataOmitsAssets(t *testing.T) {
	script, home, _ := installerFixture(t)
	t.Setenv("MINO_TEST_EMPTY_TAG_ASSETS", "1")
	output, err := exec.Command("/bin/bash", script, "v0.1.0").CombinedOutput()
	if err != nil {
		t.Fatalf("installation with incomplete tag metadata: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(home, ".mino", "bin", "mino")); err != nil {
		t.Fatal(err)
	}
}
