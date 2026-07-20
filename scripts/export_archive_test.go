package scripts

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportArchiveValidTreeIsReproducible(t *testing.T) {
	fixture := newArchiveExportFixture(t)
	fixture.writeFile("README.md", "fixture\n")
	fixture.writeFile("internal/example/example.go", "package example\n")
	fixture.commit("valid tree")

	first := filepath.Join(fixture.root, ".artifacts", "first.zip")
	second := filepath.Join(fixture.root, ".artifacts", "second.zip")

	fixture.requireExportSuccess("-OutputPath", ".artifacts/first.zip", "-Force")
	fixture.requireExportSuccess("-OutputPath", ".artifacts/first.zip", "-Force")
	fixture.requireExportSuccess("-OutputPath", ".artifacts/second.zip", "-Force")

	firstHash := sha256File(t, first)
	secondHash := sha256File(t, second)
	if firstHash != secondHash {
		t.Fatalf("same treeish exports are not reproducible: %s != %s", firstHash, secondHash)
	}
}

func TestExportArchiveRejectsBlockedTrackedPaths(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		includeArtifacts bool
	}{
		{name: "idea directory", path: ".idea/workspace.xml"},
		{name: "vscode directory", path: ".vscode/settings.json"},
		{name: "artifacts directory", path: ".artifacts/local.txt", includeArtifacts: true},
		{name: "certs directory", path: "certs/localhost.crt"},
		{name: "bin directory", path: "bin/pet-study"},
		{name: "dist directory", path: "dist/pet-study.zip"},
		{name: "tmp directory", path: "tmp/scratch.txt"},
		{name: "curl header spill file", path: "-H"},
		{name: "request json", path: "req.json"},
		{name: "named request json", path: "req-create-user.json"},
		{name: "dotenv", path: ".env"},
		{name: "dotenv suffix", path: ".env.local"},
		{name: "macos scratch", path: ".DS_Store"},
		{name: "windows scratch", path: "Thumbs.db"},
		{name: "log file", path: "server.log"},
		{name: "local file", path: "settings.local"},
		{name: "pem file", path: "certs-public/localhost.pem"},
		{name: "key file", path: "secrets/localhost.key"},
		{name: "ppk file", path: "secret.ppk"},
		{name: "p12 file", path: "secret.p12"},
		{name: "pfx file", path: "secret.pfx"},
		{name: "intellij module", path: "project.iml"},
		{name: "vim swap", path: "notes.swp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newArchiveExportFixture(t)
			fixture.writeFile("README.md", "fixture\n")
			fixture.writeFile(tt.path, "blocked\n")
			fixture.commitWithOptions("blocked path", tt.includeArtifacts)

			outputPath := filepath.Join(fixture.root, "release.zip")
			output, err := fixture.runExport("-OutputPath", "release.zip", "-Force")
			if err == nil {
				t.Fatalf("export succeeded with blocked path %s", tt.path)
			}
			if !strings.Contains(output, filepath.ToSlash(tt.path)) {
				t.Fatalf("export failure did not mention blocked path %s:\n%s", tt.path, output)
			}
			assertFileMissing(t, outputPath)
			fixture.assertNoTempArchives()
		})
	}
}

func TestExportArchiveRejectsPrivateKeyMarkersWithoutPublishing(t *testing.T) {
	for _, tt := range []struct {
		name    string
		content string
	}{
		{
			name:    "small marker",
			content: privateKeyMarker("RSA") + "\nredacted\n",
		},
		{
			name:    "large marker",
			content: strings.Repeat("A", 1<<20+64) + "\n" + privateKeyMarker("OPENSSH") + "\nredacted\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newArchiveExportFixture(t)
			fixture.writeFile("README.md", "fixture\n")
			fixture.writeFile("docs/secret.txt", tt.content)
			fixture.commit("private key marker")

			outputPath := filepath.Join(fixture.root, ".artifacts", "unsafe.zip")
			output, err := fixture.runExport("-OutputPath", ".artifacts/unsafe.zip", "-Force")
			if err == nil {
				t.Fatalf("export succeeded with private-key marker")
			}
			if !strings.Contains(output, "private key marker") {
				t.Fatalf("export failure did not mention private key marker:\n%s", output)
			}
			assertFileMissing(t, outputPath)
			fixture.assertNoTempArchives()
		})
	}
}

