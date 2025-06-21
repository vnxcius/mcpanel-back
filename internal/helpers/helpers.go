package helpers

import (
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type modlist struct {
	Mods []mod `json:"mods"`
}

type mod struct {
	Name string `json:"name"`
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
			name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			mods = append(mods, mod{
				Name: name,
			})
		}
	}

	return json.Marshal(modlist{
		Mods: mods,
	})
}

/*
Checks if the Minecraft server is online by dialing a TCP connection up to
120 seconds. Returns true if the connection is successful, false otherwise.
*/
func WaitMinecraftServer(wantedStatus string) bool {
	const timeout = 3 * time.Second

	for range 10 { // retry for up to ~120 seconds
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

func IsMinecraftCurrentlyOnline() bool {
	slog.Info("Checking if Minecraft server is online", "addr", addr)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
