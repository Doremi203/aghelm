package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTheRepositoryNamesNoUpstream(t *testing.T) {
	files, err := trackedFiles("../../..")
	if err != nil {
		t.Skipf("git ls-files: %v", err)
	}
	found, err := check("../../..", files)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) > 0 {
		t.Fatalf("upstream names outside the exception list:\n%q", found)
	}
}

func TestCheck(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("doc.md", "line one\nrun agent-manager here\n")
	write("snake.py", "agent_manager = 1\n")
	write("comment.go", "package x\n\n// AgentManager was the old type.\n")
	write("import.go", "package x\n\nimport _ \"github.com/Doremi203/aghelm/internal/x\"\n")
	write("main.go", "const upstreamEnvPrefix = \"AGENT_MANAGER_\"\nvar v = \"AGENT_MANAGER_SESSION_ID\"\n")
	write("other.go", "const p = \"AGENT_MANAGER_\"\n")
	write("internal/status/testdata/screen.txt", "agentmgr\n")
	write("blob.bin", "agent-manager\x00")
	write("skills/agentmgr/x.md", "fine\n")

	files := []string{"doc.md", "snake.py", "comment.go", "import.go", "main.go", "other.go", "internal/status/testdata/screen.txt", "blob.bin", "skills/agentmgr/x.md", "gone.md"}
	found, err := check(dir, files)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"doc.md:2: run agent-manager here",
		"snake.py:1: agent_manager = 1",
		"comment.go:3: // AgentManager was the old type.",
		"main.go:2: var v = \"AGENT_MANAGER_SESSION_ID\"",
		"other.go:1: const p = \"AGENT_MANAGER_\"",
		"skills/agentmgr/x.md: path",
	}
	if !reflect.DeepEqual(found, want) {
		t.Fatalf("found = %q\nwant %q", found, want)
	}
}
