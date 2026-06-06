package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewLoggerFormats(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		wantErr   bool
		checkJSON bool
	}{
		{name: "text", format: "text"},
		{name: "json", format: "json", checkJSON: true},
		{name: "unknown", format: "yaml", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger, err := newLogger(&buf, slog.LevelInfo, tt.format)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("newLogger(%q) err = nil, want error", tt.format)
				}
				return
			}
			if err != nil {
				t.Fatalf("newLogger(%q) err = %v", tt.format, err)
			}
			logger.Info("hello", "k", "v")
			out := buf.String()
			if !strings.Contains(out, "hello") {
				t.Errorf("output missing message; got %q", out)
			}
			if tt.checkJSON {
				var m map[string]any
				if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
					t.Errorf("json format should emit valid JSON; got %q (err %v)", out, err)
				}
			}
		})
	}
}

func TestNewLoggerLevelGating(t *testing.T) {
	var buf bytes.Buffer
	logger, err := newLogger(&buf, slog.LevelInfo, "text")
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("should-not-appear")
	if buf.Len() != 0 {
		t.Errorf("Debug record must be suppressed at Info level; got %q", buf.String())
	}

	buf.Reset()
	dbg, err := newLogger(&buf, slog.LevelDebug, "text")
	if err != nil {
		t.Fatal(err)
	}
	dbg.Debug("should-appear")
	if !strings.Contains(buf.String(), "should-appear") {
		t.Errorf("Debug record must appear at Debug level; got %q", buf.String())
	}
}
