package utils

import (
	"OMEGA3-IOT/internal/model"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type PropertyField struct {
	Value interface{}        `json:"value"`
	Meta  model.PropertyMeta `json:"meta"`
}

type StoredProperty struct {
	Type   string                   `json:"type"`
	Fields map[string]PropertyField `json:"fields"`
}

type StoredProperties []StoredProperty

// Valuer 接口（写入数据库时自动调用）
func (p StoredProperties) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}

// Scanner 接口（读取数据库时自动调用）
func (p *StoredProperties) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
}
