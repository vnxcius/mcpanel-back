package logging

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ModChangelog struct {
	mu      sync.Mutex
	current *os.File
	date    string
	dir     string
}

type ModChangeType string

type ModChangeEntry struct {
	Time string        `json:"time"`
	Type ModChangeType `json:"type"`
	Name string        `json:"name"`
}

var modlog *ModChangelog

const (
	ModAdded   ModChangeType = "added"
	ModDeleted ModChangeType = "deleted"
	ModUpdated ModChangeType = "updated"
)

func (t ModChangeType) IsValid() bool {
	return t == ModAdded || t == ModDeleted || t == ModUpdated
}

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

func SetupModlistChangelog(dir string) {
	_ = os.MkdirAll(dir, 0o755)
	modlog = &ModChangelog{dir: dir}
	modlog.rotateIfNeeded()
}

func LogModChange(name string, changeType ModChangeType) ModChangeEntry {
	if !changeType.IsValid() {
		slog.Warn("Invalid mod change type", "type", changeType)
		return ModChangeEntry{}
	}

	entry := ModChangeEntry{
		Time: time.Now().Format(time.RFC3339),
		Name: name,
		Type: changeType,
	}

	modlog.mu.Lock()
	defer modlog.mu.Unlock()

	modlog.rotateIfNeeded()

	_ = json.NewEncoder(modlog.current).Encode(entry)
	return entry
}

func (l *ModChangelog) rotateIfNeeded() {
	today := time.Now().Format("2006-01-02")
	if today == l.date {
		return
	}

	if l.current != nil {
		l.current.Close()
	}

	path := filepath.Join(l.dir, today+".log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic("cannot open mod changelog: " + err.Error())
	}

	l.current = f
	l.date = today
}
