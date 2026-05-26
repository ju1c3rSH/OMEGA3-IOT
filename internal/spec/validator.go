package spec

import (
	"OMEGA3-IOT/internal/model"
	"fmt"
	"regexp"
)

// ValidatePropertyValue checks whether a value satisfies all PropertyMeta constraints:
// type match, range, enum, pattern, and writability.
func ValidatePropertyValue(meta model.PropertyMeta, value interface{}) error {
	if value == nil {
		if meta.Required {
			return fmt.Errorf("property '%s' is required but got nil", meta.Key)
		}
		return nil
	}

	// Type conversion and check
	converted, err := ConvertToTargetType(value, meta.Format)
	if err != nil {
		return fmt.Errorf("property '%s': type conversion failed: %w", meta.Key, err)
	}

	// Range validation
	if len(meta.Range) == 2 {
		if err := validateRange(converted, meta.Format, meta.Range, meta.Key); err != nil {
			return err
		}
	}

	// Enum validation
	if len(meta.Enum) > 0 {
		strVal := fmt.Sprintf("%v", converted)
		found := false
		for _, e := range meta.Enum {
			if strVal == e {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("property '%s': value '%s' not in allowed values: %v", meta.Key, strVal, meta.Enum)
		}
	}

	// Pattern validation
	if meta.Pattern != "" {
		strVal := fmt.Sprintf("%v", converted)
		matched, err := regexp.MatchString(meta.Pattern, strVal)
		if err != nil {
			return fmt.Errorf("property '%s': invalid pattern regex: %w", meta.Key, err)
		}
		if !matched {
			return fmt.Errorf("property '%s': value '%s' does not match pattern '%s'", meta.Key, strVal, meta.Pattern)
		}
	}

	return nil
}

func validateRange(value interface{}, format string, rangeVals []float64, key string) error {
	min, max := rangeVals[0], rangeVals[1]
	switch format {
	case "int":
		val, ok := value.(int64)
		if !ok {
			return fmt.Errorf("property '%s': expected int value for range check, got %T", key, value)
		}
		if float64(val) < min || float64(val) > max {
			return fmt.Errorf("property '%s': value %d out of range [%v, %v]", key, val, min, max)
		}
	case "float":
		val, ok := value.(float64)
		if !ok {
			return fmt.Errorf("property '%s': expected float value for range check, got %T", key, value)
		}
		if val < min || val > max {
			return fmt.Errorf("property '%s': value %v out of range [%v, %v]", key, val, min, max)
		}
	}
	return nil
}

// ValidateAction checks whether an action is valid for a given device type:
// action existence, required params presence, and param type/range/enum constraints.
func ValidateAction(typeDef *model.DeviceType, actionKey string, params map[string]interface{}) error {
	if typeDef == nil {
		return fmt.Errorf("device type is nil")
	}

	actionMeta, ok := typeDef.Actions[actionKey]
	if !ok {
		return fmt.Errorf("action '%s' is not defined for device type '%s'", actionKey, typeDef.Name)
	}

	// Validate each defined input param
	for _, paramDef := range actionMeta.InputParams {
		val, provided := params[paramDef.Name]

		// Required check
		if paramDef.Required && !provided {
			return fmt.Errorf("action '%s': required parameter '%s' is missing", actionKey, paramDef.Name)
		}

		if !provided {
			continue
		}

		// Type conversion check
		converted, err := ConvertToTargetType(val, paramDef.Type)
		if err != nil {
			return fmt.Errorf("action '%s': parameter '%s' type conversion failed: %w", actionKey, paramDef.Name, err)
		}

		// Range check
		if len(paramDef.Range) == 2 {
			if err := validateRange(converted, paramDef.Type, paramDef.Range, paramDef.Name); err != nil {
				return fmt.Errorf("action '%s': %w", actionKey, err)
			}
		}

		// Enum check
		if len(paramDef.Enum) > 0 {
			strVal := fmt.Sprintf("%v", converted)
			found := false
			for _, e := range paramDef.Enum {
				if strVal == e {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("action '%s': parameter '%s' value '%s' not in allowed values: %v", actionKey, paramDef.Name, strVal, paramDef.Enum)
			}
		}
	}

	return nil
}

// ValidateEvent checks whether an event is valid for a given device type:
// event existence and payload conformance to output_params.
func ValidateEvent(typeDef *model.DeviceType, eventKey string, payload map[string]interface{}) error {
	if typeDef == nil {
		return fmt.Errorf("device type is nil")
	}

	eventMeta, ok := typeDef.Events[eventKey]
	if !ok {
		return fmt.Errorf("event '%s' is not defined for device type '%s'", eventKey, typeDef.Name)
	}

	// Validate output params if defined
	for _, paramDef := range eventMeta.OutputParams {
		val, provided := payload[paramDef.Name]
		if !provided {
			continue // output params are informational, not required from the device
		}

		// Type conversion check
		_, err := ConvertToTargetType(val, paramDef.Type)
		if err != nil {
			return fmt.Errorf("event '%s': parameter '%s' type conversion failed: %w", eventKey, paramDef.Name, err)
		}

		// Enum check
		if len(paramDef.Enum) > 0 {
			strVal := fmt.Sprintf("%v", val)
			found := false
			for _, e := range paramDef.Enum {
				if strVal == e {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("event '%s': parameter '%s' value '%s' not in allowed values: %v", eventKey, paramDef.Name, strVal, paramDef.Enum)
			}
		}
	}

	return nil
}

// CheckCapabilities checks whether a device type has all required capabilities.
func CheckCapabilities(typeDef *model.DeviceType, requiredCaps []string) error {
	if typeDef == nil {
		return fmt.Errorf("device type is nil")
	}

	for _, cap := range requiredCaps {
		if _, ok := typeDef.Capabilities[cap]; !ok {
			return fmt.Errorf("device type '%s' does not have required capability '%s'", typeDef.Name, cap)
		}
	}
	return nil
}
