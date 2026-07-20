package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	requiredProtocVersion        = "libprotoc 34.0"
	requiredProtocGenGoVersion   = "v1.36.11"
	requiredProtocGenGRPCVersion = "1.6.1"
)

var protoFiles = []string{
	"internal/transport/pb/user.proto",
	"internal/transport/pb/job.proto",
}

func main() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fail(err)
	}

	if err := requireExactVersion("protoc", []string{"--version"}, requiredProtocVersion); err != nil {
		fail(err)
	}
	if err := requireVersionToken("protoc-gen-go", []string{"--version"}, requiredProtocGenGoVersion); err != nil {
		fail(err)
	}
	if err := requireVersionToken("protoc-gen-go-grpc", []string{"--version"}, requiredProtocGenGRPCVersion); err != nil {
		fail(err)
	}

	args := []string{
		"-I", ".",
		"--go_out=.",
		"--go_opt=paths=source_relative",
		"--go-grpc_out=.",
		"--go-grpc_opt=paths=source_relative",
	}
	args = append(args, protoFiles...)

	cmd := exec.Command("protoc", args...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fail(fmt.Errorf("run protoc: %w", err))
	}
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root from %s", wd)
		}
	}
}

func requireExactVersion(command string, args []string, want string) error {
	got, err := commandOutput(command, args...)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%s version %q, want %q", command, got, want)
	}
	return nil
}

func requireVersionToken(command string, args []string, want string) error {
	got, err := commandOutput(command, args...)
	if err != nil {
		return err
	}
	if !hasNormalizedVersionToken(got, want) {
		return fmt.Errorf("%s version %q, want exact token %q", command, got, want)
	}
	return nil
}

func hasNormalizedVersionToken(output string, want string) bool {
	want = normalizeVersionToken(want)
	for _, token := range strings.Fields(output) {
		if normalizeVersionToken(token) == want {
			return true
		}
	}
	return false
}

func normalizeVersionToken(token string) string {
	return strings.Trim(token, " \t\r\n\"'`()[]{}<>,;")
}

func commandOutput(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(stdout.String() + stderr.String())
		if output == "" {
			return "", fmt.Errorf("run %s: %w", command, err)
		}
		return "", fmt.Errorf("run %s: %w: %s", command, err, output)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
