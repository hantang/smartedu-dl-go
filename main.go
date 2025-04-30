package main

import (
	"flag"
	"log/slog"

	"github.com/hantang/smartedudlgo/internal/ui"
)

func main() {
	isDebug := flag.Bool("debug", false, "Enable debug logging")
	isLocal := flag.Bool("local", false, "Enable local file mode")
	flag.Parse()
	if *isDebug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Debug mode enabled")
	}
	if *isLocal {
		slog.Debug("Local file mode enabled")
	}

	// os.Setenv("FYNE_FONT", "./assets/DouyinSansBold.ttf")
	ui.InitUI(*isLocal)
}
