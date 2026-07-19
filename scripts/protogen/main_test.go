package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPinnedGeneratorVersions(t *testing.T) {
	for name, version := range map[string]string{
		"protoc":             requiredProtocVersion,
		"protoc-gen-go":      requiredProtocGenGoVersion,
		"protoc-gen-go-grpc": requiredProtocGenGRPCVersion,
	} {
		if version == "" {
			t.Fatalf("%s version is empty", name)
		}
		if strings.Contains(version, "latest") {
			t.Fatalf("%s version is not pinned: %q", name, version)
		}
	}
}

func TestProtoFilesAreCanonicalAndDeterministic(t *testing.T) {
	want := []string{
		"internal/transport/pb/user.proto",
		"internal/transport/pb/job.proto",
	}

	if !slices.Equal(protoFiles, want) {
		t.Fatalf("protoFiles=%v want %v", protoFiles, want)
	}
}

func TestFindRepoRoot(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}

	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "module pet-study") {
		t.Fatalf("resolved root %s does not contain pet-study go.mod", root)
	}
}
