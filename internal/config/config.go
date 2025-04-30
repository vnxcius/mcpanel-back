package config

import (
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func SetupLogger(filePath string) {
	logDir := filepath.Dir(filePath)
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		panic("Failed to create log directory: " + err.Error())
	}

	err = os.Remove(filePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Error("Failed to remove old log file", "path", filePath, "error", err)
	}

	logFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic("Failed to open log file for writing: " + err.Error())
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Fatal("Failed to load location: ", err)
	}

	opts := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				t := a.Value.Time().In(location)
				a.Value = slog.StringValue(t.Format(time.RFC3339))
			}
			return a
		},
		Level: slog.LevelDebug,
	}

	logger := slog.New(slog.NewJSONHandler(multiWriter, opts))
	slog.SetDefault(logger)
}
