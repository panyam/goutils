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

func GetMapField(input StringMap, field_names []string) (interface{}, error) {
	curr := input
	for index, field_name := range field_names {
		child := curr[field_name]
		if child == nil {
			return nil, nil
		}
		if childmap, ok := curr[field_name].(StringMap); ok {
			curr = childmap
		} else {
			return nil, fmt.Errorf("field_path (%s) at index %d is not a map", strings.Join(field_names, "/"), index)
		}
	}
	return curr, nil
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
		field_path := field_paths_and_values[i].(string)
		field_value := field_paths_and_values[i+1]
		field_names := strings.Split(field_path, "/")
		err := SetMapField(input, field_names, field_value)
		if err != nil {
			return err
		}
	}
	return nil
}

func SetMapField(input StringMap, field_names []string, value interface{}) error {
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

/*
// Given a nested struct, GetNestedFieldValue returns the value of a specified nested
// field in the given struct.  The field path is a list of components that may have
// been unsplit (eg as field1.field2.field3...)
func GetNestedFieldValue(msg proto.Message, fieldPath string) (interface{}, error) {
	// Split the field path into individual field names.
	fieldNames := strings.Split(fieldPath, ".")

	// Start with the top-level message.
	curMsg := reflect.ValueOf(msg)

	// Traverse the nested fields.
	for _, fieldName := range fieldNames {
		// Get the current field's value.
		curField := curMsg.Elem().FieldByName(fieldName)

		// If the field is a pointer, dereference it.
		if curField.Kind() == reflect.Ptr {
			if curField.IsNil() {
				return nil, fmt.Errorf("nested field '%s' is nil", fieldName)
			}
			curField = curField.Elem()
		}

		// Check if the current field is valid.
		if !curField.IsValid() {
			return nil, fmt.Errorf("invalid field '%s'", fieldName)
		}

		// Update the current message to the nested field.
		curMsg = curField
	}

	// Return the final value of the field.
	return curMsg.Interface(), nil
}

// convertValue converts the given value to the appropriate type for the specified field descriptor.
func convertValue(fieldDesc *descriptorpb.FieldDescriptorProto, value interface{}) (interface{}, error) {
	switch fieldDesc.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		v, ok := value.(bool)
		if !ok {
			return nil, errors.New("must be a boolean value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		v, ok := value.(float64)
		if !ok {
			return nil, errors.New("must be a double value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		v, ok := value.(float32)
		if !ok {
			return nil, errors.New("must be a float value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		v, ok := value.(int32)
		if !ok {
			return nil, errors.New("must be a 32-bit integer value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		v, ok := value.(int64)
		if !ok {
			return nil, errors.New("must be a 64-bit integer value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		v, ok := value.(uint32)
		if !ok {
			return nil, errors.New("must be a positive 32-bit integer value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		v, ok := value.(uint64)
		if !ok {
			return nil, errors.New("must be a positive 64-bit integer value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		v, ok := value.(string)
		if !ok {
			return nil, errors.New("must be a string value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		v, ok := value.([]byte)
		if !ok {
			return nil, errors.New("must be a byte slice value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		// If the value is a string, look up the corresponding enum value by name.
		if v, ok := value.(string); ok {
			enumDesc, err := getEnumDescriptor(fieldDesc)
			if err != nil {
				return nil, err
			}
			for _, enumVal := range enumDesc.GetValue() {
				if enumVal.GetName() == v {
					return enumVal.GetNumber(), nil
				}
			}
			return nil, fmt.Errorf("unknown enum value '%s'", v)
		}

		// Otherwise, assume the value is already an integer.
		v, ok := value.(int32)
		if !ok {
			return nil, errors.New("must be an integer value")
		}
		return v, nil

	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		// Convert the value to a proto.Message.
		v, ok := value.(proto.Message)
		if !ok {
			return nil, errors.New("must be a message value")
		}

		// If the field is a google.protobuf.Struct, we need to convert it to the appropriate type.
		if fieldDesc.GetTypeName() == ".google.protobuf.Struct" {
			structValue, err := structpb.NewStruct(v)
			if err != nil {
				return nil, err
			}
			return structValue, nil
		}

		// Otherwise, assume the value is already the appropriate type.
		return v, nil

	default:
		return nil, fmt.Errorf("unsupported field type %v", fieldDesc.GetType())
	}
}
*/
