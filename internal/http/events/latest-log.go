package events

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type logBuffer struct {
	sync.Mutex
	buf   []string
	total int
}

func (m *WSManager) getLastLogLines(n int) ([]string, error) {
	slog.Debug("Getting last log lines", "path", logsPath, "n", n)
	file, err := os.Open(filepath.Clean(logsPath))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

func tailLogs() {
	lines := make(chan string, 1000)
	go tailFile(logsPath, lines)

	const maxLines = 500
	buf := &logBuffer{}

	go startProducer(lines, buf, maxLines)
	go startConsumer(buf)
}

func tailFile(filePath string, lines chan<- string) {
	slog.Debug("Starting log tailing", "path", filePath)
	file, fi := openLog(filePath)
	defer file.Close()

	lastMod, offset := fi.ModTime(), fi.Size()
	file.Seek(0, io.SeekEnd)

	for {
		time.Sleep(time.Second)
		currentFi, err := os.Stat(filePath)
		if err != nil {
			log.Printf("file missing, waiting for recreation...")
			continue
		}

		if rotated(currentFi, lastMod, offset) {
			slog.Debug("Log file rotated, restarting tailing", "path", filePath)
			file.Close()
			file, fi = openLog(filePath)
			lastMod, offset = fi.ModTime(), 0
			continue
		}

		offset, lastMod = readNew(file, currentFi, offset, lines)
	}
}

func openLog(path string) (*os.File, os.FileInfo) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	fi, err := f.Stat()
	if err != nil {
		log.Fatalf("failed to get file stats: %v", err)
	}
	return f, fi
}

func rotated(fi os.FileInfo, lastMod time.Time, offset int64) bool {
	return fi.ModTime().Before(lastMod) || fi.Size() < offset
}

func readNew(f *os.File, fi os.FileInfo, offset int64, out chan<- string) (int64, time.Time) {
	if fi.Size() <= offset {
		return offset, fi.ModTime()
	}

	f.Seek(offset, io.SeekStart)
	buf := make([]byte, fi.Size()-offset)

	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		log.Fatalf("failed to read file: %v", err)
	}

	lines := strings.Split(strings.ReplaceAll(string(buf[:n]), "\r\n", "\n"), "\n")
	for _, line := range lines {
		select {
		case out <- line:
		default: // drop if channel is full
			slog.Warn("Log channel full, dropping line")
		}
	}

	return fi.Size(), fi.ModTime()
}

func startProducer(src <-chan string, dst *logBuffer, max int) {
	for line := range src {
		dst.append(line, max)
	}
}

func startConsumer(buf *logBuffer) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastSent int
	for range ticker.C {
		lines, total := buf.pending(lastSent)
		if len(lines) == 0 {
			continue
		}
		payload, _ := json.Marshal(struct {
			Lines []string `json:"lines"`
		}{Lines: lines})

		Manager.broadcast(Event{Type: EventLogAppend, Payload: payload})
		lastSent = total
	}
}

func (b *logBuffer) append(line string, max int) {
	if line == "" {
		return
	}
	b.Lock()
	b.buf = append(b.buf, line)
	b.total++
	if overflow := len(b.buf) - max; overflow > 0 {
		b.buf = b.buf[overflow:]
	}
	b.Unlock()
}

func (b *logBuffer) pending(since int) (out []string, newTotal int) {
	b.Lock()
	defer b.Unlock()

	if since >= b.total {
		return nil, b.total
	}
	start := len(b.buf) - (b.total - since)
	return b.buf[start:], b.total
}
