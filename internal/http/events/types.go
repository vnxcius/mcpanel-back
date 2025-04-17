package events

type ServerStatus string

const (
	Starting   ServerStatus = "starting"
	Online     ServerStatus = "online"
	Offline    ServerStatus = "offline"
	Restarting ServerStatus = "restarting"
	Stopping   ServerStatus = "stopping"
)
