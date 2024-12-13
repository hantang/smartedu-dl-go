package main

import (
	"flag"
	"log/slog"

	"github.com/hantang/smartedudlgo/internal/ui"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()
	if *debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Debug mode enabled")
	}

	ui.InitUI()
}
