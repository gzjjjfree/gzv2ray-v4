//go:build !confonly
// +build !confonly

package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"example.com/gztest"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/features"
	"github.com/gzjjjfree/gzv2ray-v4/features/dns"
	"github.com/gzjjjfree/gzv2ray-v4/features/dns/localdns"
	"github.com/gzjjjfree/gzv2ray-v4/features/inbound"
	"github.com/gzjjjfree/gzv2ray-v4/features/outbound"
	"github.com/gzjjjfree/gzv2ray-v4/features/policy"
	"github.com/gzjjjfree/gzv2ray-v4/features/routing"
	"github.com/gzjjjfree/gzv2ray-v4/features/stats"
)

// Server is an instance of V2Ray. At any time, there must be at most one Server instance running.
type Server interface { //Server 是 V2Ray 的一个实例，任何时候都最多只能有一个 Server 实例在运行。
	common.Runnable
}

// ServerType returns the type of the server.
// ServerType 返回服务器的类型。
func ServerType() interface{} {
	return (*Instance)(nil)
}

// Start starts the V2Ray instance, including all registered features. When Start returns error, the state of the instance is unknown.
// A V2Ray instance can be started only once. Upon closing, the instance is not guaranteed to start again.
// Start 启动 V2Ray 实例，包括所有已注册的功能。当 Start 返回错误时，实例的状态为未知。
// V2Ray 实例只能启动一次。关闭后，实例不保证再次启动。
// v2ray:api:stable
func (s *Instance) Start() error {
	fmt.Println("in gzv2ray.go func (s *Instance) Start()")
	s.access.Lock()
	defer s.access.Unlock()

	s.running = true
	for _, f := range s.features {
		fmt.Println("in gzv2ray.go func (s *Instance) Start() : ", reflect.TypeOf(f))
		if err := f.Start(); err != nil {
			return err
		}
	}

	fmt.Println("V2Ray ", Version(), " started")

	return nil
}

// New returns a new V2Ray instance based on given configuration.
// New 根据给定的配置返回一个新的 V2Ray 实例。
// The instance is not started at this point.
// 此时实例尚未启动。
// To ensure V2Ray instance works properly, the config must contain one Dispatcher, one InboundHandlerManager and one OutboundHandlerManager. Other features are optional.
// 为了确保 V2Ray 实例正常运行，配置必须包含一个 Dispatcher、一个 InboundHandlerManager 和一个 OutboundHandlerManager。其他功能是可选的。
func New(config *Config) (*Instance, error) {
	fmt.Println("in gzv2ray.go func New")
	var server = &Instance{ctx: context.Background()} // Instance 实例 结合了 V2Ray 中的所有功能
	// Background 返回一个非零的空 [Context]。它永远不会被取消，没有值，也没有截止期限。它通常由主函数、初始化和测试使用，并作为传入请求的顶级 Context。
if server.featureResolutions == nil {
	fmt.Println("in gzv2ray.go func New server.featureResolutions == nil")
}
	done, err := initInstanceWithConfig(config, server) // 函数参数为 配置文件和一个空的 Context，返回一个布尔值和错误
	if done {                                           // config 配置不正确时，done 为真，返回空及错误
		fmt.Println("in gzv2ray.go New err is: ", err)
		return nil, err
	}
	//fmt.Println("in gzv2ray.go func New return")
	return server, nil // config 配置正确时，返回sever
}

// Instance combines all functionalities in V2Ray.
// 实例 结合了 V2Ray 中的所有功能。
type Instance struct {
	access             sync.Mutex         // Mutex 是一种互斥锁。Mutex 的零值表示未锁定的互斥锁。首次使用后不得复制 Mutex。
	features           []features.Feature // {common.HasType common.Runnable} Runnable 是可以根据需要开始工作和停止的对象的接口。HasType 是知道其类型的对象的接口
	featureResolutions []resolution       // {deps []reflect.Type callback interface{}} Type 是 Go 类型的表示 callback 一个接口
	running            bool

	ctx context.Context // Context 类型，它携带跨 API 边界和进程之间的截止日期、取消信号和其他请求范围的值
}

type resolution struct {
	deps     []reflect.Type
	callback interface{}
}