func TestExportArchiveFailedForcePreservesExistingOutput(t *testing.T) {
	fixture := newArchiveExportFixture(t)
	fixture.writeFile("README.md", "fixture\n")
	fixture.commit("valid tree")

	outputPath := filepath.Join(fixture.root, ".artifacts", "release.zip")
	fixture.requireExportSuccess("-OutputPath", ".artifacts/release.zip", "-Force")
	before := sha256File(t, outputPath)

	fixture.writeFile("docs/secret.txt", privateKeyMarker("")+"\nredacted\n")
	fixture.commit("invalid tree")

	output, err := fixture.runExport("-OutputPath", ".artifacts/release.zip", "-Force")
	if err == nil {
		t.Fatalf("export succeeded with private-key marker")
	}
	if !strings.Contains(output, "private key marker") {
		t.Fatalf("export failure did not mention private key marker:\n%s", output)
	}

	after := sha256File(t, outputPath)
	if before != after {
		t.Fatalf("failed -Force export replaced existing archive: %s != %s", before, after)
	}
	fixture.assertNoTempArchives()
}

type archiveExportFixture struct {
	t    *testing.T
	root string
}

func newArchiveExportFixture(t *testing.T) *archiveExportFixture {
	t.Helper()

	if _, err := exec.LookPath("pwsh"); err != nil {
		t.Skip("pwsh is not available")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	root := t.TempDir()
	fixture := &archiveExportFixture{t: t, root: root}
	fixture.copyScript("common.ps1")
	fixture.copyScript("export-archive.ps1")
	fixture.runCommand("git", "init")
	return fixture
}

func (f *archiveExportFixture) copyScript(name string) {
	f.t.Helper()

	content, err := os.ReadFile(name)
	if err != nil {
		f.t.Fatal(err)
	}
	target := filepath.Join(f.root, "scripts", name)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *archiveExportFixture) writeFile(relativePath string, content string) {
	f.t.Helper()

	path := filepath.Join(f.root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *archiveExportFixture) commit(message string) {
	f.t.Helper()

	f.commitWithOptions(message, false)
}

func (f *archiveExportFixture) commitWithOptions(message string, includeArtifacts bool) {
	f.t.Helper()

	f.runCommand("git", "add", "-A", "-f")
	if !includeArtifacts {
		f.runCommand("git", "rm", "--cached", "-r", "--ignore-unmatch", ".artifacts")
	}
	f.runCommand("git", "-c", "user.email=codex@example.invalid", "-c", "user.name=Codex", "commit", "--quiet", "-m", message)
}

func (f *archiveExportFixture) requireExportSuccess(args ...string) string {
	f.t.Helper()

	output, err := f.runExport(args...)
	if err != nil {
		f.t.Fatalf("export failed: %v\n%s", err, output)
	}
	return output
}

func (f *archiveExportFixture) runExport(args ...string) (string, error) {
	f.t.Helper()

	commandArgs := append([]string{"-NoProfile", "-File", filepath.Join(f.root, "scripts", "export-archive.ps1")}, args...)
	cmd := exec.Command("pwsh", commandArgs...)
	cmd.Dir = f.root
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (f *archiveExportFixture) runCommand(name string, args ...string) {
	f.t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = f.root
	if output, err := cmd.CombinedOutput(); err != nil {
		f.t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, string(output))
	}
}

func (f *archiveExportFixture) assertNoTempArchives() {
	f.t.Helper()

	matches, err := filepath.Glob(filepath.Join(f.root, ".artifacts", "*.tmp-*"))
	if err != nil {
		f.t.Fatal(err)
	}
	if len(matches) > 0 {
		f.t.Fatalf("temporary archives were not removed: %v", matches)
	}
}

func sha256File(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func assertFileMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, stat err=%v", path, err)
	}
}

func privateKeyMarker(kind string) string {
	if kind == "" {
		return "-----BEGIN " + "PRIVATE KEY-----"
	}
	return "-----BEGIN " + kind + " PRIVATE KEY-----"
}
