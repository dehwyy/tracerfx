package dspan

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Attribute struct {
	Key   string
	Value any
}

func Attr(key string, value any) Attribute {
	return Attribute{key, value}
}

func extractFields(key string, value any) map[string]any {
	result := make(map[string]any)

	val := reflect.ValueOf(value)
	if !val.IsValid() {
		return result
	}

	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			result[key] = "<nil>"
			return result
		}
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		typ := val.Type()
		added := false
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}

			fieldKey := key + "." + field.Name
			if key == "" {
				fieldKey = field.Name
			}

			fieldVal := val.Field(i).Interface()

			subVal := reflect.ValueOf(fieldVal)
			if (subVal.Kind() == reflect.Struct) || (subVal.Kind() == reflect.Ptr && !subVal.IsNil() && subVal.Elem().Kind() == reflect.Struct) {
				subFields := extractFields(fieldKey, fieldVal)
				for k, v := range subFields {
					result[k] = v
					added = true
				}
			} else {
				result[fieldKey] = fieldVal
				added = true
			}
		}
		if !added {
			result[key] = fmt.Sprintf("%T(%+v)", value, value)
		}
	} else {
		result[key] = value
	}

	return result
}

func setAttr(span trace.Span, key string, val any) {
	switch value := val.(type) {
	case int:
		span.SetAttributes(attribute.Int(key, value))
	case int64:
		span.SetAttributes(attribute.Int64(key, value))
	case float64:
		span.SetAttributes(attribute.Float64(key, value))
	case string:
		span.SetAttributes(attribute.String(key, value))
	case bool:
		span.SetAttributes(attribute.Bool(key, value))
	default:
		span.SetAttributes(attribute.String(key, fmt.Sprintf("%+v", val)))
	}
}
