package utils

import (
	"fmt"
	"log/slog"
	"strings"
)

type StrMap = map[string]any

func GetMapFieldForced(input StrMap, fieldPath interface{}) any {
	if out, err := GetMapField(input, fieldPath); err != nil {
		slog.Debug(err.Error())
		return nil
	} else {
		return out
	}
}

func GetMapField(input StrMap, fieldPath interface{}) (any, error) {
	curr := input
	var field_names []string
	if _, ok := fieldPath.(string); ok {
		field_names = strings.Split(fieldPath.(string), "/")
	} else {
		field_names = fieldPath.([]string)
	}
	for index, field_name := range field_names {
		child := curr[field_name]
		if child == nil {
			return nil, nil
		} else if index == len(field_names)-1 {
			return child, nil
		}
		if childmap, ok := curr[field_name].(StrMap); ok {
			curr = childmap
		} else {
			return nil, fmt.Errorf("field_path (%s) at index %d is not a map", strings.Join(field_names, "/"), index)
		}
	}
	return nil, nil
}

/**
 * SetMapFields takes a map and a list of field paths and values and sets the values in the map
 * @param input the map to set the values in
 * @param field_paths_and_values a list of field paths and values.  The field paths are strings separated by "/" and the values are the values to set
 * @return error if there is an error at the first fieldpath that failed.
 */
func SetMapFields(input StrMap, field_paths_and_values ...interface{}) (StrMap, error) {
	var err error
	num_args := len(field_paths_and_values)
	for i := 0; i < num_args; i += 2 {
		field_path := field_paths_and_values[i]
		field_value := field_paths_and_values[i+1]
		input, err = SetMapField(input, field_path, field_value)
		if err != nil {
			return input, err
		}
	}
	return input, nil
}

/**
 * Sets a map field at a given field path ensuring that everything until the leaf is a dictionary indeed.
 */
func SetMapField(input StrMap, fieldPath interface{}, value interface{}) (StrMap, error) {
	if input == nil {
		input = make(StrMap)
	}
	var field_names []string
	if _, ok := fieldPath.(string); ok {
		field_names = strings.Split(fieldPath.(string), "/")
	} else {
		field_names = fieldPath.([]string)
	}
	curr := input
	for index, field_name := range field_names {
		if index == len(field_names)-1 {
			curr[field_name] = value
		} else {
			child := curr[field_name]
			if child == nil {
				curr[field_name] = make(StrMap)
			}
			if childmap, ok := curr[field_name].(StrMap); ok {
				curr = childmap
			} else {
				return input, fmt.Errorf("field_path (%s) at index %d is not a map", strings.Join(field_names, "/"), index)
			}
		}
	}
	return input, nil
}

/**
 * Copy a value in a given field path from the source into a fieldpath in the dest if
 * a. source field path is valid and exists
 * b. dest field path is valid (or needs to be created).
 */
func CopyMapFields(input StrMap, output StrMap, field_paths ...interface{}) (StrMap, error) {
	num_args := len(field_paths)
	for i := 0; i < num_args; i += 2 {
		src_field_path := field_paths[i]
		dst_field_path := field_paths[i+1]
		srcval, err := GetMapField(input, src_field_path)
		if err != nil {
			return output, err
		}
		if output, err = SetMapField(output, dst_field_path, srcval); err != nil {
			return output, err
		}
	}
	return output, nil
}
