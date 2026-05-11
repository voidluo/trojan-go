package config

import (
	"context"
	"encoding/json"

	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

var creators = make(map[string]Creator)

// Creator creates default config struct for a module
type Creator func() any

// 用于全局配置清洗的接口
type normalizer interface {
	Normalize()
}

// RegisterConfigCreator registers a config struct for parsing
func RegisterConfigCreator(name string, creator Creator) {
	name += "_CONFIG"
	creators[name] = creator
}

func parseJSON(data []byte) (map[string]any, error) {
	result := make(map[string]any)
	for name, creator := range creators {
		config := creator()
		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
		normalizeConfig(config)
		result[name] = config
	}
	return result, nil
}

func parseYAML(data []byte) (map[string]any, error) {
	result := make(map[string]any)
	for name, creator := range creators {
		config := creator()
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
		normalizeConfig(config)
		result[name] = config
	}
	return result, nil
}

func WithJSONConfig(ctx context.Context, data []byte) (context.Context, error) {
	var configs map[string]any
	var err error
	configs, err = parseJSON(data)
	if err != nil {
		return ctx, err
	}
	for name, config := range configs {
		ctx = context.WithValue(ctx, name, config)
	}
	return ctx, nil
}

func WithYAMLConfig(ctx context.Context, data []byte) (context.Context, error) {
	var configs map[string]any
	var err error
	configs, err = parseYAML(data)
	if err != nil {
		return ctx, err
	}
	for name, config := range configs {
		ctx = context.WithValue(ctx, name, config)
	}
	return ctx, nil
}

func WithConfig(ctx context.Context, name string, cfg any) context.Context {
	name += "_CONFIG"
	return context.WithValue(ctx, name, cfg)
}

// FromContext extracts config from a context
func FromContext(ctx context.Context, name string) any {
	return ctx.Value(name + "_CONFIG")
}

// normalizeConfig 深度遍历结构体，强制补齐 WebSocket 路径
func normalizeConfig(cfg any) {
	if cfg == nil {
		return
	}
	val := reflect.ValueOf(cfg)
	normalizeValue(val)
}

func normalizeValue(val reflect.Value) {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldName := val.Type().Field(i).Name
			
			// 1. 如果是 Websocket 结构体，继续深入
			// 2. 如果字段名是 Path 且属于某个叫 Websocket 的结构或父级
			if fieldName == "Path" && field.Kind() == reflect.String {
				path := field.String()
				// 如果是管理面板路径且为空，给一个默认的特殊路径，避免与 Websocket 冲突
				structName := val.Type().Name()
				if path == "" && structName == "AdminConfig" {
					path = "/trojan-go-admin/"
				}
				if !strings.HasPrefix(path, "/") {
					path = "/" + path
				}
				// 对于管理面板这类挂载路径，强制以 / 结尾以配合重定向逻辑
				if structName == "AdminConfig" && !strings.HasSuffix(path, "/") {
					path += "/"
				}
				if field.CanSet() {
					field.SetString(path)
				}
			}
			
			// 递归处理子字段
			if field.Kind() == reflect.Struct || field.Kind() == reflect.Ptr {
				normalizeValue(field)
			}
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			normalizeValue(val.MapIndex(key))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			normalizeValue(val.Index(i))
		}
	}
}
