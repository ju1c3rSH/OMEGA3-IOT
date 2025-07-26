package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Properties struct {
	Items map[string]*PropertyItem `json:"items"`
}

type PropertyItem struct {
	Value string       `json:"value"`
	Meta  PropertyMeta `json:"meta"`
}

func (p Properties) Value() (driver.Value, error) {
	if p.Items == nil {
		return nil, nil
	}
	return json.Marshal(p.Items)
}

func (p *Properties) Scan(value interface{}) error {
	if value == nil {
		p.Items = make(map[string]*PropertyItem)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into Properties", value)
	}

	items := make(map[string]*PropertyItem)
	if err := json.Unmarshal(bytes, &items); err != nil {
		return err
	}

	p.Items = items
	return nil
}

type Property interface {
	GetValue(name string) (interface{}, error)
	SetValue(name string, value interface{}) error
	GetAll() map[string]interface{}
	GetMeta() map[string]PropertyMeta
	//这段是弃用代码
}
