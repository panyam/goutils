package reflect

/*
import (
	// "errors"
	"errors"
	"fmt"
	"reflect"
	"strings"

	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"google.golang.org/protobuf/types/descriptorpb"
)

// getFieldDescriptor returns the field descriptor for the specified field name in the given message type.
func GetFieldDescriptor(msgType reflect.Type, fieldName string) (*descriptorpb.FieldDescriptorProto, error) {
	// Get the descriptor for the message type.
	msgDesc, err := descriptor.MessageDescriptorProtoFor(msgType.Elem())
	if err != nil {
		return nil, err
	}

	// Find the field descriptor for the specified field name.
	for _, fieldDesc := range msgDesc.GetField() {
		if fieldDesc.GetName() == fieldName {
			return fieldDesc, nil
		}
	}

	return nil, fmt.Errorf("unknown field '%s'", fieldName)
}

// getFieldDescriptor gets the field descriptor for a given nested field path in a message descriptor.
func GetNestedFieldDescriptor(msgDesc *descriptorpb.DescriptorProto, fieldPath []string) (*descriptorpb.FieldDescriptorProto, error) {
	if len(fieldPath) == 0 {
		return nil, errors.New("field path cannot be empty")
	}
	for _, fieldName := range fieldPath {
		fieldDesc := msgDesc.FindFieldByName(fieldName)
		if fieldDesc == nil {
			return nil, fmt.Errorf("field '%s' not found in message '%s'", fieldName, msgDesc.GetName())
		}

		// If the field is a message or group, get its descriptor.
		if fieldDesc.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE ||
			fieldDesc.GetType() == descriptorpb.FieldDescriptorProto_TYPE_GROUP {
			msgDesc = GetMessageDescriptor(fieldDesc.GetTypeName())
			if msgDesc == nil {
				return nil, fmt.Errorf("message descriptor not found for field '%s'", fieldDesc.GetName())
			}
		} else {
			// If the field is not a message or group, we've reached the end of the field path.
			return fieldDesc, nil
		}
	}

	// This should never happen.
	return nil, errors.New("invalid field path")
}

// GetMessageDescriptor gets the message descriptor for a given type name.
func GetMessageDescriptor(typeName string) *descriptorpb.DescriptorProto {
	parts := strings.Split(typeName, ".")
	if len(parts) < 2 {
		return nil
	}

	packageName := strings.Join(parts[:len(parts)-1], ".")
	messageName := parts[len(parts)-1]

	fileDesc, err := protoregistry.GlobalFiles.FindFileByPath(packageName + ".proto")
	if err != nil {
		return nil
	}

	msgDesc, err := fileDesc.FindMessage(messageName)
	if err != nil {
		return nil
	}

	return msgDesc
}

// getEnumDescriptor gets the enum descriptor for a given field descriptor.
func getEnumDescriptor(fieldDesc *descriptorpb.FieldDescriptorProto) (*descriptorpb.EnumDescriptorProto, error) {
	typeName := fieldDesc.GetTypeName()
	if !strings.HasPrefix(typeName, ".") {
		// The type name is not fully qualified, so we need to prepend the package name.
		msgDesc := fieldDesc.GetFile().GetMessageTypes()[0]
		typeName = "." + msgDesc.GetPackage() + "." + typeName
	}

	parts := strings.Split(typeName, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid enum type name '%s'", typeName)
	}

	packageName := strings.Join(parts[:len(parts)-1], ".")
	enumName := parts[len(parts)-1]

	fileDesc, err := protoregistry.GlobalFiles.FindFileByPath(packageName + ".proto")
	if err != nil {
		return nil, err
	}

	enumDesc, err := fileDesc.FindEnum(enumName)
	if err != nil {
		return nil, err
	}

	return enumDesc, nil
}
*/
