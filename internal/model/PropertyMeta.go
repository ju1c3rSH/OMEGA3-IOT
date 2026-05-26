package model

type PropertyMeta struct {
	Key                  string    `yaml:"key" json:"key"`
	Writable             bool      `yaml:"writable" json:"writable"`
	Description          string    `yaml:"description" json:"description"`
	Unit                 string    `yaml:"unit,omitempty" json:"unit,omitempty"`
	Range                []float64 `yaml:"range,omitempty" json:"range,omitempty"`
	Format               string    `yaml:"format" json:"format"`
	Enum                 []string  `yaml:"enum,omitempty" json:"enum,omitempty"`
	Required             bool      `yaml:"required,omitempty" json:"required,omitempty"`
	Pattern              string    `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	RequiredCapabilities []string  `yaml:"required_capabilities,omitempty" json:"required_capabilities,omitempty"`
}