func initInstanceWithConfig(config *Config, server *Instance) (bool, error) {
	if config.Transport != nil { // 当有全局设置时
		fmt.Println("in gzv2ray config.Transport != nil ")
		features.PrintDeprecatedFeatureWarning("global transport settings") // 打印有关已弃用功能的警告。
	}
	if err := config.Transport.Apply(); err != nil { // Apply 应用全局设置，能设置时，返回服务器设置错误为真
		fmt.Println("in gzv2ray config.Transport.Apply() ")
		return true, err
	}

	for _, appSettings := range config.App { // 读取每个 App 设置
		//fmt.Println("in gzv2ray.go appSettings  ")
		//gztest.GetMessageReflectType(appSettings)
		// 将字段 App 类型 []*serial.TypedMessage 转化为 protoreflect.ProtoMessage 类型
		settings, err := appSettings.GetInstance()
		if err != nil { // 如果返回不是没有错误，返回服务器设置错误为真
			fmt.Println("in gzv2ray.go appSettings.GetInstance() ")
			return true, err
		}

		//fmt.Println("in gzv2ray.go obj  ")
		// 根据已注册的 protoreflect.ProtoMessage 类型初始化为一个接口
		obj, err := CreateObject(server, settings)
		if err != nil {
			fmt.Println("in gzv2ray CreateObject(server, settings) ")
			return true, err
		}
		fmt.Println("in gzv2ray.go feature  ")
		// 检查接口是否实现 v2ray 的实例类型
		if feature, ok := obj.(features.Feature); ok {
			// fmt.Println("in gzv2ray.go feature ok ")
			if err := server.AddFeature(feature); err != nil {
				fmt.Println("in gzv2ray server.AddFeature(feature) ")
				return true, err
			}
		}
	}
	//fmt.Println("in gzv2ray end APP set ")
	// 创建以下4个类型的默认设置
	essentialFeatures := []struct {
		Type     interface{}
		Instance features.Feature
	}{
		{dns.ClientType(), localdns.New()},              // dns 类型
		{policy.ManagerType(), policy.DefaultManager{}}, // 本地策略
		{routing.RouterType(), routing.DefaultRouter{}}, // 路由
		{stats.ManagerType(), stats.NoopManager{}},      // 信息统计
	}
	// 无此设置就添加默认设置
	for _, f := range essentialFeatures {
		if server.GetFeature(f.Type) == nil {
			if err := server.AddFeature(f.Instance); err != nil {
				return true, err
			}
		}
	}
	//
	if server.featureResolutions != nil {
		return true, errors.New("not all dependency are resolved")
	}
	// fmt.Println("in gzv2ray.go addInboundHandlers")
	if err := addInboundHandlers(server, config.Inbound); err != nil {
		return true, err
	}
	//fmt.Println("in gzv2ray.go addOutboundHandlers")
	if err := addOutboundHandlers(server, config.Outbound); err != nil {
		return true, err
	}
	//fmt.Println("in gzv2ray.go initInstanceWithConfig return")
	return false, nil
}

// Type implements common.HasType.类型实现 common.HasType。
func (s *Instance) Type() interface{} {
	return ServerType()
}

// Close shutdown the V2Ray instance.关闭 V2Ray 实例。
func (s *Instance) Close() error {
	fmt.Println("in gzv2ray.go func (s *Instance) Close()")
	s.access.Lock()
	defer s.access.Unlock()

	s.running = false

	var errorsmsg []interface{}
	for _, f := range s.features {
		if err := f.Close(); err != nil {
			errorsmsg = append(errorsmsg, err)
		}
	}
	if len(errorsmsg) > 0 {
		return errors.New("failed to close all features")
	}

	return nil
}

// AddFeature registers a feature into current Instance.
// AddFeature 将一个功能注册到当前实例中。
func (s *Instance) AddFeature(feature features.Feature) error {
	fmt.Println("in gzv2ray.go func (s *Instance) AddFeature(feature features.Feature) : ", reflect.TypeOf(feature.Type()))
	s.features = append(s.features, feature) // 内置函数 append 将元素附加到切片的末尾，有必要存储 append 的结果，通常存储在保存切片本身的变量中

	if s.running { // 开始时 running 应该是 false，如果实例已运行，检查功能是否错误
		if err := feature.Start(); err != nil { // feature.Start 返回有错误的时
			fmt.Println("failed to start feature")
		}
		return nil
	}
	// 确认 server featureResolutions 是否存在，存在时需要执行功能类型判断
	if s.featureResolutions == nil {
		return nil
	}
	//fmt.Println("in gzv2ray.go func (s *Instance) AddFeature(feature features.Feature) s.featureResolutions != nil ")
	var pendingResolutions []resolution
	for _, r := range s.featureResolutions {
		finished, err := r.resolve(s.features)
		if finished && err != nil {
			return err
		}
		// 
		if !finished {
			//fmt.Println("in gzv2ray.go func (s *Instance) AddFeature(feature features.Feature) !finished ")
			pendingResolutions = append(pendingResolutions, r) 
		}
	}
	if len(pendingResolutions) == 0 {
		//fmt.Println("in gzv2ray.go func (s *Instance) AddFeature(feature features.Feature) s.featureResolutions = nil ")
		s.featureResolutions = nil
	} else if len(pendingResolutions) < len(s.featureResolutions) { 
		s.featureResolutions = pendingResolutions
	}

	return nil
}

