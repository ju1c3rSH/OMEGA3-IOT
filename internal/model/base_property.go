package model

import "fmt"

type PropertyItem struct {
	Value TypedValue   `json:"value"`
	Meta  PropertyMeta `json:"meta"`
}

type PropertyValue struct {
	DeviceID    string     `json:"device_id"`
	PropertyKey string     `json:"property_key"`
	Value       TypedValue `json:"value"`
}

func (pmd *PropertyMeta) Validate(tv *TypedValue) error {
	if tv.Type != pmd.Format {
		return fmt.Errorf("type not match")
	}

	if len(pmd.Range) == 2 {
		switch val := tv.V.(type) {

		case int:
			if float64(val) < pmd.Range[0] || float64(val) > pmd.Range[1] {
				return fmt.Errorf("value out of range")
			}
		case float64:
			if val < pmd.Range[0] || val > pmd.Range[1] {
				return fmt.Errorf("value out of range")
			}
		}
	}

	if len(pmd.Enum) > 0 {
		strVal := fmt.Sprintf("%v", tv.Value)
		for _, e := range pmd.Enum {
			if e == strVal {
				return nil
			}
		}
		return fmt.Errorf("value %v not in enum %v", tv.Value, pmd.Enum)
	}

	return nil
}
