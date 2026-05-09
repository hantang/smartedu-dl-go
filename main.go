package main

import (
	"flag"
	"log/slog"

	"github.com/hantang/smartedudlgo/internal/ui"
)

func main() {
	isDebug := flag.Bool("debug", false, "Enable debug logging")
	isLocal := flag.Bool("local", false, "Enable local file mode")
	isSave := flag.Bool("save", false, "Save fetched JSON data to data/ directory; only active with --debug")
	threads := flag.Int("threads", 10, "Max concurrency for video download")
	flag.Parse()
	if *isDebug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Debug mode enabled")
	}
	if *isLocal {
		slog.Debug("Local file mode enabled")
	}
	saveFetchedData := *isDebug && *isSave
	if *isSave && !*isDebug {
		slog.Warn("--save ignored because debug mode is disabled")
	}
	if saveFetchedData {
		slog.Debug("Save fetched JSON data enabled")
	}

	// os.Setenv("FYNE_FONT", "./assets/DouyinSansBold.ttf")
	ui.InitUI(*isLocal, *threads, saveFetchedData)
}