func (r *resolution) resolve(allFeatures []features.Feature) (bool, error) { // resoleve 解析接口
	fmt.Println("in gzv2ray.go func (r *resolution) resolve")
	//gztest.GetMessageReflectType(r.deps)
	var fs []features.Feature
	// r 是  Feature 类型列表及回调函数
	for _, d := range r.deps {
		// 在功能中查找 deps 类型匹配
		f := getFeature(allFeatures, d)
		// 找到最后的参数类型 *stats.Manager 后，f 才会 != nill 才会执行后面的代码
		if f == nil { // 当无匹配时，返回
			//fmt.Println("in gzv2ray.go func (r *resolution) resolve f == nil")
			return false, nil
		}
		//fmt.Println("in gzv2ray.go func (r *resolution) resolve fs = append(fs, f)")
		// 把找到已注册的需要功能汇总到 fs 
		fs = append(fs, f) // 当匹配时，汇总到变量 fs
	}
	//fmt.Println("in gzv2ray.go func (r *resolution) resolve callback := reflect.ValueOf(r.callback) len(fs): ", len(fs))
	// 
	callback := reflect.ValueOf(r.callback) // ValueOf 返回一个新的值，该值初始化为接口 i 中存储的具体值。 ValueOf(nil) 返回零值。
	var input []reflect.Value
	// 需要 Feature 类型的列表
	callbackType := callback.Type() 
	for i := 0; i < callbackType.NumIn(); i++ { // NumIn 返回函数类型的输入参数数量。如果类型的 Kind 不是 Func，则会引起混乱。
		pt := callbackType.In(i) // In 返回函数类型的第 i 个输入参数的类型。如果类型的 Kind 不是 Func，则会引起混乱。如果 i 不在 [0, NumIn()) 范围内，则会引起混乱。
		for _, f := range fs {
			//fmt.Println("in gzv2ray.go func (r *resolution) resolve f := range fs i: ", i, " ", reflect.TypeOf(f.Type()))
			// 判定已注册的需求功能是否能赋值给回调函数当参数
			if reflect.TypeOf(f).AssignableTo(pt) { // AssignableTo 报告该类型的值是否可以分配给类型 u
				input = append(input, reflect.ValueOf(f)) // 把匹配的  具体值添加到 input
				break // break 从头轮询，是因为有可能回调函数的参数有同类别，所以要再次轮询 ?
			}
		}
	}
	//fmt.Println("in gzv2ray.go func (r *resolution) resolve len(input)： ", len(input), callbackType.NumIn())
	if len(input) != callbackType.NumIn() {
		panic("Can't get all input parameters") // 内置函数 panic 会停止当前 goroutine 的正常执行。
	}

	var err error
	ret := callback.Call(input)                          // Call 使用输入参数 in 调用函数 v。例如，如果 len(in) == 3，则 v.Call(in) 表示 Go 调用 v(in[0], in[1], in[2])。input 就是参数列表
	errInterface := reflect.TypeOf((*error)(nil)).Elem() // TypeOf 返回表示 i 的动态类型的反射 [Type]。如果 i 是 nil 接口值，则 TypeOf 返回 nil。Elem 返回类型的元素类型。
	for i := len(ret) - 1; i >= 0; i-- {
		if ret[i].Type() == errInterface { // ret.Type 有错误或空时，检查回调返回值是否有错误
			v := ret[i].Interface() // 接口以 interface{} 形式返回 v 的当前值。它相当于：var i interface{} = (v 的底层值) 如果通过访问未导出的结构字段获取值，则会引起混乱。
			if v != nil {
				err = v.(error)
			}
			break
		}
	}

	return true, err
}

// GetFeature returns a feature of the given type, or nil if such feature is not registered.
// GetFeature 返回给定类型的特征，如果该特征未注册，则返回 nil。
func (s *Instance) GetFeature(featureType interface{}) features.Feature {
	fmt.Println("in gzv2ray.go func (s *Instance) GetFeature(featureType interface{})")
	return getFeature(s.features, reflect.TypeOf(featureType))
}

