package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Thing interface {
	GetPropertyDef(key string) (*PropertyMeta, bool)

	GetAllPropertyDef() map[string]*PropertyMeta

	SetProperty(deviceUUID string, key string, string, value interface{}) (*PropertyValue, error)
	GetProperty(key string) (*TypedInstancePropertyItem, bool)

	TriggerAction(deviceUUID string, actionKey string, params map[string]interface{})
	ReportEvent(deviceUUID string, eventKey string, payload map[string]interface{})

	HasCapacity(cap string) bool
}

type ThingProxy struct {
	instance *Instance
	typeDef  *DeviceType
	mu       sync.RWMutex
}

func (tp *ThingProxy) GetPropertyMeta(key string) (*PropertyMeta, bool) {
	if _, ok := tp.typeDef.Properties[key]; !ok {
		return nil, false
	}
	def := tp.typeDef.Properties[key]
	return &def, true

}

func (tp *ThingProxy) GetAllPropertyMetaDef() map[string]*PropertyMeta {
	if tp.typeDef == nil {
		return nil
	}
	metas := make(map[string]*PropertyMeta, len(tp.typeDef.Properties))
	for key, item := range tp.typeDef.Properties {
		if _, ok := tp.typeDef.Properties[key]; !ok {
			continue
		}

		metas[key] = &item
	}
	return metas
}

func (tp *ThingProxy) GetProperty(key string) (*TypedInstancePropertyItem, bool) {
	if _, ok := tp.typeDef.Properties[key]; !ok {
		return nil, false
	}
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	if tp.instance.Properties.Items == nil {
		return nil, false
	}
	item := tp.instance.Properties.Items[key]

	return item, true
}

// SetProperty 设置设备属性的值
func (tp *ThingProxy) SetProperty(deviceUUID string, key string, value interface{}) (*PropertyValue, error) {
	if deviceUUID == "" {
		return nil, fmt.Errorf("deviceUUID cannot be empty")
	}
	if key == "" {
		return nil, fmt.Errorf("property key cannot be empty")
	}
	if value == nil {
		return nil, fmt.Errorf("property value cannot be nil")
	}

	propDef, err := tp.getPropertyDefinition(key)
	if err != nil {
		return nil, err
	}

	if !propDef.Meta.Writable {
		return nil, fmt.Errorf("property '%s' is not writable", key)
	}

	typedValue, err := tp.convertValueToType(value, propDef.Meta.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value for property '%s': %w", key, err)
	}

	tv, err := NewTypedValue(typedValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create typed value for property '%s': %w", key, err)
	}

	// 验证值是否符合元数据
	if err := tp.validatePropertyValue(key, tv, propDef.Meta); err != nil {
		return nil, err
	}

	tp.mu.Lock()
	if tp.instance.Properties.Items == nil {
		tp.instance.Properties.Items = make(map[string]*TypedInstancePropertyItem)
	}

	tp.instance.Properties.Items[key] = &TypedInstancePropertyItem{
		Value: *tv,
		Meta:  propDef.Meta,
	}
	tp.mu.Unlock()

	return &PropertyValue{
		DeviceID:    deviceUUID,
		PropertyKey: key,
		Value:       *tv,
	}, nil
}

// 辅助方法：获取属性定义
func (tp *ThingProxy) getPropertyDefinition(key string) (*TypedInstancePropertyItem, error) {
	if tp == nil {
		return nil, fmt.Errorf("thing proxy is nil")
	}

	tp.mu.RLock()
	defer tp.mu.RUnlock()

	if tp.instance.Properties.Items == nil {
		return nil, fmt.Errorf("properties not initialized")
	}

	prop, ok := tp.instance.Properties.Items[key]
	if !ok {
		return nil, fmt.Errorf("property '%s' not found", key)
	}

	return prop, nil
}

// 辅助方法：转换值到指定类型
func (tp *ThingProxy) convertValueToType(value interface{}, targetType string) (interface{}, error) {
	if value == nil {
		return nil, fmt.Errorf("value cannot be nil")
	}

	switch targetType {
	case "int":
		return tp.toInt(value)
	case "float":
		return tp.toFloat(value)
	case "bool":
		return tp.toBool(value)
	case "string":
		return tp.toString(value)
	case "time":
		return tp.toTime(value)
	case "json":
		return tp.toJSON(value)
	default:
		// 未知类型，返回原值
		return value, nil
	}
}

