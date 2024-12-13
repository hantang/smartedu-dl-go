package main

import (
	"log/slog"

	"github.com/hantang/smartedudlgo/internal/ui"
)

func main() {
	// slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Debug("Debug mode enabled")

	ui.InitUI()
}
