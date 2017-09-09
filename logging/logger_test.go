package logging

import (
	"testing"
)

type mockWriter struct {
	written *[]string
}

func (m mockWriter) Write(p []byte) (int, error) {
	*m.written = append(*m.written, string(p))
	return len(p), nil
}

func TestCanLogDebugMessages(t *testing.T) {
	m := mockWriter{&[]string{}}

	l := New(m, nil)
	l.Debug("hello")

	written := *m.written
	if len(written) != 1 {
		t.Fatalf("Logger wrote the wrong number of times expected %d, but got %d", 1, len(*m.written))
	}

	if written[0][len(written[0])-6:] != "hello\n" {
		t.Errorf("Logger wrote the wrong message. Expected '%s', but got '%s'", "hello\n", written[0][len(written[0])-6:])
	}

	if written[0][:5] != "DEBUG" {
		t.Errorf("Logger wrote the wrong prefix. Expected '%s', but got '%s'", "DEBUG", written[0][:5])
	}
}

func TestCanLogWarnMessages(t *testing.T) {
	m := mockWriter{&[]string{}}

	l := New(nil, m)
	l.Warn("hello")

	written := *m.written
	if len(written) != 1 {
		t.Fatalf("Logger wrote the wrong number of times expected %d, but got %d", 1, len(*m.written))
	}

	if written[0][len(written[0])-6:] != "hello\n" {
		t.Errorf("Logger wrote the wrong message. Expected '%s', but got '%s'", "hello\n", written[0][len(written[0])-6:])
	}

	if written[0][:4] != "WARN" {
		t.Errorf("Logger wrote the wrong prefix. Expected '%s', but got '%s'", "WARN", written[0][:4])
	}
}

func TestDoesNotPanicWithNilAsWriters(t *testing.T) {
	l := New(nil, nil)
	l.Warn("hello")
	l.Debug("hello")
}
