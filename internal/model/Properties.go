package model

import "fmt"

func (t TestDevice) GetValue(name string) (interface{}, error) {
	switch name {
	case "BatteryLevel":
		return int(t.BatteryLevel), nil
	case "Type":
		return t.Type, nil
	default:
		return nil, fmt.Errorf("Unknown device type: %s", name)

	}
}

func (t TestDevice) SetValue(name string, value interface{}) error {
	switch name {
	case "BatteryLevel":
		t.BatteryLevel = value.(int)
	case "Type":
		t.Type = value.(string)
	default:
		return fmt.Errorf("prop not found", name)
	}
	return nil
}

func (t TestDevice) GetAll() map[string]interface{} {
	return map[string]interface{}{
		"BatteryLevel": t.BatteryLevel,
		"Type":         t.Type,
	}
}

func (t TestDevice) GetMeta() map[string]PropertyMeta {
	return nil
}
