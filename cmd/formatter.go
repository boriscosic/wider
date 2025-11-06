package main

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"strings"
)

func formatAge(t metav1.Time) string {
	duration := metav1.Now().Sub(t.Time)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds())

	// Show days only if >= 2 days
	if days >= 2 {
		return fmt.Sprintf("%dd", days)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", seconds)
}

func getValueByPath(pn PodWithWider, path string) (string, error) {
	// Remove leading dot if present
	path = strings.TrimPrefix(path, ".")

	// Split by dots, but respect escaped dots
	parts := splitPath(path)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty path")
	}

	var current interface{}

	switch parts[0] {
	case "pod":
		current = pn.Pod
		parts = parts[1:]
	case "node":
		if pn.Node == nil {
			return "<none>", nil
		}
		current = pn.Node
		parts = parts[1:]
	case "serviceAccount", "sa":
		if pn.ServiceAccount == nil {
			return "<none>", nil
		}
		current = pn.ServiceAccount
		parts = parts[1:]
	case "pvcs", "pvc":
		if len(pn.PVCs) == 0 {
			return "<none>", nil
		}
		// For PVCs array, return comma-separated names or allow indexing
		if len(parts) == 1 {
			names := []string{}
			for _, pvc := range pn.PVCs {
				names = append(names, pvc.Name)
			}
			return strings.Join(names, ","), nil
		}
		// TODO: Could add array indexing support like pvcs[0].name
		current = pn.PVCs
		parts = parts[1:]
	default:
		return "", fmt.Errorf("path must start with 'pod' or 'node', got: %s", parts[0])
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%v", current), nil
	}

	for i, part := range parts {
		if part == "" {
			continue
		}

		val := reflect.ValueOf(current)

		// Handle pointers
		for val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return "<none>", nil
			}
			val = val.Elem()
		}

		if !val.IsValid() {
			return "", fmt.Errorf("invalid value at part %d (%s)", i, part)
		}

		// Handle map access (e.g., labels[key])
		if val.Kind() == reflect.Map {
			key := reflect.ValueOf(part)
			mapVal := val.MapIndex(key)
			if !mapVal.IsValid() {
				return "<none>", nil
			}
			current = mapVal.Interface()
			continue
		}

		if val.Kind() != reflect.Struct {
			return "", fmt.Errorf("cannot access field %s on non-struct type %v", part, val.Kind())
		}

		// Try to find field by JSON tag first, then by capitalized name
		field := findFieldByJSONTag(val, part)

		if !field.IsValid() {
			// Fallback to capitalized field name
			fieldName := capitalizeFirst(part)
			field = val.FieldByName(fieldName)
		}

		if !field.IsValid() {
			return "", fmt.Errorf("field %s not found", part)
		}

		current = field.Interface()
	}

	return fmt.Sprintf("%v", current), nil
}

func splitPath(path string) []string {
	var parts []string
	var current strings.Builder
	escaped := false

	for i := 0; i < len(path); i++ {
		if escaped {
			// Add the character after backslash literally
			current.WriteByte(path[i])
			escaped = false
		} else if path[i] == '\\' {
			// Next character is escaped
			escaped = true
		} else if path[i] == '.' {
			// Unescaped dot is a separator
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(path[i])
		}
	}

	// Add the last part
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func findFieldByJSONTag(val reflect.Value, tagName string) reflect.Value {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}
		// JSON tag format: "name,omitempty" or just "name"
		tagParts := strings.Split(jsonTag, ",")
		if len(tagParts) > 0 && tagParts[0] == tagName {
			return val.Field(i)
		}
	}
	return reflect.Value{}
}

func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
