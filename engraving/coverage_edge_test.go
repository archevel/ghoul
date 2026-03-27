package engraving

import (
	"bytes"
	"strings"
	"testing"
)

// --- New() backward-compat constructor ---

func TestNewWithDebugWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, nil)
	logger.Debug("hello debug")
	if buf.Len() == 0 {
		t.Error("expected debug output when debug writer is provided")
	}
}

func TestNewWithWarnWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := New(nil, &buf)
	logger.Warn("hello warn")
	if buf.Len() == 0 {
		t.Error("expected warn output when warn writer is provided")
	}
}

func TestNewWithWarnWriterDebugSuppressed(t *testing.T) {
	var buf bytes.Buffer
	logger := New(nil, &buf)
	logger.Debug("should not appear")
	if buf.Len() != 0 {
		t.Error("debug output should be suppressed when only warn writer is given")
	}
}

func TestNewWithBothNilDisabled(t *testing.T) {
	logger := New(nil, nil)
	// Should not panic
	logger.Trace("trace")
	logger.Debug("debug")
	logger.Warn("warn")
}

// --- log with args ---

type mockRepr struct{ val string }

func (m mockRepr) Repr() string { return m.val }

func TestLogWithReprArgs(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter(&buf, LevelTrace)
	logger.Trace("test", mockRepr{"hello"})
	output := buf.String()
	if !strings.Contains(output, "hello") {
		t.Errorf("expected Repr() value in output, got: %s", output)
	}
}

func TestLogWithNilArg(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter(&buf, LevelTrace)
	logger.Trace("test", nil)
	output := buf.String()
	if !strings.Contains(output, "<nil>") {
		t.Errorf("expected <nil> in output, got: %s", output)
	}
}

func TestLogWithMixedArgs(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter(&buf, LevelTrace)
	logger.Trace("test", 42, "plain", nil, mockRepr{"repr-val"})
	output := buf.String()
	if !strings.Contains(output, "42") {
		t.Errorf("expected '42' in output, got: %s", output)
	}
	if !strings.Contains(output, "repr-val") {
		t.Errorf("expected 'repr-val' in output, got: %s", output)
	}
	if !strings.Contains(output, "<nil>") {
		t.Errorf("expected '<nil>' in output, got: %s", output)
	}
}

func TestLogNoArgs(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter(&buf, LevelTrace)
	logger.Trace("simple message")
	output := buf.String()
	if !strings.Contains(output, "simple message") {
		t.Errorf("expected message in output, got: %s", output)
	}
}

func TestNewWithNilWriterDisablesOutput(t *testing.T) {
	logger := NewWithWriter(nil, LevelTrace)
	// Should not panic
	logger.Trace("test")
	logger.Debug("test")
	logger.Warn("test")
}
