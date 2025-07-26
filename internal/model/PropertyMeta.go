package model

import (
	"encoding/json"
	"reflect"
	"strings"
)

type PropertyMeta struct {
	Writable    bool
	Description string
	Unit        string
	Range       []int
	Format      string
	Enum        []string
	//TODO Required Type ?
}

func parseMeta(metaStr string) map[string]string {
	meta := make(map[string]string)
	if metaStr == "" {
		return meta
	}
	pairs := strings.Split(metaStr, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			meta[kv[0]] = kv[1]
		}
	}
	return meta
}

func ExtractPropertyMeta(p interface{}) map[string]PropertyMeta {
	t := reflect.TypeOf(p).Elem()
	//v := reflect.ValueOf(p).Elem()
	metaMap := make(map[string]PropertyMeta)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		metaStr := field.Tag.Get("meta")

		if tag == "" {
			continue
		}

		meta := parseMeta(metaStr)
		propertyMeta := PropertyMeta{
			Writable:    meta["writable"] == "true",
			Description: meta["description"],
			Unit:        meta["unit"],
			Format:      meta["format"],
		}

		if rangeVal, ok := meta["range"]; ok {
			var r []int
			if err := json.Unmarshal([]byte(rangeVal), &r); err == nil {
				propertyMeta.Range = r
			}
		}

		if enumVal, ok := meta["enum"]; ok {
			var e []string
			if err := json.Unmarshal([]byte(enumVal), &e); err == nil {
				propertyMeta.Enum = e
			}
		}

		metaMap[tag] = propertyMeta
	}

	return metaMap
}
