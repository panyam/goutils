package reflect

import (
	// "errors"
	"fmt"
	"strings"
	// "reflect"
	// "github.com/golang/protobuf/proto"
	// descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	// "google.golang.org/protobuf/types/descriptorpb"
	// "google.golang.org/protobuf/types/known/structpb"
)

type StringMap = map[string]interface{}

func GetMapField[T any](input StringMap, fieldPath interface{}) (interface{}, error) {
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
			return child.(T), nil
		}
		if childmap, ok := curr[field_name].(StringMap); ok {
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
func SetMapFields(input StringMap, field_paths_and_values ...interface{}) error {
	num_args := len(field_paths_and_values)
	for i := 0; i < num_args; i += 2 {
		field_path := field_paths_and_values[i]
		field_value := field_paths_and_values[i+1]
		err := SetMapField(input, field_path, field_value)
		if err != nil {
			return err
		}
	}
	return nil
}

func SetMapField(input StringMap, fieldPath interface{}, value interface{}) error {
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
				curr[field_name] = make(StringMap)
			}
			if childmap, ok := curr[field_name].(StringMap); ok {
				curr = childmap
			} else {
				return fmt.Errorf("field_path (%s) at index %d is not a map", strings.Join(field_names, "/"), index)
			}
		}
	}
	return nil
}