func getFeature(allFeatures []features.Feature, t reflect.Type) features.Feature {
	//fmt.Println("in gzv2ray.go func getFeature(allFeatures []features.Feature, t reflect.Type) t: ", reflect.ValueOf(t))
	for _, f := range allFeatures {
		//fmt.Println("in gzv2ray.go func getFeature(allFeatures []features.Feature, t reflect.Type)  range allFeatures: ", reflect.TypeOf(f.Type()))
		if reflect.TypeOf(f.Type()) == t {
			//fmt.Println("in gzv2ray.go func getFeature(allFeatures []features.Feature, t reflect.Type) reflect.TypeOf(f.Type()) == t")
			return f
		}
	}
	return nil
}

func addInboundHandlers(server *Instance, configs []*InboundHandlerConfig) error {
	fmt.Println("in gzv2ray.go func addInboundHandlers")
	for _, inboundConfig := range configs {
		//fmt.Println("in gzv2ray.go func addInboundHandlers range configs")
		//gztest.GetMessageReflectType(*inboundConfig)
		if err := AddInboundHandler(server, inboundConfig); err != nil {
			return err
		}
	}

	return nil
}

func AddInboundHandler(server *Instance, config *InboundHandlerConfig) error {
	fmt.Println("in gzv2ray.go func AddInboundHandler")
	inboundManager := server.GetFeature(inbound.ManagerType()).(inbound.Manager)
	//fmt.Println("in gzv2ray.go func addInboundHandler inboundManager := server.GetFeature ok")
	rawHandler, err := CreateObject(server, config)
	if err != nil {
		return err
	}
	handler, ok := rawHandler.(inbound.Handler)
	if !ok {
		return errors.New("not an InboundHandler")
	}
	if err := inboundManager.AddHandler(server.ctx, handler); err != nil {
		return err
	}
	return nil
}

// RequireFeatures is a helper function to require features from Instance in context.
// RequireFeatures 是一个辅助函数，用于在上下文中请求来自实例的特征
// See Instance.RequireFeatures for more information.
// 查看 Instance.RequireFeatures 以了解更多信息。
func RequireFeatures(ctx context.Context, callback interface{}) error {
	v := MustFromContext(ctx)
	return v.RequireFeatures(callback)
}

// RequireFeatures registers a callback, which will be called when all dependent features are registered.
// RequireFeatures 注册一个回调，当所有依赖功能都注册后将调用该回调。
// The callback must be a func(). All its parameters must be features.Feature.
// 回调必须是一个 func()。它的所有参数必须是 features.Feature。
func (s *Instance) RequireFeatures(callback interface{}) error {
	fmt.Println("in gzv2ray.go (s *Instance) RequireFeatures")
	callbackType := reflect.TypeOf(callback)
	// 确认回调接口是函数
	if callbackType.Kind() != reflect.Func {
		panic("not a function")
	}

	var featureTypes []reflect.Type
	// featureTypes 汇总回调函数各个参数 feature 的指针
	for i := 0; i < callbackType.NumIn(); i++ {
		featureTypes = append(featureTypes, reflect.PointerTo(callbackType.In(i)))
	}
	
	r := resolution{
		deps:     featureTypes,
		callback: callback,
	}
	
	if finished, err := r.resolve(s.features); finished {
		return err
	}
	fmt.Println("in gzv2ray.go (s *Instance) RequireFeatures r ")
	gztest.GetMessageReflectType(r.deps)
	// 把没有注册的依赖功能类型列表 r.deps 添加到实例 featureResolutions 中
	s.featureResolutions = append(s.featureResolutions, r)
	return nil
}

func addOutboundHandlers(server *Instance, configs []*OutboundHandlerConfig) error {
	for _, outboundConfig := range configs {
		if err := AddOutboundHandler(server, outboundConfig); err != nil {
			return err
		}
	}

	return nil
}

func AddOutboundHandler(server *Instance, config *OutboundHandlerConfig) error {
	outboundManager := server.GetFeature(outbound.ManagerType()).(outbound.Manager)
	rawHandler, err := CreateObject(server, config)
	if err != nil {
		return err
	}
	handler, ok := rawHandler.(outbound.Handler)
	if !ok {
		return errors.New("not an OutboundHandler")
	}
	if err := outboundManager.AddHandler(server.ctx, handler); err != nil {
		return err
	}
	return nil
}
