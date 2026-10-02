package log

import (
	"fmt"
	"reflect"
	"time"
)

func normalizeFields(args []any) []any {
	fields := make([]any, 0, len(args)+len(args)%2)

	for i := 0; i < len(args); {
		switch arg := args[i].(type) {
		case Attr:
			fields = append(fields, arg.Key, normalizeValue(arg.Value))
			i++
		case string:
			if i+1 >= len(args) {
				fields = append(fields, BadKey, arg)
				i++
				continue
			}
			fields = append(fields, arg, normalizeValue(args[i+1]))
			i += 2
		default:
			fields = append(fields, BadKey, normalizeValue(arg))
			i++
		}
	}

	return fields
}

func normalizeValue(v any) any {
	switch value := v.(type) {
	case nil, string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, []byte, error, time.Time, time.Duration:
		return value
	case fmt.Stringer:
		if rv := reflect.ValueOf(value); rv.Kind() == reflect.Pointer && rv.IsNil() {
			return nil
		}
		return value.String()
	}
	return v
}
