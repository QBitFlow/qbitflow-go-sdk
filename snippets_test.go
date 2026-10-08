package qbitflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestSnippets runs the hub's embed-snippets --check on this SDK: the examples' docs:start /
// docs:end regions, examples/snippets.manifest.json and the README's embedded blocks must agree
// with the hub's snippets/catalog.json. The hub is QBITFLOW_HUB_DIR, else the checkout this SDK
// sits in (../..); without it, or without node, the test is skipped.
func TestSnippets(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	isHub := func(dir string) bool {
		for _, f := range []string{"scripts/embed-snippets.mjs", "snippets/catalog.json"} {
			if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
				return false
			}
		}
		return true
	}

	hub := os.Getenv("QBITFLOW_HUB_DIR")
	switch {
	case hub != "" && !isHub(hub):
		t.Fatalf("QBITFLOW_HUB_DIR=%s has no scripts/embed-snippets.mjs and snippets/catalog.json", hub)
	case hub == "":
		hub = filepath.Join(root, "..", "..")
		if !isHub(hub) {
			t.Skip("the QBitFlow hub is not found (set QBITFLOW_HUB_DIR): snippets not checked")
		}
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed: snippets not checked")
	}

	out, err := exec.Command(node, filepath.Join(hub, "scripts", "embed-snippets.mjs"), "--check", root).CombinedOutput()
	if err != nil {
		t.Fatalf("embed-snippets --check failed (%v):\n%s", err, out)
	}
}