// 转换为整数
func (tp *ThingProxy) toInt(value interface{}) (int64, error) {
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

// 转换为浮点数
func (tp *ThingProxy) toFloat(value interface{}) (float64, error) {
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

// 转换为布尔值
func (tp *ThingProxy) toBool(value interface{}) (bool, error) {
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
		i, _ := v.Int64()
		return i != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// 转换为字符串
func (tp *ThingProxy) toString(value interface{}) (string, error) {
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
		// 尝试 JSON 序列化
		if bytes, err := json.Marshal(v); err == nil {
			return string(bytes), nil
		}
		return fmt.Sprintf("%v", v), nil
	}
}

// 转换为时间戳
func (tp *ThingProxy) toTime(value interface{}) (int64, error) {
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
		// 尝试解析时间字符串
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UnixMilli(), nil
		}
		if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
			return t.UnixMilli(), nil
		}
		// 尝试解析时间戳
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to time", value)
	}
}

// 转换为 JSON
func (tp *ThingProxy) toJSON(value interface{}) (interface{}, error) {
	// 如果已经是 map/slice，直接返回
	switch value.(type) {
	case map[string]interface{}, []interface{}:
		return value, nil
	}

	// 尝试序列化再反序列化，确保是有效的 JSON
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

// 验证属性值
func (tp *ThingProxy) validatePropertyValue(key string, tv *TypedValue, meta PropertyMeta) error {
	// 1. 类型验证
	if tv.Type != meta.Format {
		// 允许一些合理的类型转换
		if meta.Format == "float" && tv.Type == "int" {
			// int 可以转 float
		} else if meta.Format == "int" && tv.Type == "float" {
			// 检查 float 是否是整数
			floatVal, ok := tv.V.(float64)
			if !ok {
				return fmt.Errorf("expected float, got %T", tv.V)
			}
			if floatVal != float64(int64(floatVal)) {
				return fmt.Errorf("property '%s': expected int, got float with decimal: %v", key, floatVal)
			}
		} else {
			return fmt.Errorf("property '%s': type mismatch - expected %s, got %s",
				key, meta.Format, tv.Type)
		}
	}

	// 2. 范围验证
	if len(meta.Range) == 2 {
		switch tv.Type {
		case "int":
			if val, ok := tv.V.(int64); ok {
				if val < int64(meta.Range[0]) || val > int64(meta.Range[1]) {
					return fmt.Errorf("property '%s': value %d out of range [%v, %v]",
						key, val, meta.Range[0], meta.Range[1])
				}
			}
		case "float":
			if val, ok := tv.V.(float64); ok {
				if val < meta.Range[0] || val > meta.Range[1] {
					return fmt.Errorf("property '%s': value %v out of range [%v, %v]",
						key, val, meta.Range[0], meta.Range[1])
				}
			}
		}
	}

	// 3. 枚举验证
	if len(meta.Enum) > 0 {
		strVal := fmt.Sprintf("%v", tv.V)
		valid := false
		for _, e := range meta.Enum {
			if strVal == e {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("property '%s': value '%s' not in allowed values: %v",
				key, strVal, meta.Enum)
		}
	}

	return nil
}

// SetPropertyWithContext 带超时控制的设置属性
func (tp *ThingProxy) SetPropertyWithContext(ctx context.Context, deviceUUID string, key string, value interface{}) (*PropertyValue, error) {
	resultCh := make(chan *PropertyValue, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := tp.SetProperty(deviceUUID, key, value)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case result := <-resultCh:
		return result, nil
	}
}

// BatchSetProperties 批量设置属性
func (tp *ThingProxy) BatchSetProperties(deviceUUID string, properties map[string]interface{}) (map[string]*PropertyValue, error) {
	results := make(map[string]*PropertyValue)
	var lastError error

	for key, value := range properties {
		result, err := tp.SetProperty(deviceUUID, key, value)
		if err != nil {
			lastError = err
			// 继续处理其他属性，但记录错误
			results[key] = nil
		} else {
			results[key] = result
		}
	}

	if lastError != nil {
		return results, fmt.Errorf("batch set completed with errors: %w", lastError)
	}

	return results, nil
}
