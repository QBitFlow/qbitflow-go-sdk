package qbitflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestSnippets is part of the maintainers' conformance suite: it checks that the examples'
// docs:start / docs:end regions, examples/snippets.manifest.json and the README's embedded
// blocks agree with the website's snippet catalog. It needs the maintainers' workspace
// (QBITFLOW_HUB_DIR, else the directory two levels up) and node; outside it, it is skipped.
func TestSnippets(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	isWorkspace := func(dir string) bool {
		for _, f := range []string{"scripts/embed-snippets.mjs", "snippets/catalog.json"} {
			if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
				return false
			}
		}
		return true
	}

	dir := os.Getenv("QBITFLOW_HUB_DIR")
	switch {
	case dir != "" && !isWorkspace(dir):
		t.Fatalf("QBITFLOW_HUB_DIR=%s has no scripts/embed-snippets.mjs and snippets/catalog.json", dir)
	case dir == "":
		dir = filepath.Join(root, "..", "..")
		if !isWorkspace(dir) {
			t.Skip("outside the maintainers' workspace (set QBITFLOW_HUB_DIR): snippets not checked")
		}
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed: snippets not checked")
	}

	out, err := exec.Command(node, filepath.Join(dir, "scripts", "embed-snippets.mjs"), "--check", root).CombinedOutput()
	if err != nil {
		t.Fatalf("embed-snippets --check failed (%v):\n%s", err, out)
	}
}
