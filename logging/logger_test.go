package logging

import (
	"log/slog"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
)

type mockWriter struct {
	written *[]string
}

func (m mockWriter) Write(p []byte) (int, error) {
	*m.written = append(*m.written, string(p))
	return len(p), nil
}

func TestCanLogTraceMessages(t *testing.T) {
	m := mockWriter{&[]string{}}

	l := NewWithWriter(m, LevelTrace)
	l.Trace("hello")

	written := *m.written
	if len(written) != 1 {
		t.Fatalf("Logger wrote the wrong number of times expected %d, but got %d", 1, len(*m.written))
	}

	if !strings.Contains(written[0], "hello") {
		t.Errorf("Logger did not contain expected message 'hello', got '%s'", written[0])
	}

	if !strings.Contains(written[0], "level=TRACE") {
		t.Errorf("Logger did not contain expected level 'TRACE', got '%s'", written[0])
	}
}

func TestCanLogWarnMessages(t *testing.T) {
	m := mockWriter{&[]string{}}

	l := NewWithWriter(m, slog.LevelWarn)
	l.Warn("hello")

	written := *m.written
	if len(written) != 1 {
		t.Fatalf("Logger wrote the wrong number of times expected %d, but got %d", 1, len(*m.written))
	}

	if !strings.Contains(written[0], "hello") {
		t.Errorf("Logger did not contain expected message 'hello', got '%s'", written[0])
	}

	if !strings.Contains(written[0], "level=WARN") {
		t.Errorf("Logger did not contain expected level 'WARN', got '%s'", written[0])
	}
}

func TestDoesNotPanicWithNilAsWriters(t *testing.T) {
	l := New(nil, nil)
	l.Warn("hello")
	l.Debug("hello")
	l.Trace("hello")
}

func TestCanLogExpressions(t *testing.T) {
	m := mockWriter{&[]string{}}

	l := NewWithWriter(m, slog.LevelWarn)

	l.Warn("hello: %s", e.String("<name here>"))

	written := *m.written
	if len(written) != 1 {
		t.Fatalf("Logger wrote the wrong number of times expected %d, but got %d", 1, len(written))
	}

	// Check that expressions are logged as structured data
	if !strings.Contains(written[0], "args=") {
		t.Errorf("Logger did not contain structured args, got '%s'", written[0])
	}

	if !strings.Contains(written[0], "level=WARN") {
		t.Errorf("Logger did not contain WARN level, got '%s'", written[0])
	}

	if !strings.Contains(written[0], "hello") {
		t.Errorf("Logger did not contain message, got '%s'", written[0])
	}
}