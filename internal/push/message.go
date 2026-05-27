package push

import (
	"encoding/json"
	"time"
)

// Message types — server → client
const (
	TypeEventPush       = "event.push"
	TypeDeviceStatus    = "device.status"
	TypePropertyUpdate  = "property.update"
	TypeActionResult    = "action.result"
	TypeSystemNotice    = "system.notice"
	TypePong            = "pong"
	TypeActionResponse  = "action.response"
)

// Message types — client → server
const (
	TypeACK        = "ack"
	TypeActionSend = "action.send"
	TypePing       = "ping"
)

// Message is the envelope for all WebSocket communication.
type Message struct {
	Type    string      `json:"type"`
	Seq     int64       `json:"seq,omitempty"`
	TS      int64       `json:"ts"`
	Payload interface{} `json:"payload,omitempty"`
}

// NewMessage creates a new Message with the current timestamp.
func NewMessage(msgType string, payload interface{}) *Message {
	return &Message{
		Type:    msgType,
		TS:      time.Now().Unix(),
		Payload: payload,
	}
}

// NewSequencedMessage creates a Message with a sequence number (for ACK-eligible messages).
func NewSequencedMessage(msgType string, seq int64, payload interface{}) *Message {
	return &Message{
		Type:    msgType,
		Seq:     seq,
		TS:      time.Now().Unix(),
		Payload: payload,
	}
}

// ─── Server → Client Payloads ───

// EventPushPayload is sent when a device reports an event.
type EventPushPayload struct {
	EventKey   string      `json:"event_key"`
	DeviceUUID string      `json:"device_uuid"`
	Severity   string      `json:"severity"`
	Data       interface{} `json:"data,omitempty"`
}

// DeviceStatusPayload is sent when a device goes online/offline.
type DeviceStatusPayload struct {
	DeviceUUID string `json:"device_uuid"`
	Online     bool   `json:"online"`
	LastSeen   int64  `json:"last_seen"`
}

// PropertyUpdatePayload is sent when device properties change.
type PropertyUpdatePayload struct {
	DeviceUUID string                 `json:"device_uuid"`
	Properties map[string]interface{} `json:"properties"`
}

// ActionResultPayload is sent when an action execution result arrives.
type ActionResultPayload struct {
	DeviceUUID string `json:"device_uuid"`
	Command    string `json:"command"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// SystemNoticePayload is sent for system-level notifications.
type SystemNoticePayload struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// ActionResponsePayload is sent in response to an action.send from the client.
type ActionResponsePayload struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ─── Client → Server Payloads ───

// ACKPayload is sent by the client to acknowledge receipt of a message.
type ACKPayload struct {
	ACKSeq int64 `json:"ack_seq"`
}

// ActionSendPayload is sent by the client to trigger a device action.
type ActionSendPayload struct {
	DeviceUUID string                 `json:"device_uuid"`
	Command    string                 `json:"command"`
	Params     map[string]interface{} `json:"params,omitempty"`
}

// IncomingMessage is the raw message received from the client.
type IncomingMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}
