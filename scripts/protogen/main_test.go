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

func TestNormalizedVersionTokenComparison(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
		match  bool
	}{
		{
			name:   "exact protoc-gen-go token",
			output: "protoc-gen-go.exe v1.36.11",
			want:   "v1.36.11",
			match:  true,
		},
		{
			name:   "exact grpc token",
			output: "protoc-gen-go-grpc 1.6.1",
			want:   "1.6.1",
			match:  true,
		},
		{
			name:   "wrapped staticcheck module token",
			output: "staticcheck.exe 2026.1 (v0.7.0)",
			want:   "v0.7.0",
			match:  true,
		},
		{
			name:   "suffix near match is rejected",
			output: "protoc-gen-go v1.36.110",
			want:   "v1.36.11",
			match:  false,
		},
		{
			name:   "grpc suffix near match is rejected",
			output: "protoc-gen-go-grpc 1.6.10",
			want:   "1.6.1",
			match:  false,
		},
		{
			name:   "prefix near match is rejected",
			output: "protoc-gen-go x-v1.36.11",
			want:   "v1.36.11",
			match:  false,
		},
		{
			name:   "embedded token is rejected",
			output: "protoc-gen-go version=v1.36.11",
			want:   "v1.36.11",
			match:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNormalizedVersionToken(tt.output, tt.want); got != tt.match {
				t.Fatalf("hasNormalizedVersionToken(%q, %q)=%v want %v", tt.output, tt.want, got, tt.match)
			}
		})
	}
}
