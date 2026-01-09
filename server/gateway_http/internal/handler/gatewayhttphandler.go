// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zerogame/server/gateway_http/internal/logic"
	"zerogame/server/gateway_http/internal/svc"
	"zerogame/server/gateway_http/internal/types"
)

// Gateway_httpHandler 原有的handler（保持兼容）
func Gateway_httpHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.Request
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewGateway_httpLogic(r.Context(), svcCtx)
		resp, err := l.Gateway_http(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// GenericGatewayHandler 通用网关handler - 支持动态路由
func GenericGatewayHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS处理
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var req types.GenericRequest
		var err error

		// 支持多种参数传递方式
		switch r.Method {
		case "GET":
			// 从URL路径解析：/api/{service}/{method}
			err = parseFromURL(r, &req)
		case "POST", "PUT", "DELETE":
			// 从JSON body解析
			err = httpx.ParseJsonBody(r, &req)
		default:
			httpx.Error(w, fmt.Errorf("Unsupported HTTP method"))
			return
		}

		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewGenericLogic(r.Context(), svcCtx)
		resp, err := l.GenericGateway(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// RESTfulGatewayHandler RESTful风格的网关handler
// 支持路径如：/api/login/logon, /api/user/getUserInfo
func RESTfulGatewayHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS处理
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 从URL路径解析服务和方法
		service, method, err := parseServiceMethodFromPath(r.URL.Path)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 解析请求数据
		var data map[string]interface{}

		switch r.Method {
		case "GET":
			// 从查询参数解析
			data = make(map[string]interface{})
			for key, values := range r.URL.Query() {
				if len(values) > 0 {
					data[key] = values[0]
				}
			}
		case "POST", "PUT", "DELETE":
			// 从请求体解析参数，支持JSON和表单数据
			contentType := r.Header.Get("Content-Type")

			// 初始化数据map
			data = make(map[string]interface{})

			// 优先从查询参数获取数据
			for key, values := range r.URL.Query() {
				if len(values) > 0 {
					data[key] = values[0]
				}
			}

			// 根据Content-Type处理请求体
			if strings.Contains(contentType, "application/json") {
				// JSON请求体 - 直接读取body并解析
				body, err := io.ReadAll(r.Body)
				if err != nil {
					httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to read request body: %w", err))
					return
				}

				if len(body) > 0 {
					var jsonData map[string]interface{}
					if err := json.Unmarshal(body, &jsonData); err != nil {
						httpx.ErrorCtx(r.Context(), w, fmt.Errorf("invalid JSON request body: %w", err))
						return
					}
					// 合并JSON数据（JSON优先级高于查询参数）
					for key, value := range jsonData {
						data[key] = value
					}
				}
			} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
				// 表单数据
				if err := r.ParseForm(); err != nil {
					httpx.ErrorCtx(r.Context(), w, fmt.Errorf("invalid form data: %w", err))
					return
				}
				// 合并表单数据
				for key, values := range r.PostForm {
					if len(values) > 0 {
						data[key] = values[0]
					}
				}
			}
			// 如果没有获取到任何数据，给出提示
			if len(data) == 0 {
				httpx.ErrorCtx(r.Context(), w, fmt.Errorf("no request data provided (neither query parameters nor request body)"))
				return
			}
		}

		l := logic.NewGenericLogic(r.Context(), svcCtx)
		resp, err := l.HandleRESTful(service, method, data)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// parseFromURL 从URL路径解析请求参数
func parseFromURL(r *http.Request, req *types.GenericRequest) error {
	// 从查询参数解析service和method
	req.Service = r.URL.Query().Get("service")
	req.Method = r.URL.Query().Get("method")

	if req.Service == "" || req.Method == "" {
		return errors.New("service and method are required")
	}

	// 从其他查询参数构建data
	req.Data = make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if key != "service" && key != "method" && len(values) > 0 {
			req.Data[key] = values[0]
		}
	}

	return nil
}

// parseServiceMethodFromPath 从URL路径解析服务和方法
// 例如：/api/login/logon -> service="login", method="logon"
// 支持大小写自动转换：/api/UserService/GetUserInfo -> service="user", method="getUserInfo"
func parseServiceMethodFromPath(path string) (service, method string, err error) {
	// 移除开头的斜杠
	path = strings.TrimPrefix(path, "/")

	// 分割路径
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] != "api" {
		return "", "", fmt.Errorf("Invalid path format. Expected: /api/{service}/{method}")
	}

	serviceName := strings.ToLower(parts[1])

	// 智能服务名转换 - 根据已知的服务进行精确匹配
	switch serviceName {
	case "login", "loginservice", "login_service":
		service = "login"
	case "user", "userservice", "user_service", "users":
		service = "user"
	// 可以在这里添加更多服务名的别名
	// case "hall", "hallservice":
	//     service = "hall"
	default:
		// 如果找不到匹配的服务名，尝试直接使用小写版本
		service = serviceName
	}

	method = parts[2] // 方法名保持原样，因为RPC方法名通常是驼峰命名

	if service == "" || method == "" {
		return "", "", fmt.Errorf("Service and method cannot be empty")
	}

	return service, method, nil
}
