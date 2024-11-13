package common

import (
	"context"
	"reflect"
	"errors"
	"fmt"
	
)

// CreateObject creates an object by its config. The config type must be registered through RegisterConfig().
// CreateObject 根据其配置创建一个对象。配置类型必须通过 RegisterConfig() 注册。
func CreateObject(ctx context.Context, config interface{}) (interface{}, error) {
	// 包reflect实现了运行时反射，允许程序操作任意类型的对象。典型的用途是使用静态类型interface{}获取一个值，并通过调用TypeOf（返回一个Type）来提取其动态类型信息。
	configType := reflect.TypeOf(config)
	//fmt.Println("in common-type.go func CreateObject configType: ", configType)
	
	creator, found := typeCreatorRegistry[configType]
	if !found {
		fmt.Println("in common-type.go func CreateObject !found")
		return nil, errors.New(configType.String() + " is not registered")
	}

	//fmt.Println("in common-type.go func CreateObject return")
	return creator(ctx, config)
}

// ConfigCreator is a function to create an object by a config.
// ConfigCreator 是一个通过配置创建对象的函数。
type ConfigCreator func(ctx context.Context, config interface{}) (interface{}, error)

var (
	typeCreatorRegistry = make(map[reflect.Type]ConfigCreator)
)

// RegisterConfig registers a global config creator. The config can be nil but must have a type.
// RegisterConfig 注册一个全局配置创建者。配置可以为 nil，但必须有一个类型
func RegisterConfig(config interface{}, configCreator ConfigCreator) error {
	
	configType := reflect.TypeOf(config)
	//fmt.Println("in common-type.go func RegisterConfig : ", configType)
	if _, found := typeCreatorRegistry[configType]; found {
		return errors.New(configType.Name() + " is already registered")
	}
	typeCreatorRegistry[configType] = configCreator
	return nil
}