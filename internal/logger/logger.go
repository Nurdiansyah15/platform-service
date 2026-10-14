package logger

import (
	"log/slog"
	"os"
)

// New membuat *slog.Logger berdasarkan environment dan format.
//   - format == "json" / "text": dipakai apa adanya.
//   - format kosong: fallback ke default lama (json untuk production, text untuk lainnya).
//
// Level: production => INFO, lainnya => DEBUG.
// Handler dibungkus contextHandler supaya request_id di context otomatis ter-attach.
func New(env, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	if env == "production" {
		opts.Level = slog.LevelInfo
	}

	useJSON := env == "production"
	switch format {
	case "json":
		useJSON = true
	case "text":
		useJSON = false
	}

	var base slog.Handler
	if useJSON {
		base = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		base = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(&contextHandler{next: base})
}
