package model

type Capability struct {
	Code        string `yaml:"code" json:"code"`
	Description string `yaml:"description" json:"description"`
	Required    bool   `yaml:"required" json:"required"`
}
