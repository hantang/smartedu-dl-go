package main

import (
	"log/slog"

	"github.com/hantang/smartedudlgo/internal"
)

func main() {
	// slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Debug("Debug mode enabled")

	internal.InitUI()
}
