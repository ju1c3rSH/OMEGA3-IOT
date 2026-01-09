package model

import "time"

type Action struct {
	Command   string      `json:"command"`
	Params    interface{} `json:"params"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

type ActionType string

const (
	GOON                   ActionType = "go_on"
	EnablePropertiesUpload ActionType = "enable_properties_upload"
	ToggleLight            ActionType = "ToggleLight"
)

func NewEnablePropertiesUploadAction(intervalSec int) *Action {
	return &Action{
		Command: string(EnablePropertiesUpload),
		Params: map[string]interface{}{
			"interval_sec": intervalSec,
		},
		Timestamp: time.Now().Unix(),
	}
}

func NewToggleLightAction() *Action {
	return &Action{
		Command:   string(ToggleLight),
		Params:    map[string]interface{}{},
		Timestamp: time.Now().Unix(),
	}
}
