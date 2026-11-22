// Command check-identity fails when a tracked file names the upstream
// project this fork was renamed from. A leftover name is either a surface
// two installs would share (a socket, a directory, a variable) or text that
// sends a user to the wrong project.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// forbidden are the upstream names: its binary and module, its spellings in
// code, its tmux socket, its environment prefix, and its owners.
var forbidden = []string{
	"agent-manager",
	"agent_manager",
	"AgentManager",
	"Agent Manager",
	"agentmgr",
	"AGENT_MANAGER_",
	"wainmann",
	"YoanWai",
}

// exemptPaths are files, or directories when they end in a slash, left
// unchecked as a whole. Like every list here, it only shrinks.
var exemptPaths = []string{
	// The licence and the attribution Apache-2.0 §4 asks to keep.
	"LICENSE",
	"NOTICE",
	// This tool holds the forbidden names and its test fixtures.
	"tools/architecture/check-identity/",
	// CLI screens recorded as input to the status rules; editing one
	// changes what the test checks.
	"internal/status/testdata/",
	// The design describes the state before the rename.
	"docs/designs/upstream-detach.md",
}

// exemptText is the exact text a file may hold. Every other forbidden name
// in that file still fails.
var exemptText = map[string][]string{
	"README.md": {"aghelm is a modified version of [agent-manager](https://github.com/YoanWai/agent-manager)."},
	// The prefix of the upstream variables main strips at startup.
	"main.go": {`"AGENT_MANAGER_"`},
	// Recorded CLI screens, as in internal/status/testdata.
	"internal/status/status_copilot_test.go": {`agent-manager rename "<name>"`},
	"internal/status/status_test.go":         {"• agent-manager "},
	"internal/status/transcript_test.go":     {"~/d/agent-manager-worktrees/grok-test"},
}

func main() {
	files, err := trackedFiles(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "git ls-files:", err)
		os.Exit(1)
	}
	found, err := check(".", files)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(found) > 0 {
		fmt.Fprintf(os.Stderr, "upstream names outside the exception list in tools/architecture/check-identity:\n  %s\n", strings.Join(found, "\n  "))
		os.Exit(1)
	}
	fmt.Printf("PASS: no tracked file names the upstream project\n")
}

// trackedFiles lists the paths git tracks under root, relative to it.
func trackedFiles(root string) ([]string, error) {
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, name := range strings.Split(string(out), "\x00") {
		if name != "" {
			files = append(files, name)
		}
	}
	return files, nil
}

// check returns "path:line: text" for each line of files under root that
// holds a forbidden name, and "path: path" for a path that does. A file
// listed but missing from the work tree, or a binary one, is skipped.
func check(root string, files []string) ([]string, error) {
	var found []string
	for _, name := range files {
		if exempt(name) {
			continue
		}
		if hasForbidden(name) {
			found = append(found, name+": path")
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if binary(data) {
			continue
		}
		for i, line := range strings.Split(string(data), "\n") {
			text := line
			for _, allowed := range exemptText[name] {
				text = strings.ReplaceAll(text, allowed, "")
			}
			if hasForbidden(text) {
				found = append(found, fmt.Sprintf("%s:%d: %s", name, i+1, strings.TrimSpace(line)))
			}
		}
	}
	return found, nil
}

func exempt(name string) bool {
	for _, path := range exemptPaths {
		if name == path || strings.HasSuffix(path, "/") && strings.HasPrefix(name, path) {
			return true
		}
	}
	return false
}

func hasForbidden(text string) bool {
	for _, name := range forbidden {
		if strings.Contains(text, name) {
			return true
		}
	}
	return false
}

// binary reports a NUL in the first 8000 bytes, the test git itself uses.
func binary(data []byte) bool {
	return bytes.IndexByte(data[:min(len(data), 8000)], 0) >= 0
}
