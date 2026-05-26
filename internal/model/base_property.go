package model

type PropertyValue struct {
	DeviceID    string     `json:"device_id"`
	PropertyKey string     `json:"property_key"`
	Value       TypedValue `json:"value"`
}
