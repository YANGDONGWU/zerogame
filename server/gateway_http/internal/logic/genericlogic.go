package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	loginpb "zerogame/pb/login"
	userpb "zerogame/pb/user"
	"zerogame/server/gateway_http/internal/svc"
	"zerogame/server/gateway_http/internal/types"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mapping"
)

// GenericLogic 通用HTTP网关逻辑
type GenericLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	// 服务路由映射
	serviceRoutes map[string]interface{}
}

// NewGenericLogic 创建通用逻辑处理器
func NewGenericLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenericLogic {
	l := &GenericLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}

	// 初始化服务路由
	l.initServiceRoutes()

	return l
}

// initServiceRoutes 初始化服务路由映射
func (l *GenericLogic) initServiceRoutes() {
	l.serviceRoutes = map[string]interface{}{
		"login": l.svcCtx.LoginRpc,
		"user":  l.svcCtx.UserRpc,
		// 可以在这里添加更多的服务路由
		// "hall":  l.svcCtx.HallRpc,
		// "game":  l.svcCtx.GameRpc,
	}
}

// GenericGateway 通用网关处理方法
func (l *GenericLogic) GenericGateway(req *types.GenericRequest) (*types.GenericResponse, error) {
	l.Infof("GenericGateway request: service=%s, method=%s", req.Service, req.Method)

	// 参数校验
	if err := l.validateRequest(req); err != nil {
		return &types.GenericResponse{
			Code:    10000002, // SYSTEM_INVALID_PARAMS
			Message: fmt.Sprintf("Invalid request: %v", err),
		}, nil
	}

	// 路由到对应的RPC服务
	resp, err := l.routeToService(req)
	if err != nil {
		l.Errorf("Failed to route to service: %v", err)
		return &types.GenericResponse{
			Code:    10000003, // SYSTEM_RPC_CALL_ERROR
			Message: fmt.Sprintf("Service call failed: %v", err),
		}, nil
	}

	return resp, nil
}

// validateRequest 校验请求参数
func (l *GenericLogic) validateRequest(req *types.GenericRequest) error {
	if req.Service == "" {
		return errors.New("service is required")
	}
	if req.Method == "" {
		return errors.New("method is required")
	}

	// 检查服务是否存在
	if _, exists := l.serviceRoutes[req.Service]; !exists {
		return fmt.Errorf("service '%s' not found", req.Service)
	}

	return nil
}

// routeToService 路由到具体的RPC服务
func (l *GenericLogic) routeToService(req *types.GenericRequest) (*types.GenericResponse, error) {
	serviceClient, exists := l.serviceRoutes[req.Service]
	if !exists {
		return nil, fmt.Errorf("service '%s' not found", req.Service)
	}

	// 使用反射调用对应的RPC方法
	result, err := l.callRPCMethod(serviceClient, req)
	if err != nil {
		return nil, err
	}

	// 转换响应格式
	return l.convertRPCResponse(result)
}

// callRPCMethod 使用反射调用RPC方法
func (l *GenericLogic) callRPCMethod(serviceClient interface{}, req *types.GenericRequest) (interface{}, error) {
	clientValue := reflect.ValueOf(serviceClient)
	methodName := strings.Title(req.Method) // 首字母大写

	method := clientValue.MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method '%s' not found in service '%s'", methodName, req.Service)
	}

	// 构建请求参数
	requestParam, err := l.buildRPCRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build RPC request: %v", err)
	}

	// 调用RPC方法
	callResults := method.Call([]reflect.Value{
		reflect.ValueOf(l.ctx),
		reflect.ValueOf(requestParam),
	})

	// 处理调用结果
	if len(callResults) != 2 {
		return nil, fmt.Errorf("unexpected RPC call results count: %d", len(callResults))
	}

	// 检查错误
	if !callResults[1].IsNil() {
		err, ok := callResults[1].Interface().(error)
		if ok && err != nil {
			return nil, err
		}
	}

	// 返回响应
	return callResults[0].Interface(), nil
}

// buildRPCRequest 构建RPC请求参数
func (l *GenericLogic) buildRPCRequest(req *types.GenericRequest) (interface{}, error) {
	// 根据服务和方法名构造请求类型
	requestTypeName := fmt.Sprintf("%s.%sRequest", req.Service, strings.Title(req.Method))

	// 使用proto生成的类型
	var request interface{}
	switch requestTypeName {
	case "login.LogonRequest":
		request = &loginpb.LogonRequest{}
	case "user.GetUserInfoRequest":
		request = &userpb.GetUserInfoRequest{}
	// 可以在这里添加更多服务的请求类型
	// case "hall.CreateRoomRequest":
	//     request = &hallpb.CreateRoomRequest{}
	default:
		// 对于不支持的类型，返回错误
		return nil, fmt.Errorf("unsupported request type: %s", requestTypeName)
	}

	// 将请求数据映射到proto结构体
	if err := mapping.UnmarshalKey(req.Data, request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request data: %v", err)
	}

	return request, nil
}


// convertRPCResponse 转换RPC响应为通用格式
func (l *GenericLogic) convertRPCResponse(rpcResp interface{}) (*types.GenericResponse, error) {
	// 将RPC响应转换为map格式
	respData, err := l.rpcResponseToMap(rpcResp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert RPC response: %v", err)
	}

	return &types.GenericResponse{
		Code:    0, // 成功
		Message: "success",
		Data:    respData,
	}, nil
}

// rpcResponseToMap 将RPC响应转换为map格式
func (l *GenericLogic) rpcResponseToMap(resp interface{}) (map[string]interface{}, error) {
	// 使用JSON序列化转换为map格式
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// RESTful风格的路由处理
func (l *GenericLogic) HandleRESTful(service, method string, data map[string]interface{}) (*types.GenericResponse, error) {
	req := &types.GenericRequest{
		Service: service,
		Method:  method,
		Data:    data,
	}

	return l.GenericGateway(req)
}
