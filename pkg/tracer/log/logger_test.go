package log

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

type infoOnlyLogger struct {
	calls int
}

func (l *infoOnlyLogger) Info(string, ...any)  { l.calls++ }
func (l *infoOnlyLogger) Error(string, ...any) { l.calls++ }
func (l *infoOnlyLogger) With(...any) Logger   { return l }

func TestDebugIsDroppedAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	l := newZerologLogger(&buf, zerolog.InfoLevel)

	Debug(l, "Span started", "span_name", "x")
	l.Info("visible")

	out := buf.String()
	if strings.Contains(out, "Span started") {
		t.Fatalf("debug line leaked at info level: %s", out)
	}
	if !strings.Contains(out, "visible") {
		t.Fatalf("info line missing: %s", out)
	}
}

func TestDebugIsWrittenAtDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	l := newZerologLogger(&buf, zerolog.DebugLevel)

	Debug(l, "Span started", "span_name", "x")

	if !strings.Contains(buf.String(), `"span_name":"x"`) {
		t.Fatalf("debug line missing at debug level: %s", buf.String())
	}
}

func TestDebugSkipsLoggersWithoutDebug(t *testing.T) {
	l := &infoOnlyLogger{}

	Debug(l, "Span started")

	if l.calls != 0 {
		t.Fatalf("expected no calls, got %d", l.calls)
	}
}

func TestWithKeepsLevel(t *testing.T) {
	var buf bytes.Buffer
	l := newZerologLogger(&buf, zerolog.InfoLevel).With("k", "v")

	Debug(l, "Span ended")

	if buf.Len() != 0 {
		t.Fatalf("child logger lost level: %s", buf.String())
	}
}
