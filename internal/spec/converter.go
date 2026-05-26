package spec

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// ConvertToInt converts a value to int64.
func ConvertToInt(value interface{}) (int64, error) {
	if value == nil {
		return 0, fmt.Errorf("cannot convert nil to int")
	}
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case json.Number:
		return v.Int64()
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// ConvertToFloat converts a value to float64.
func ConvertToFloat(value interface{}) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("cannot convert nil to float")
	}
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	case json.Number:
		return v.Float64()
	default:
		return 0, fmt.Errorf("cannot convert %T to float", value)
	}
}

// ConvertToBool converts a value to bool.
func ConvertToBool(value interface{}) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("cannot convert nil to bool")
	}
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case float64:
		return v != 0, nil
	case string:
		return strconv.ParseBool(v)
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return false, err
		}
		return i != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// ConvertToString converts a value to string.
func ConvertToString(value interface{}) (string, error) {
	if value == nil {
		return "", fmt.Errorf("cannot convert nil to string")
	}
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case json.Number:
		return v.String(), nil
	default:
		if bytes, err := json.Marshal(v); err == nil {
			return string(bytes), nil
		}
		return fmt.Sprintf("%v", v), nil
	}
}

// ConvertToTime converts a value to int64 millisecond timestamp.
func ConvertToTime(value interface{}) (int64, error) {
	if value == nil {
		return 0, fmt.Errorf("cannot convert nil to time")
	}
	switch v := value.(type) {
	case time.Time:
		return v.UnixMilli(), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UnixMilli(), nil
		}
		if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
			return t.UnixMilli(), nil
		}
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to time", value)
	}
}

// ConvertToTargetType converts a value to the target type specified by a string.
// Supported targets: "int", "float", "bool", "string", "time", "json".
func ConvertToTargetType(value interface{}, targetType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch targetType {
	case "int":
		return ConvertToInt(value)
	case "float":
		return ConvertToFloat(value)
	case "bool":
		return ConvertToBool(value)
	case "string":
		return ConvertToString(value)
	case "time":
		return ConvertToTime(value)
	case "json":
		return toJSON(value)
	default:
		return value, nil
	}
}

func toJSON(value interface{}) (interface{}, error) {
	switch value.(type) {
	case map[string]interface{}, []interface{}:
		return value, nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON value: %w", err)
	}
	var result interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON value: %w", err)
	}
	return result, nil
}
