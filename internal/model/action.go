package model

type Action struct {
	Command   string `json:"command"`
	Params    string `json:"params"`
	Timestamp string `json:"timestamp,omitempty"`
}
