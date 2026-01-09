# HTTP网关 (Gateway HTTP)

基于go-zero的动态HTTP网关服务，支持自动路由到不同的业务服务，无需手动定义handler。

## 🚀 核心特性

- **🔄 动态路由**: 根据请求路径自动路由到对应服务
- **📋 多参数支持**: 支持GET查询参数、POST JSON、RESTful路径参数
- **🔍 服务发现**: 自动发现和调用注册的RPC服务
- **🛡️ 错误处理**: 统一的错误处理和响应格式
- **📊 健康检查**: 内置健康检查接口

## 📖 使用方式

### 1. RESTful风格调用 (推荐)

```bash
# 用户登录
curl -X POST http://localhost:8888/api/login/logon \
  -H "Content-Type: application/json" \
  -d '{"accounts": "user123", "password": "pass123"}'

# 获取用户信息
curl -X GET "http://localhost:8888/api/user/getUserInfo?user_id=123"

# 响应格式
{
  "code": 0,
  "message": "success",
  "data": {
    "userId": 123,
    "nickname": "allen",
    "gold": 1111
  }
}
```

### 2. 通用网关调用

```bash
# 通用POST接口
curl -X POST http://localhost:8888/api/generic \
  -H "Content-Type: application/json" \
  -d '{
    "service": "login",
    "method": "logon",
    "data": {
      "accounts": "user123",
      "password": "pass123"
    }
  }'
```

### 3. GET查询参数调用

```bash
# 通过查询参数指定服务和方法
curl "http://localhost:8888/api/generic?service=login&method=logon&accounts=user123&password=pass123"
```

## 🏗️ 架构设计

```
HTTP请求 → 网关路由器 → 服务发现 → RPC调用 → 响应格式化
     ↓           ↓           ↓         ↓           ↓
   /api/*    动态解析    etcd注册   gRPC调用   JSON响应
```

### 路由规则

| 请求路径 | HTTP方法 | 解析方式 | 示例 |
|----------|----------|----------|------|
| `/api/{service}/{method}` | GET | 路径参数 + 查询参数 | `/api/user/getUserInfo?user_id=123` |
| `/api/{service}/{method}` | POST | 路径参数 + JSON body | `/api/login/logon` + `{"accounts": "user"}` |
| `/api/generic` | POST | JSON body指定service/method | `/api/generic` + `{"service": "login", "method": "logon"}` |

### 大小写处理

**🔄 智能大小写转换**
- **服务名**: 自动转换为小写 (UserService → user)
- **方法名**: 保持原样 (GetUserInfo → GetUserInfo)

```bash
# 支持以下所有格式：
GET /api/user/getUserInfo
GET /api/User/getUserInfo
GET /api/UserService/GetUserInfo  # ✅ 自动转换为小写
GET /api/USER/GetUserInfo         # ✅ 自动转换为小写
```

### 类型安全修复

**🔧 修复类型不匹配问题**
- **之前**: 使用动态结构体，可能导致类型不匹配
- **现在**: 直接使用proto生成的类型，确保类型安全

```go
// ❌ 之前的实现（可能有类型不匹配）
request = &struct {
    Accounts string `json:"accounts"`
    Password string `json:"password"`
}{}

// ✅ 现在的实现（类型安全）
request = &loginpb.LogonRequest{}
```

### JSON解析优化

**🔧 修复httpx.ParseJsonBody错误**
- **问题**: `httpx.ParseJsonBody` 类型不匹配和错误处理
- **解决方案**: 完整的参数解析策略，支持多种Content-Type

```go
// ✅ 现在的实现（多格式支持）
case "POST", "PUT", "DELETE":
    // 初始化数据map
    data = make(map[string]interface{})

    // 优先从查询参数获取数据
    for key, values := range r.URL.Query() {
        if len(values) > 0 {
            data[key] = values[0]
        }
    }

    // 根据Content-Type处理请求体
    contentType := r.Header.Get("Content-Type")
    if strings.Contains(contentType, "application/json") {
        // JSON请求体
        var jsonData map[string]interface{}
        if err := httpx.ParseJsonBody(r, &jsonData); err != nil {
            httpx.ErrorCtx(r.Context(), w, fmt.Errorf("invalid JSON: %w", err))
            return
        }
        // 合并JSON数据（JSON优先级高于查询参数）
        for key, value := range jsonData {
            data[key] = value
        }
    } else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
        // 表单数据
        if err := r.ParseForm(); err != nil {
            httpx.ErrorCtx(r.Context(), w, fmt.Errorf("invalid form: %w", err))
            return
        }
        for key, values := range r.PostForm {
            if len(values) > 0 {
                data[key] = values[0]
            }
        }
    }
```

