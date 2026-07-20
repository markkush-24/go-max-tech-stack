package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanCheckoutGateFailsWhenToolsModuleDrifts(t *testing.T) {
	if _, err := exec.LookPath("pwsh"); err != nil {
		t.Skip("pwsh is not available")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/checkoutfixture\n\ngo 1.25.0\n")
	writeFile(t, root, "cmd/api/main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, root, "internal/placeholder/placeholder.go", "package placeholder\n")
	writeFile(t, root, "tools/go.mod", strings.Join([]string{
		"module example.com/checkoutfixture/tools",
		"",
		"go 1.25.0",
		"",
		"require example.com/unused v0.0.0",
		"",
		"replace example.com/unused => ./unused",
		"",
	}, "\n"))
	writeFile(t, root, "tools/unused/go.mod", "module example.com/unused\n\ngo 1.25.0\n")
	writeFile(t, root, "tools/unused/unused.go", "package unused\n")
	copyScriptFile(t, "common.ps1", filepath.Join(root, "scripts", "common.ps1"))
	copyScriptFile(t, "check-clean-checkout.ps1", filepath.Join(root, "scripts", "check-clean-checkout.ps1"))

	runCommand(t, root, "git", "init")
	runCommand(t, root, "git", "add", ".")

	cmd := exec.Command("pwsh", "-NoProfile", "-File", filepath.Join(root, "scripts", "check-clean-checkout.ps1"))
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("check-clean-checkout.ps1 succeeded; want failure from tools/go.mod drift")
	}

	out := string(output)
	if !strings.Contains(out, "tools/go.mod") && !strings.Contains(out, `tools\go.mod`) {
		t.Fatalf("check-clean-checkout.ps1 failed without reporting tools/go.mod drift:\n%s", out)
	}
}

func writeFile(t *testing.T, root string, relativePath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyScriptFile(t *testing.T, sourceName string, target string) {
	t.Helper()

	content, err := os.ReadFile(sourceName)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runCommand(t *testing.T, dir string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, string(output))
	}
}
