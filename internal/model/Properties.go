package model

import (
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type Properties struct {
	Items map[string]*TypedInstancePropertyItem `json:"items"`
}
type InstancePropertyItem struct {
	Value string       `json:"value"`
	Meta  PropertyMeta `json:"meta"`
}

type TypedInstancePropertyItem struct {
	Value TypedValue   `json:"value"`
	Meta  PropertyMeta `json:"meta"`
}

//TODO 这里是设备通过MQTT上传的重要节点

type TypedValue struct {
	V         interface{} `json:"v"`
	Type      string      `json:"type"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

func (tv *TypedValue) String() string {
	str, err := tv.ToString()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return str
}
func (tv *TypedValue) ToString() (string, error) {
	if tv == nil {
		return "", fmt.Errorf("typed value is nil")
	}

	switch tv.Type {
	case "string":
		if str, ok := tv.V.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("value is not string type: got %T", tv.V)

	case "int":
		switch v := tv.V.(type) {
		case int64:
			return strconv.FormatInt(v, 10), nil
		case int:
			return strconv.Itoa(v), nil
		case int32:
			return strconv.FormatInt(int64(v), 10), nil
		case int16:
			return strconv.FormatInt(int64(v), 10), nil
		case int8:
			return strconv.FormatInt(int64(v), 10), nil
		case uint64:
			return strconv.FormatUint(v, 10), nil
		case uint:
			return strconv.FormatUint(uint64(v), 10), nil
		default:
			return "", fmt.Errorf("value is not integer type: got %T", tv.V)
		}

	case "float":
		// 处理多种浮点类型
		switch v := tv.V.(type) {
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64), nil
		case float32:
			return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
		default:
			return "", fmt.Errorf("value is not float type: got %T", tv.V)
		}

	case "bool":
		if b, ok := tv.V.(bool); ok {
			return strconv.FormatBool(b), nil
		}
		return "", fmt.Errorf("value is not bool type: got %T", tv.V)

	case "time":
		// 时间戳转字符串
		switch v := tv.V.(type) {
		case int64:
			t := time.UnixMilli(v)
			return t.Format(time.RFC3339), nil
		case int:
			t := time.UnixMilli(int64(v))
			return t.Format(time.RFC3339), nil
		case time.Time:
			return v.Format(time.RFC3339), nil
		case string:
			// 已经是字符串格式的时间
			return v, nil
		default:
			return "", fmt.Errorf("value is not timestamp type: got %T", tv.V)
		}

	case "json":
		// JSON 对象转字符串
		switch v := tv.V.(type) {
		case []byte:
			return string(v), nil
		case json.RawMessage:
			return string(v), nil
		case string:
			return v, nil
		case map[string]interface{}, []interface{}:
			// 如果是解析后的 JSON 对象，重新序列化
			bytes, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("failed to marshal JSON: %w", err)
			}
			return string(bytes), nil
		default:
			return "", fmt.Errorf("value is not JSON type: got %T", tv.V)
		}

	case "binary", "bytes":
		// 处理二进制数据
		switch v := tv.V.(type) {
		case []byte:
			return base64.StdEncoding.EncodeToString(v), nil
		case string:
			// 可能是 base64 编码的字符串
			return v, nil
		default:
			return "", fmt.Errorf("value is not binary type: got %T", tv.V)
		}

	default:
		// 未知类型，尝试 JSON 序列化
		if bytes, err := json.Marshal(tv.V); err == nil {
			return string(bytes), nil
		}
		// 最后使用 fmt.Sprint
		return fmt.Sprintf("%v", tv.V), nil
	}
}

// 带默认值的版本
func (tv *TypedValue) ToStringOrDefault(defaultValue string) string {
	if str, err := tv.ToString(); err == nil {
		return str
	}
	return defaultValue
}

// 获取原始值的便捷方法
func (tv *TypedValue) Raw() interface{} {
	if tv == nil {
		return nil
	}
	return tv.V
}

// 判断是否为空
func (tv *TypedValue) IsEmpty() bool {
	if tv == nil || tv.V == nil {
		return true
	}

	switch tv.Type {
	case "string":
		return tv.V.(string) == ""
	case "int":
		if i, ok := tv.V.(int64); ok {
			return i == 0
		}
	case "float":
		if f, ok := tv.V.(float64); ok {
			return f == 0
		}
	case "bool":
		if b, ok := tv.V.(bool); ok {
			return !b
		}
	}
	return false
}

// NewTIPFromOldWay 从旧的字符串值创建 TypedInstancePropertyItem
// oldString: 旧的字符串格式的值
// typeOfString: 属性的类型（对应 PropertyMeta.Format）
func NewTIPFromOldWay(oldString string, typeOfString string) (*TypedInstancePropertyItem, error) {
	if oldString == "" {
		return nil, fmt.Errorf("oldString cannot be empty")
	}

	var convertedValue interface{}
	var err error

	switch typeOfString {
	case "int":
		convertedValue, err = strconv.ParseInt(oldString, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse int from '%s': %w", oldString, err)
		}

	case "float":
		convertedValue, err = strconv.ParseFloat(oldString, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse float from '%s': %w", oldString, err)
		}

	case "bool":
		convertedValue, err = strconv.ParseBool(oldString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bool from '%s': %w", oldString, err)
		}

	case "string":
		convertedValue = oldString

	case "time":
		// 尝试多种时间格式
		timeFormats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
			"15:04:05",
		}

		var t time.Time
		for _, format := range timeFormats {
			t, err = time.Parse(format, oldString)
			if err == nil {
				break
			}
		}

		if err != nil {
			// 如果都失败，尝试解析 Unix 时间戳
			timestamp, parseIntErr := strconv.ParseInt(oldString, 10, 64)
			if parseIntErr != nil {
				return nil, fmt.Errorf("failed to parse time from '%s': %w", oldString, err)
			}
			convertedValue = timestamp
		} else {
			convertedValue = t.UnixMilli()
		}

	default:
		// 对于未知类型，尝试作为 JSON 解析
		var jsonValue interface{}
		if jsonErr := json.Unmarshal([]byte(oldString), &jsonValue); jsonErr == nil {
			convertedValue = jsonValue
		} else {
			// 如果 JSON 解析失败，就作为字符串保留
			convertedValue = oldString
		}
	}

	typedValue, err := NewTypedValue(convertedValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create typed value: %w", err)
	}

	return &TypedInstancePropertyItem{
		Value: *typedValue,
		Meta:  PropertyMeta{Format: typeOfString}, // 注意：这里只有 Format，其他元数据需要另外设置
	}, nil
}

// NewTypedInstancePropertyItem 从任意值和属性元数据创建 TypedInstancePropertyItem
func NewTypedInstancePropertyItem(v interface{}, pm PropertyMeta) (*TypedInstancePropertyItem, error) {
	if pm.Format == "" {
		return nil, fmt.Errorf("property meta Format cannot be empty")
	}

	convertedValue, err := convertValueToType(v, pm.Format)
	if err != nil {
		return nil, fmt.Errorf("value conversion failed: %w", err)
	}

	typedValue, err := NewTypedValue(convertedValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create typed value: %w", err)
	}

	if err := validateValueWithMeta(typedValue, pm); err != nil {
		return nil, fmt.Errorf("value validation failed: %w", err)
	}

	return &TypedInstancePropertyItem{
		Value: *typedValue,
		Meta:  pm,
	}, nil
}

// 辅助函数：将值转换为指定类型
func convertValueToType(v interface{}, targetType string) (interface{}, error) {
	if v == nil {
		return nil, nil
	}

	switch targetType {
	case "int":
		switch val := v.(type) {
		case int:
			return int64(val), nil
		case int8:
			return int64(val), nil
		case int16:
			return int64(val), nil
		case int32:
			return int64(val), nil
		case int64:
			return val, nil
		case float32:
			return int64(val), nil
		case float64:
			return int64(val), nil
		case string:
			return strconv.ParseInt(val, 10, 64)
		case json.Number:
			return val.Int64()
		default:
			return nil, fmt.Errorf("cannot convert %T to int", v)
		}

	case "float":
		switch val := v.(type) {
		case int:
			return float64(val), nil
		case int8:
			return float64(val), nil
		case int16:
			return float64(val), nil
		case int32:
			return float64(val), nil
		case int64:
			return float64(val), nil
		case float32:
			return float64(val), nil
		case float64:
			return val, nil
		case string:
			return strconv.ParseFloat(val, 64)
		case json.Number:
			return val.Float64()
		default:
			return nil, fmt.Errorf("cannot convert %T to float", v)
		}

	case "bool":
		switch val := v.(type) {
		case bool:
			return val, nil
		case string:
			return strconv.ParseBool(val)
		case int:
			return val != 0, nil
		case json.Number:
			i, _ := val.Int64()
			return i != 0, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to bool", v)
		}

	case "string":
		switch val := v.(type) {
		case string:
			return val, nil
		case json.Number:
			return val.String(), nil
		default:
			return fmt.Sprintf("%v", v), nil
		}

	case "time":
		switch val := v.(type) {
		case time.Time:
			return val.UnixMilli(), nil
		case int64:
			return val, nil
		case string:
			// 尝试解析时间
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				return t.UnixMilli(), nil
			}
			if t, err := time.Parse("2006-01-02 15:04:05", val); err == nil {
				return t.UnixMilli(), nil
			}
			// 尝试作为时间戳解析
			return strconv.ParseInt(val, 10, 64)
		default:
			return nil, fmt.Errorf("cannot convert %T to time", v)
		}

	default:

		return v, nil
	}
}

// 根据元数据验证值
func validateValueWithMeta(tv *TypedValue, pm PropertyMeta) error {
	// 1. 验证类型是否匹配
	if tv.Type != pm.Format {
		return fmt.Errorf("type mismatch: expected %s, got %s", pm.Format, tv.Type)
	}

	// 2. 验证范围
	if len(pm.Range) == 2 {
		switch tv.Type {
		case "int":
			val := tv.V.(int64)
			if val < int64(pm.Range[0]) || val > int64(pm.Range[1]) {
				return fmt.Errorf("value %d out of range [%v, %v]", val, pm.Range[0], pm.Range[1])
			}
		case "float":
			val := tv.V.(float64)
			if val < pm.Range[0] || val > pm.Range[1] {
				return fmt.Errorf("value %v out of range [%v, %v]", val, pm.Range[0], pm.Range[1])
			}
		}
	}

	// 3. 验证枚举值
	if len(pm.Enum) > 0 {
		strVal := fmt.Sprintf("%v", tv.V)
		valid := false
		for _, e := range pm.Enum {
			if strVal == e {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("value %s not in enum %v", strVal, pm.Enum)
		}
	}

	return nil
}

// 批量从旧格式创建新格式属性
func NewTypedPropertiesFromOld(deviceType *DeviceType, oldValues map[string]string) (map[string]*TypedInstancePropertyItem, error) {
	props := make(map[string]*TypedInstancePropertyItem)

	for key, oldValue := range oldValues {
		// 获取属性的元数据
		meta, ok := deviceType.Properties[key]
		if !ok {
			return nil, fmt.Errorf("property %s not found in device type", key)
		}

		// 从旧值创建类型化属性
		item, err := NewTIPFromOldWay(oldValue, meta.Format)
		if err != nil {
			return nil, fmt.Errorf("failed to convert property %s: %w", key, err)
		}

		// 设置完整的元数据
		item.Meta = meta

		props[key] = item
	}

	return props, nil
}

// NewTypedValueFromOld 从旧的字符串值和类型信息创建 TypedValue
// oldValue: 旧的字符串格式的值
// valueType: 值的类型（对应 PropertyMeta.Format）
func NewTypedValueFromOld(oldValue string, valueType string) (*TypedValue, error) {
	if oldValue == "" {
		return nil, fmt.Errorf("oldValue cannot be empty")
	}

	var convertedValue interface{}
	var err error

	switch valueType {
	case "int":
		convertedValue, err = strconv.ParseInt(oldValue, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse int from '%s': %w", oldValue, err)
		}

	case "float":
		convertedValue, err = strconv.ParseFloat(oldValue, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse float from '%s': %w", oldValue, err)
		}

	case "bool":
		convertedValue, err = strconv.ParseBool(oldValue)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bool from '%s': %w", oldValue, err)
		}

	case "string":
		convertedValue = oldValue

	case "time":
		// 尝试多种时间格式
		timeFormats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
			"15:04:05",
			time.RFC3339Nano,
			time.DateTime,
			time.DateOnly,
			time.TimeOnly,
		}

		var t time.Time
		for _, format := range timeFormats {
			t, err = time.Parse(format, oldValue)
			if err == nil {
				break
			}
		}

		if err != nil {
			// 如果都失败，尝试解析 Unix 时间戳（秒、毫秒、微秒、纳秒）
			timestamp, parseIntErr := strconv.ParseInt(oldValue, 10, 64)
			if parseIntErr != nil {
				return nil, fmt.Errorf("failed to parse time from '%s': %w", oldValue, err)
			}

			// 根据时间戳的长度判断单位
			switch {
			case timestamp > 1e18: // 纳秒
				convertedValue = timestamp / 1e6 // 转毫秒
			case timestamp > 1e15: // 微秒
				convertedValue = timestamp / 1e3 // 转毫秒
			case timestamp > 1e12: // 毫秒
				convertedValue = timestamp
			case timestamp > 1e9: // 秒
				convertedValue = timestamp * 1000 // 转毫秒
			default:
				convertedValue = timestamp * 1000 // 假设是秒，转毫秒
			}
		} else {
			convertedValue = t.UnixMilli()
		}

	case "json":
		// 尝试解析 JSON
		var jsonValue interface{}
		if err := json.Unmarshal([]byte(oldValue), &jsonValue); err != nil {
			return nil, fmt.Errorf("failed to parse JSON from '%s': %w", oldValue, err)
		}
		convertedValue = jsonValue

	case "binary", "bytes":
		//处理二进制数据（可能是base64编码的字符串）
		convertedValue = []byte(oldValue)
		valueType = "binary" // 保持类型

	default:
		// 对于未知类型，尝试作为josn解析，如果失败就作为字符串保留
		var jsonValue interface{}
		if jsonErr := json.Unmarshal([]byte(oldValue), &jsonValue); jsonErr == nil {
			convertedValue = jsonValue
			valueType = "json"
		} else {
			convertedValue = oldValue
			// 保持原类型，但如果原类型为空，设为 string
			if valueType == "" {
				valueType = "string"
			}
		}
	}

	return &TypedValue{
		V:         convertedValue,
		Type:      valueType,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func NewTypedValue(v interface{}) (*TypedValue, error) {
	tv := &TypedValue{Timestamp: time.Now().UnixMilli()}

	switch val := v.(type) {
	case int, int8, int16, int32, int64:
		tv.V = reflect.ValueOf(v).Int()
		tv.Type = "int"
	case uint, uint8, uint16, uint32, uint64:
		tv.V = reflect.ValueOf(v).Uint()
		tv.Type = "int" // 无符号整数也作为 int 处理
	case float32, float64:
		tv.V = reflect.ValueOf(v).Float()
		tv.Type = "float"
	case bool:
		tv.V = val
		tv.Type = "bool"
	case string:
		tv.V = val
		tv.Type = "string"
	case time.Time:
		tv.V = val.UnixMilli()
		tv.Type = "time"
	case []byte:
		tv.V = val
		tv.Type = "binary"
	case json.RawMessage:
		tv.V = val
		tv.Type = "json"
	case map[string]interface{}, []interface{}:
		// 复杂类型，尝试 JSON 序列化
		jsonBytes, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal complex type: %w", err)
		}
		tv.V = json.RawMessage(jsonBytes)
		tv.Type = "json"
	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
	return tv, nil
}

func (tv TypedValue) Value() (driver.Value, error) {
	return json.Marshal(tv)
}

func (tv *TypedValue) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	return json.Unmarshal(value.([]byte), tv)
}

func (p Properties) Value() (driver.Value, error) {
	if p.Items == nil {
		return nil, nil
	}
	return json.Marshal(p.Items)
}

func (p *Properties) Scan(value interface{}) error {
	if value == nil {
		p.Items = make(map[string]*TypedInstancePropertyItem)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into Properties", value)
	}

	items := make(map[string]*TypedInstancePropertyItem)
	if err := json.Unmarshal(bytes, &items); err != nil {
		return err
	}

	p.Items = items
	return nil
}
