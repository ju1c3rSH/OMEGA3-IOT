package model

type DeviceEvent struct {
	DeviceUUID string `json:"device_uuid"`
	Timestamp  int64  `json:"timestamp"`
	EventKey   string `json:"event_key"`
	Type       string `json:"type"`
	Content    string `json:"content"`
}
