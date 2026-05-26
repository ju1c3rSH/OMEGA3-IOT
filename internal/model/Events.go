package model

// DeviceEvent 运行时事件实例 — 设备上报的事件数据
type DeviceEvent struct {
	DeviceUUID string `json:"device_uuid"`
	Timestamp  int64  `json:"timestamp"`
	EventKey   string `json:"event_key"`
	Type       string `json:"type"`
	Content    string `json:"content"`
}

// EventMeta 事件类型定义 — YAML 中的事件模板
type EventMeta struct {
	Key                  string        `yaml:"key" json:"key"`
	Description          string        `yaml:"description" json:"description"`
	Severity             string        `yaml:"severity" json:"severity"`
	OutputParams         []OutputParam `yaml:"output_params" json:"output_params"`
	RetentionDays        int           `yaml:"retention_days" json:"retention_days"`
	RequiredCapabilities []string      `yaml:"required_capabilities,omitempty" json:"required_capabilities,omitempty"`
}

// OutputParam 事件输出参数定义
type OutputParam struct {
	Name        string   `yaml:"name" json:"name"`
	Type        string   `yaml:"type" json:"type"`
	Description string   `yaml:"description" json:"description"`
	Enum        []string `yaml:"enum,omitempty" json:"enum,omitempty"`
}
