package printer

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// PrettyPrint prints a struct (or any value) in a human-readable format,
// safely handling pointers (including nil pointers) and nested structs.
func PrettyPrint(v any) {
	fmt.Fprintln(os.Stdout, prettyPrintValue(v, 0, "  "))
}

// PrettyPrintIndent prints a struct (or any value) with a given indent prefix.
func PrettyPrintIndent(v any, indentPrefix string) {
	fmt.Fprintln(os.Stdout, prettyPrintValue(v, 0, indentPrefix))
}

func prettyPrintValue(v any, depth int, indentPrefix string) string {
	indent := strings.Repeat(indentPrefix, depth)
	if v == nil {
		return indent + "<nil>"
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			return indent + "<nil>"
		}
		return prettyPrintValue(val.Elem().Interface(), depth, indentPrefix)

	case reflect.Struct:
		return prettyPrintStruct(val, depth, indentPrefix)

	case reflect.Slice, reflect.Array:
		return prettyPrintSlice(val, depth, indentPrefix)

	case reflect.Map:
		return prettyPrintMap(val, depth, indentPrefix)

	case reflect.String:
		return indent + fmt.Sprintf("%q", val.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return indent + fmt.Sprintf("%d", val.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return indent + fmt.Sprintf("%d", val.Uint())

	case reflect.Float32, reflect.Float64:
		return indent + fmt.Sprintf("%f", val.Float())

	case reflect.Bool:
		return indent + fmt.Sprintf("%t", val.Bool())

	default:
		return indent + fmt.Sprintf("%v", val.Interface())
	}
}

func prettyPrintStruct(val reflect.Value, depth int, indentPrefix string) string {
	indent := strings.Repeat(indentPrefix, depth)
	typ := val.Type()
	lines := []string{}

	// If it's a well-known type, just fmt it
	if val.NumField() == 0 {
		return indent + fmt.Sprintf("%+v", val.Interface())
	}

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		fieldVal := val.Field(i)
		name := field.Name
		// Use json tag if present
		if tag := field.Tag.Get("json"); tag != "" {
			if tag != "-" {
				if parts := strings.Split(tag, ","); len(parts) > 0 && parts[0] != "" {
					name = parts[0]
				}
			}
		}
		lines = append(lines, indent+name+": "+prettyPrintValue(fieldVal.Interface(), depth+1, indentPrefix))
	}

	return strings.Join(lines, "\n")
}

func prettyPrintSlice(val reflect.Value, depth int, indentPrefix string) string {
	indent := strings.Repeat(indentPrefix, depth)
	if val.Len() == 0 {
		return indent + "[]"
	}

	// Check if it's a byte slice (string-like)
	if val.Type().Elem().Kind() == reflect.Uint8 {
		return indent + fmt.Sprintf("%q", val.Bytes())
	}

	lines := []string{indent + "["}
	for i := 0; i < val.Len(); i++ {
		lines = append(lines, prettyPrintValue(val.Index(i).Interface(), depth+1, indentPrefix))
	}
	lines = append(lines, indent+"]")
	return strings.Join(lines, "\n")
}

func prettyPrintMap(val reflect.Value, depth int, indentPrefix string) string {
	indent := strings.Repeat(indentPrefix, depth)
	if val.Len() == 0 {
		return indent + "map[string]any{}"
	}

	lines := []string{indent + "map[string]any{"}
	iter := val.MapRange()
	for iter.Next() {
		key := fmt.Sprintf("%v", iter.Key().Interface())
		lines = append(lines, indent+key+": "+prettyPrintValue(iter.Value().Interface(), depth+1, indentPrefix))
	}
	lines = append(lines, indent+"}")
	return strings.Join(lines, "\n")
}
