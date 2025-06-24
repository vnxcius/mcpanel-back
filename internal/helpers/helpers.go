package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vnxcius/mcpanel-back/internal/logging"
)

type modlist struct {
	Mods []mod `json:"mods"`
}

type mod struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
}

const addr = "localhost:25565"

/*
Returns the list of mods in the mods folder.
*/
func GetMods() ([]byte, error) {
	path := os.Getenv("MODS_PATH")

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var mods []mod
	for _, e := range entries {
		// only add .jar files to the list
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".jar") {
			info, err := os.Stat(filepath.Join(path, e.Name()))
			if err != nil {
				continue
			}
			mods = append(mods, mod{
				Name:    info.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime().Unix(),
			})
		}

	}

	sort.SliceStable(mods, func(i, j int) bool {
		return strings.ToLower(mods[i].Name) < strings.ToLower(mods[j].Name)
	})

	return json.Marshal(modlist{
		Mods: mods,
	})
}

/*
Waits for the Minecraft server to be online or offline for up to 120 seconds.
Returns true if the server is online, false otherwise.
*/
func WaitMinecraftServer(wantedStatus string) bool {
	const timeout = 3 * time.Second

	for range 120 { // retry for up to ~120 seconds
		slog.Info("Waiting until Minecraft server is " + wantedStatus + "...")
		conn, err := net.DialTimeout("tcp", addr, timeout)
		switch wantedStatus {
		case "online":
			if err == nil {
				defer conn.Close()
				return true
			}
		case "offline":
			if err != nil {
				return true
			}
		}

		if err == nil {
			conn.Close()
		}

		time.Sleep(1 * time.Second)
	}

	// Timed out
	slog.Error(
		"Timed out waiting for Minecraft server to come "+wantedStatus,
		"addr", addr,
	)
	return false
}

/*
Checks if the Minecraft server is online by dialing a TCP connection up to
120 seconds. Returns true if the connection is successful, false otherwise.
*/
func IsMinecraftCurrentlyOnline() bool {
	slog.Info("Checking if Minecraft server is online", "addr", addr)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

/*
Deletes a given mod from the mods folder.
*/
func DeleteModFromDir(modsDir, modName string) error {
	target := filepath.Join(modsDir, modName)

	// extra safety: ensure we're still inside modsDir after Join/Clean
	if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(modsDir)) {
		return errors.New("invalid path")
	}

	if err := os.Remove(target); err != nil {
		if os.IsNotExist(err) {
			return errors.New("mod not found")
		}

		return err
	}

	logging.LogModChange(modName, logging.ModDeleted)
	return nil
}

type skippedFile struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

/*
Uploads mods to the mods folder.
Returns the list of uploaded mods, the list of skipped files and any errors.
*/
func UploadModsToDir(
	files []*multipart.FileHeader,
	modsDir string,
	c *gin.Context,
) ([]string, []skippedFile, error) {
	const (
		maxFileSize  int64 = 100 << 20 // 100 MB
		maxTotalSize int64 = 500 << 20 // 500 MB
	)

	var (
		uploaded  []string
		skipped   []skippedFile
		totalSize int64
	)

	for _, fh := range files {
		switch {
		case !strings.EqualFold(filepath.Ext(fh.Filename), ".jar"):
			skipped = append(skipped, skippedFile{fh.Filename, "not .jar"})
			continue
		case fh.Size > maxFileSize:
			skipped = append(skipped, skippedFile{fh.Filename, "file too big"})
			continue
		case totalSize+fh.Size > maxTotalSize:
			skipped = append(skipped, skippedFile{fh.Filename, "total size limit exceeded"})
			continue
		}

		dst := filepath.Join(modsDir, filepath.Base(fh.Filename))
		if !strings.HasPrefix(filepath.Clean(dst), filepath.Clean(modsDir)) {
			skipped = append(skipped, skippedFile{fh.Filename, "invalid path"})
			continue
		}
		if _, err := os.Stat(dst); err == nil {
			skipped = append(skipped, skippedFile{fh.Filename, "duplicate"})
			continue
		}

		if err := c.SaveUploadedFile(fh, dst); err != nil {
			skipped = append(skipped, skippedFile{fh.Filename, "save error"})
			continue
		}

		uploaded = append(uploaded, fh.Filename)
		logging.LogModChange(fh.Filename, logging.ModAdded)
	}

	if len(uploaded) == 0 {
		return nil, skipped, errors.New("no files uploaded")
	}

	return uploaded, skipped, nil
}

/*
Updates a mod in the mods folder.
Returns any errors.
*/
func UpdateModFromDir(
	file *multipart.FileHeader,
	modsDir string,
	oldModBase string, // e.g. "test-1.20.1-1304"
	c *gin.Context,
) error {
	oldPath := filepath.Join(modsDir, oldModBase)
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("mod %q not found", oldModBase)
	}

	newPath := filepath.Join(modsDir, filepath.Base(file.Filename))
	if err := c.SaveUploadedFile(file, newPath); err != nil {
		return fmt.Errorf("saving new mod: %w", err)
	}

	if err := os.Remove(oldPath); err != nil {
		_ = os.Remove(newPath) // rollback
		return fmt.Errorf("removing old mod: %w", err)
	}

	logging.LogModChange(fmt.Sprintf("%s → %s", oldModBase+".jar", file.Filename), logging.ModUpdated)
	return nil
}
