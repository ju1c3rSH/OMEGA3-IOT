package spec

import (
	"OMEGA3-IOT/internal/model"
	"testing"
)

func TestValidatePropertyValue_TypeMismatch(t *testing.T) {
	meta := model.PropertyMeta{
		Key:    "battery_level",
		Format: "int",
	}
	err := ValidatePropertyValue(meta, "not_a_number")
	if err == nil {
		t.Error("expected type mismatch error, got nil")
	}
}

func TestValidatePropertyValue_RangeInt(t *testing.T) {
	meta := model.PropertyMeta{
		Key:    "battery_level",
		Format: "int",
		Range:  []float64{0, 100},
	}

	// Valid value
	if err := ValidatePropertyValue(meta, 50); err != nil {
		t.Errorf("expected no error for value 50, got: %v", err)
	}

	// Out of range
	if err := ValidatePropertyValue(meta, 150); err == nil {
		t.Error("expected range error for value 150, got nil")
	}
}

func TestValidatePropertyValue_RangeFloat(t *testing.T) {
	meta := model.PropertyMeta{
		Key:    "temperature",
		Format: "float",
		Range:  []float64{-40, 85},
	}

	if err := ValidatePropertyValue(meta, 25.5); err != nil {
		t.Errorf("expected no error for 25.5, got: %v", err)
	}

	if err := ValidatePropertyValue(meta, 100.0); err == nil {
		t.Error("expected range error for 100.0, got nil")
	}
}

func TestValidatePropertyValue_Enum(t *testing.T) {
	meta := model.PropertyMeta{
		Key:    "device_status",
		Format: "string",
		Enum:   []string{"online", "offline", "sleep", "error"},
	}

	if err := ValidatePropertyValue(meta, "online"); err != nil {
		t.Errorf("expected no error for 'online', got: %v", err)
	}

	if err := ValidatePropertyValue(meta, "unknown"); err == nil {
		t.Error("expected enum error for 'unknown', got nil")
	}
}

func TestValidatePropertyValue_Pattern(t *testing.T) {
	meta := model.PropertyMeta{
		Key:     "gps_location",
		Format:  "string",
		Pattern: `^-?\d+\.\d+,-?\d+\.\d+$`,
	}

	if err := ValidatePropertyValue(meta, "31.2304,121.4737"); err != nil {
		t.Errorf("expected no error for valid GPS, got: %v", err)
	}

	if err := ValidatePropertyValue(meta, "invalid"); err == nil {
		t.Error("expected pattern error for 'invalid', got nil")
	}
}

func TestValidatePropertyValue_Required(t *testing.T) {
	meta := model.PropertyMeta{
		Key:      "switch_status",
		Format:   "bool",
		Required: true,
	}

	if err := ValidatePropertyValue(meta, nil); err == nil {
		t.Error("expected required error for nil, got nil")
	}
}

func TestValidatePropertyValue_NilOptional(t *testing.T) {
	meta := model.PropertyMeta{
		Key:      "remark",
		Format:   "string",
		Required: false,
	}

	if err := ValidatePropertyValue(meta, nil); err != nil {
		t.Errorf("expected no error for nil optional, got: %v", err)
	}
}

func TestValidatePropertyValue_IntConversion(t *testing.T) {
	meta := model.PropertyMeta{
		Key:    "battery_level",
		Format: "int",
		Range:  []float64{0, 100},
	}

	// JSON numbers come as float64
	if err := ValidatePropertyValue(meta, float64(75)); err != nil {
		t.Errorf("expected no error for float64->int conversion, got: %v", err)
	}

	// String number
	if err := ValidatePropertyValue(meta, "50"); err != nil {
		t.Errorf("expected no error for string->int conversion, got: %v", err)
	}
}

func TestValidateAction_Valid(t *testing.T) {
	typeDef := &model.DeviceType{
		Name: "TestDevice",
		Actions: map[string]model.ActionMeta{
			"reboot": {
				Key:         "reboot",
				Description: "Remote reboot",
				InputParams: []model.InputParam{},
				TimeoutSec:  30,
				Idempotent:  true,
			},
		},
	}

	if err := ValidateAction(typeDef, "reboot", nil); err != nil {
		t.Errorf("expected no error for valid action, got: %v", err)
	}
}

func TestValidateAction_UnknownAction(t *testing.T) {
	typeDef := &model.DeviceType{
		Name:    "TestDevice",
		Actions: map[string]model.ActionMeta{},
	}

	if err := ValidateAction(typeDef, "nonexistent", nil); err == nil {
		t.Error("expected unknown action error, got nil")
	}
}

