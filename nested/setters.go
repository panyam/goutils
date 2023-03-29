package reflect

/*
import (
	// "errors"
	"errors"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// SetNestedFieldValue sets the value of a nested field in a protobuf message. If the intermediate
// fields along the path do not exist, default values will be created.
func SetNestedFieldValue(msg proto.Message, fieldPath []string, value interface{}) error {
	msgVal := reflect.ValueOf(msg)
	if msgVal.Kind() != reflect.Ptr || msgVal.IsNil() {
		return errors.New("message must be a non-nil pointer")
	}

	msgDesc := GetMessageDescriptor(proto.MessageName(msg))
	if msgDesc == nil {
		return errors.New("message descriptor not found")
	}

	fieldDesc, err := GetNestedFieldDescriptor(msgDesc, fieldPath)
	if err != nil {
		return err
	}

	if fieldDesc.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return fmt.Errorf("field '%s' is repeated, use AppendNestedFieldValue instead", fieldDesc.GetName())
	}

	// If the value is nil, set the field to its default value.
	if value == nil {
		value = reflect.New(reflect.TypeOf(fieldValue(fieldDesc))).Elem().Interface()
	}

	fieldValue, err := convertFieldValue(value, fieldDesc)
	if err != nil {
		return err
	}

	fieldVal := msgVal.Elem()
	for _, fieldName := range fieldPath[:len(fieldPath)-1] {
		fieldVal, err = createIntermediateField(fieldVal, fieldName)
		if err != nil {
			return err
		}
	}

	fieldVal.SetMapIndex(reflect.ValueOf(fieldPath[len(fieldPath)-1]), reflect.ValueOf(fieldValue))

	return nil
}

// createIntermediateField creates an intermediate field with the given name in a message or group.
// If the field already exists, its current value is returned. If it does not exist, a new field with
// the default value is created and returned.
func createIntermediateField(msgVal reflect.Value, fieldName string) (reflect.Value, error) {
	if msgVal.Kind() == reflect.Ptr {
		msgVal = msgVal.Elem()
	}

	fieldVal := msgVal.MapIndex(reflect.ValueOf(fieldName))
	if !fieldVal.IsValid() {
		msgType := msgVal.Type()
		msgDesc := getMessageDescriptor(msgType.Name())
		if msgDesc == nil {
			return reflect.Value{}, fmt.Errorf("message descriptor not found for type '%s'", msgType.Name())
		}

		fieldDesc := msgDesc.FindFieldByName(fieldName)
		if fieldDesc == nil {
			return reflect.Value{}, fmt.Errorf("field '%s' not found in message '%s'", fieldName, msgType.Name())
		}

		defaultValue := reflect.New(reflect.TypeOf(fieldValue(fieldDesc))).Elem()
		fieldVal = reflect.New(defaultValue.Type()).Elem()
		fieldVal.Set(defaultValue)
		msgVal.SetMapIndex(reflect.ValueOf(fieldName), fieldVal)
	}

	if fieldVal.Kind() == reflect.Ptr {
		fieldVal = fieldVal.Elem()
	}

	if fieldVal.Kind() != reflect.Map {
		return reflect.Value{}, fmt.Errorf("field '%s' is not a map", fieldName)
	}

	return fieldVal, nil
}
*/
