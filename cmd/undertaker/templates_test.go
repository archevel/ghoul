package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderMainWithPrelude(t *testing.T) {
	var buf bytes.Buffer
	err := renderMain(&buf, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "package main") {
		t.Error("expected 'package main'")
	}
	if !strings.Contains(out, `"github.com/archevel/ghoul"`) {
		t.Error("expected ghoul import")
	}
	if !strings.Contains(out, "go:embed prelude.ghl") {
		t.Error("expected go:embed directive for prelude")
	}
	if !strings.Contains(out, "preludeSource") {
		t.Error("expected preludeSource variable")
	}
	if !strings.Contains(out, "NewBare") {
		t.Error("expected NewBare call (prelude loaded manually)")
	}
	if !strings.Contains(out, "Process(") {
		t.Error("expected Process call to load prelude")
	}
}

func TestRenderMainWithoutPrelude(t *testing.T) {
	var buf bytes.Buffer
	err := renderMain(&buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "package main") {
		t.Error("expected 'package main'")
	}
	if strings.Contains(out, "go:embed") {
		t.Error("should NOT contain go:embed when prelude disabled")
	}
	if strings.Contains(out, "preludeSource") {
		t.Error("should NOT contain preludeSource when prelude disabled")
	}
	if !strings.Contains(out, "NewBare") {
		t.Error("expected NewBare call")
	}
}

func TestRenderSarcophagus(t *testing.T) {
	mummyNames := []string{"math", "strings", "github.com_foo_bar"}

	var buf bytes.Buffer
	err := renderSarcophagus(&buf, "mymodule", mummyNames)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "package main") {
		t.Error("expected 'package main'")
	}
	if !strings.Contains(out, `_ "mymodule/mummies/math_mummy"`) {
		t.Errorf("expected math_mummy import, got:\n%s", out)
	}
	if !strings.Contains(out, `_ "mymodule/mummies/strings_mummy"`) {
		t.Errorf("expected strings_mummy import, got:\n%s", out)
	}
	if !strings.Contains(out, `_ "mymodule/mummies/github.com_foo_bar_mummy"`) {
		t.Errorf("expected github.com_foo_bar_mummy import, got:\n%s", out)
	}
}

func TestRenderSarcophagusEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := renderSarcophagus(&buf, "mymodule", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "package main") {
		t.Error("expected 'package main'")
	}
	// Should still be valid Go even with no imports
	if strings.Contains(out, "import") {
		t.Error("should NOT contain import block when no mummies")
	}
}