func TestValidateAction_MissingRequiredParam(t *testing.T) {
	typeDef := &model.DeviceType{
		Name: "TestDevice",
		Actions: map[string]model.ActionMeta{
			"update_config": {
				Key: "update_config",
				InputParams: []model.InputParam{
					{Name: "report_interval", Type: "int", Required: true},
				},
			},
		},
	}

	// Missing required param
	if err := ValidateAction(typeDef, "update_config", map[string]interface{}{}); err == nil {
		t.Error("expected missing required param error, got nil")
	}

	// With required param
	if err := ValidateAction(typeDef, "update_config", map[string]interface{}{"report_interval": 60}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateAction_ParamEnum(t *testing.T) {
	typeDef := &model.DeviceType{
		Name: "TestDevice",
		Actions: map[string]model.ActionMeta{
			"calibrate": {
				Key: "calibrate",
				InputParams: []model.InputParam{
					{Name: "target_type", Type: "string", Required: true, Enum: []string{"temperature", "humidity", "all"}},
				},
			},
		},
	}

	if err := ValidateAction(typeDef, "calibrate", map[string]interface{}{"target_type": "temperature"}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if err := ValidateAction(typeDef, "calibrate", map[string]interface{}{"target_type": "invalid"}); err == nil {
		t.Error("expected enum error, got nil")
	}
}

func TestValidateAction_ParamRange(t *testing.T) {
	typeDef := &model.DeviceType{
		Name: "TestDevice",
		Actions: map[string]model.ActionMeta{
			"set_interval": {
				Key: "set_interval",
				InputParams: []model.InputParam{
					{Name: "interval_sec", Type: "int", Required: true, Range: []float64{30, 3600}},
				},
			},
		},
	}

	if err := ValidateAction(typeDef, "set_interval", map[string]interface{}{"interval_sec": 60}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if err := ValidateAction(typeDef, "set_interval", map[string]interface{}{"interval_sec": 5}); err == nil {
		t.Error("expected range error, got nil")
	}
}

func TestValidateEvent_Valid(t *testing.T) {
	typeDef := &model.DeviceType{
		Name: "TestDevice",
		Events: map[string]model.EventMeta{
			"low_battery_alarm": {
				Key:         "low_battery_alarm",
				Description: "Low battery",
				Severity:    "warning",
				OutputParams: []model.OutputParam{
					{Name: "current_level", Type: "int"},
				},
			},
		},
	}

	payload := map[string]interface{}{
		"current_level": 15,
	}

	if err := ValidateEvent(typeDef, "low_battery_alarm", payload); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateEvent_UnknownEvent(t *testing.T) {
	typeDef := &model.DeviceType{
		Name:   "TestDevice",
		Events: map[string]model.EventMeta{},
	}

	if err := ValidateEvent(typeDef, "nonexistent", nil); err == nil {
		t.Error("expected unknown event error, got nil")
	}
}

func TestValidateEvent_EnumParam(t *testing.T) {
	typeDef := &model.DeviceType{
		Name: "TestDevice",
		Events: map[string]model.EventMeta{
			"sos_alert": {
				OutputParams: []model.OutputParam{
					{Name: "trigger_reason", Type: "string", Enum: []string{"button_press", "fall_detected"}},
				},
			},
		},
	}

	// Valid enum
	if err := ValidateEvent(typeDef, "sos_alert", map[string]interface{}{"trigger_reason": "button_press"}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Invalid enum
	if err := ValidateEvent(typeDef, "sos_alert", map[string]interface{}{"trigger_reason": "invalid"}); err == nil {
		t.Error("expected enum error, got nil")
	}
}

func TestCheckCapabilities_AllPresent(t *testing.T) {
	typeDef := &model.DeviceType{
		Name: "TestDevice",
		Capabilities: map[string]model.Capability{
			"gps_module":     {Code: "gps_module", Required: true},
			"motion_sensor":  {Code: "motion_sensor", Required: false},
		},
	}

	if err := CheckCapabilities(typeDef, []string{"gps_module"}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckCapabilities_Missing(t *testing.T) {
	typeDef := &model.DeviceType{
		Name: "TestDevice",
		Capabilities: map[string]model.Capability{
			"gps_module": {Code: "gps_module", Required: true},
		},
	}

	if err := CheckCapabilities(typeDef, []string{"gps_module", "motion_sensor"}); err == nil {
		t.Error("expected missing capability error, got nil")
	}
}
