package bot

import "github.com/vnxcius/sss-backend/internal/http/events"

func getServerStatus() string {
	currentStatus := events.ServerStatusManager.GetStatus()
	return string(currentStatus)
}

// func onServerStatusChange() {
// 	currentStatus := events.ServerStatusManager.GetStatus()
// }
