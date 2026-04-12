package main

import (
	"log/slog"
	"testing"
)

func TestInitLogger_JSONForRun(t *testing.T) {
	orig := slog.Default()
	defer slog.SetDefault(orig)
	initLogger("run")
	handler := slog.Default().Handler()
	if _, ok := handler.(*slog.JSONHandler); !ok {
		t.Errorf("expected JSONHandler for 'run', got %T", handler)
	}
}

func TestInitLogger_JSONForWatch(t *testing.T) {
	orig := slog.Default()
	defer slog.SetDefault(orig)
	initLogger("watch")
	handler := slog.Default().Handler()
	if _, ok := handler.(*slog.JSONHandler); !ok {
		t.Errorf("expected JSONHandler for 'watch', got %T", handler)
	}
}

func TestInitLogger_TextForPreview(t *testing.T) {
	orig := slog.Default()
	defer slog.SetDefault(orig)
	initLogger("preview")
	handler := slog.Default().Handler()
	if _, ok := handler.(*slog.TextHandler); !ok {
		t.Errorf("expected TextHandler for 'preview', got %T", handler)
	}
}