### 支持的请求格式

**📋 Content-Type支持**
- ✅ `application/json`: JSON格式请求体
- ✅ `application/x-www-form-urlencoded`: 表单格式
- ✅ 查询参数: URL query string
- ✅ 混合模式: 同时支持JSON和查询参数

## 🔧 配置说明

```yaml
# HTTP网关配置
Name: gateway_http-api
Host: 0.0.0.0
Port: 8888

# RPC服务配置
LoginRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: login.rpc

UserRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: user.rpc
```

## 📋 API接口列表

### 兼容性接口 (保持原有功能)

```bash
# 原有接口保持不变
POST /login  # 手动实现的登录接口
GET /from/:name  # 示例接口
GET /health  # 健康检查
```

### 动态路由接口 (新增)

```bash
# 通用网关
POST /api/generic

# RESTful风格
GET|POST|PUT|DELETE /api/{service}/{method}
```

## 🛠️ 开发指南

### 添加新服务支持

1. **在proto中定义服务** (可选)
2. **更新服务路由映射**

```go
// 在genericlogic.go中添加
l.serviceRoutes = map[string]interface{}{
    "login":    l.svcCtx.LoginRpc,
    "user":     l.svcCtx.UserRpc,
    "newgame":  l.svcCtx.NewGameRpc,  // 新增服务
}
```

3. **添加请求参数映射**

```go
// 在buildRPCRequest方法中添加
case "newgame.CreateRoomRequest":
    request = &struct {
        RoomName  string `json:"room_name"`
        MaxPlayers int32  `json:"max_players"`
    }{}
```

4. **配置RPC客户端**

```yaml
# 在配置文件中添加
NewGameRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: newgame.rpc
```

## 📊 性能特性

- **异步处理**: 所有RPC调用都是异步的
- **连接复用**: 基于etcd的服务发现和连接池
- **错误恢复**: 自动重试和熔断机制
- **监控友好**: 结构化日志便于监控

## 🔍 调试技巧

### 启用详细日志

```bash
# 查看所有路由调用
grep "GenericGateway request" logs/*.log

# 查看RPC调用错误
grep "Service call failed" logs/*.log
```

### 健康检查

```bash
curl http://localhost:8888/health
# 响应: {"status":"ok","service":"gateway_http"}
```

## 🚀 部署运行

```bash
# 编译
cd server/gateway_http
go build .

# 运行
./gateway_http -f etc/gatewayhttp-api.yaml
```

## 🔄 与手动Handler对比

| 特性 | 手动Handler | 动态网关 |
|------|-------------|----------|
| **开发效率** | 🐌 每个接口都要写handler | ⚡ 配置即可调用 |
| **维护成本** | 🔧 N个接口 = N个handler | 🛠️ 1个通用handler |
| **扩展性** | 📦 需要修改代码 | 🔄 配置即可扩展 |
| **错误处理** | 🎯 各不相同 | 🎯 统一处理 |
| **测试覆盖** | 📝 需要分别测试 | 📝 通用测试 |

## 📈 最佳实践

1. **优先使用RESTful风格**: `/api/{service}/{method}`
2. **复杂参数用POST**: 大量数据或复杂结构使用POST JSON
3. **保持向后兼容**: 原有手动handler继续保留
4. **统一错误处理**: 所有错误都返回标准格式
5. **监控关键指标**: 请求量、响应时间、错误率

---

**🎯 总结**: 这个动态HTTP网关大大简化了微服务架构下的API开发，让前端可以直接调用后端服务而无需网关层的胶水代码。开发效率提升显著，维护成本大幅降低。
