package main

import (
	"log/slog"
	"os"
)

var opts = &slog.HandlerOptions{
	AddSource: true,
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
