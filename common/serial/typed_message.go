package serial

import (
	//"errors"
	"fmt"
	"reflect"
	"sync"

	//"example.com/gztest"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoiface"
	"google.golang.org/protobuf/runtime/protoimpl"
)

// GetInstance converts current TypedMessage into a proto Message.
// GetInstance 将当前 TypedMessage 转换为 proto Message。
func (v *TypedMessage) GetInstance() (proto.Message, error) {
	//mType := v.ProtoReflect()
	//fmt.Println("in common-serial-typed_message.go func (v *TypedMessage) GetInstance()")
	//gztest.GetMessageReflectType(v)
	instance, err := GetInstance(v.Type)
	//fmt.Println("in common-serial-typed_message.go func (v *TypedMessage) GetInstance() instance is:")
	//gztest.GetMessageReflectType(instance)
	if err != nil {
		fmt.Println("in common-serial-typed_message.go func (v *TypedMessage) GetInstance() GetInstance(v.Type)==err")
		return nil, err
	}
	protoMessage := instance.(proto.Message)
	//if protoMessage, ok := instance.(proto.Message); ok {
		//fmt.Println("in typed_message instance.(proto.Message) ok")
	//	PrintMessageDetails(protoMessage)
	//}
	if err := proto.Unmarshal(v.Value, protoMessage); err != nil {
		fmt.Println("in typed_message instance.(proto.Message) ")
		return nil, err
	}
	
	return protoMessage, nil
}

//func PrintMessageDetails(msg protoreflect.ProtoMessage) {
//	desc := msg.ProtoReflect().Descriptor()

//	for i := 0; i < desc.Fields().Len(); i++ {
			//range_ := desc.Fields().Get(i)
			//for j := range_.Start; j <= range_().End; j++ {
//					field := desc.Fields().Get(i)
//					fmt.Printf("Field Name: %s, Number: %d, Kind: %s, Value: %v\n",
//							field.Name(), field.Number(), field.Kind(), msg.ProtoReflect().Get(field))
			//}
//	}
//}

// GetInstance creates a new instance of the message with messageType.
// GetInstance 使用 messageType 创建消息的新实例
func GetInstance(messageType string) (interface{}, error) {
	//fmt.Println("in common-serial-typed_message.go func GetInstance(messageType string) messageType is")
	//gztest.GetMessageReflectType(messageType)
	if v, ok := messageTypeCache.Load(messageType); ok {
		fmt.Println("in common-serial-typed_message.go func GetInstance(messageType string) messageTypeCache.Load ok")
		return reflect.New(v.(reflect.Type).Elem()).Interface(), nil
	}
	mType, _ := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(messageType))
	if mType == nil {
		d, _ := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(messageType))
		if md, _ := d.(protoreflect.MessageDescriptor); md != nil && md.IsMapEntry() {
			kt := goTypeForField(md.Fields().ByNumber(1))
			vt := goTypeForField(md.Fields().ByNumber(2))
			t := reflect.MapOf(kt, vt)
			return reflect.New(t.Elem()).Interface(), nil
		}
	}
	
	msg := messageTypeToReflectType(mType)
	//fmt.Println("in GetInstance(messageType string) msg is: ", msg)
	return reflect.New(msg.Elem()).Interface(), nil
}

func messageTypeToReflectType(msgType protoreflect.MessageType) reflect.Type {
	// 创建消息实例
	msg := msgType.New().Interface()
	// 获取消息实例的类型
	return reflect.TypeOf(msg)
}

// ToTypedMessage converts a proto Message into TypedMessage.
func ToTypedMessage(message proto.Message) *TypedMessage {
	//fmt.Println("in common-serial-typed_message.go func ToTypeMMessage()")
	if message == nil {
		fmt.Println("in common-serial-typed_message.go func ToTypeMMessage() message == nil ")
		return nil
	}
	// 将结构体序列化为字节
	settings, _ := proto.Marshal(message)
	//fmt.Println("in common-serial-typed_message.go func ToTypeMMessage() settings: ", string(settings))
	//fmt.Println("in common-serial-typed_message.go func ToTypeMMessage() Type:: ", ProtoMessageToString(message))
	// type 为 protomessage 格式的名称
	return &TypedMessage{
		Type:  ProtoMessageToString(message),
		Value: settings,
	}
}

// GetMessageType returns the name of this proto Message.
//func GetMessageType(message proto.Message) string {
//	return proto.MessageName(message)
//}

func ProtoMessageToString(message proto.Message) string {
	desc := message.ProtoReflect().Descriptor()
	return string(desc.FullName())
	//data, err := proto.Marshal(message)
	//if err != nil {
	//	return ""
	//}
	//return string(data)
}

func goTypeForField(fd protoreflect.FieldDescriptor) reflect.Type {
	switch k := fd.Kind(); k {
	case protoreflect.EnumKind:
		if et, _ := protoregistry.GlobalTypes.FindEnumByName(fd.Enum().FullName()); et != nil {
			return enumGoType(et)
		}
		return reflect.TypeOf(protoreflect.EnumNumber(0))
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if mt, _ := protoregistry.GlobalTypes.FindMessageByName(fd.Message().FullName()); mt != nil {
			return messageGoType(mt)
		}
		return reflect.TypeOf((*protoreflect.Message)(nil)).Elem()
	default:
		return reflect.TypeOf(fd.Default().Interface())
	}
}

func enumGoType(et protoreflect.EnumType) reflect.Type {
	return reflect.TypeOf(et.New(0))
}

var messageTypeCache sync.Map // map[messageName]reflect.Type

//func MessageType(s string) reflect.Type {
//	if v, ok := messageTypeCache.Load(s); ok {
//		return v.(reflect.Type)
//	}

// Derive the message type from the v2 registry.
//var t reflect.Type
//if mt, _ := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(s)); mt != nil {
//	t = messageGoType(mt)
//}

// If we could not get a concrete type, it is possible that it is a
// pseudo-message for a map entry.
//	if t == nil {
//		d, _ := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(s))
//		if md, _ := d.(protoreflect.MessageDescriptor); md != nil && md.IsMapEntry() {
//			kt := goTypeForField(md.Fields().ByNumber(1))
//			vt := goTypeForField(md.Fields().ByNumber(2))
//			t = reflect.MapOf(kt, vt)
//		}
//	}

// Locally cache the message type for the given name.
//	if t != nil {
//		v, _ := messageTypeCache.LoadOrStore(s, t)
//		return v.(reflect.Type)
//	}
//	return nil
//}

func messageGoType(mt protoreflect.MessageType) reflect.Type {
	return reflect.TypeOf(MessageV1(mt.Zero().Interface()))
}

func MessageV1(m GeneratedMessage) protoiface.MessageV1 {
	return protoimpl.X.ProtoMessageV1Of(m)
}

type GeneratedMessage interface{}
