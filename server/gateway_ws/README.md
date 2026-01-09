# WebSocket网关 (Gateway WS)

基于go-zero的棋牌游戏WebSocket网关服务，提供实时通信功能。

## 功能特性

- **连接管理**: 支持大量并发WebSocket连接，自动清理死连接
- **消息协议**: 基于Protocol Buffers的统一消息协议
- **消息路由**: 根据消息类型自动路由到对应的业务处理器
- **广播推送**: 支持房间广播、全员广播、指定用户推送
- **心跳检测**: 自动检测连接活跃状态，超时清理
- **负载均衡**: 支持水平扩展部署

## 架构设计

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │◄──►│  WS Gateway     │◄──►│  Game Services  │
│                 │    │                 │    │                 │
│ - Browser       │    │ - Connection Mgmt│    │ - Login Service│
│ - Mobile App    │    │ - Message Parser │    │ - User Service │
│ - Game Client   │    │ - Message Router │    │ - Game Service │
└─────────────────┘    │ - Broadcaster    │    │ - Hall Service │
                       └─────────────────┘    └─────────────────┘
```

## 服务启动说明

### Q: go-zero自己会启动服务，为什么还要手动启动WebSocket服务器？

**A:** go-zero框架确实会自动启动配置的HTTP服务，但WebSocket网关是一个独立的网络服务：

1. **go-zero的REST服务**: 通过`rest.MustNewServer()`启动，用于处理HTTP API请求
2. **WebSocket服务**: 通过`WsServer.Start()`启动，用于处理WebSocket长连接

两者是并存的关系：
- REST服务处理短连接的API调用
- WebSocket服务处理长连接的实时通信

这种设计允许同一个应用同时提供HTTP API和WebSocket服务。

## 消息协议

### 消息结构

```protobuf
message WebSocketMessage {
  MessageHeader header = 1;
  bytes         body   = 2;
}

message MessageHeader {
  MessageType msg_type     = 1;
  string      msg_id       = 2;
  int64       timestamp    = 3;
  int32       user_id      = 4;
  string      room_id      = 5;
  string      game_id      = 6;
}
```

### Q: ParseMessageBody方法中switch case太多，是否有更智能的方式？

**A:** 已实现**策略模式 + 注册机制**优化：

#### 优化前（硬编码switch）:
```go
switch msg.Header.MsgType {
case MSG_HEARTBEAT:
    return parseHeartbeat(msg.Body)
case MSG_LOGIN:
    return parseLoginMessage(msg.Body)
// ... 9个case分支
}
```

#### 优化后（策略模式）:
```go
// 1. 定义解析器接口
type MessageBodyParser interface {
    Parse(data []byte) (interface{}, error)
    GetMessageType() MessageType
}

// 2. 注册所有解析器
parsers := map[MessageType]MessageBodyParser{
    MSG_HEARTBEAT: &HeartbeatParser{},
    MSG_LOGIN:     &LoginMessageParser{},
    // ...
}

// 3. 动态解析
parser := parsers[msg.Header.MsgType]
return parser.Parse(msg.Body)
```

#### 优势：
- **扩展性**: 新增消息类型只需实现接口，无需修改switch
- **维护性**: 每个消息类型的解析逻辑独立封装
- **性能**: map查找比switch略慢，但代码简洁性提升显著

### Q: 既然用proto定义，为什么解析消息时用json.Unmarshal而不是proto序列化？

**A:** 这是一个**开发便利性 vs 性能优化**的权衡选择：

#### 当前实现：JSON序列化
```javascript
// 前端发送
ws.send(JSON.stringify({
    header: { msg_type: 1, msg_id: "001" },
    body: JSON.stringify({ token: "xxx" })
}))
```

#### 可选实现：Proto序列化
```javascript
// 前端需要
const message = LoginMessage.create({ token: "xxx" })
const protoData = LoginMessage.encode(message).finish()
ws.send(protoData) // 二进制数据
```

#### 对比分析：

| 方面 | JSON | Proto |
|------|------|-------|
| **可读性** | ✅ 人类可读，易调试 | ❌ 二进制，难调试 |
| **兼容性** | ✅ 前端原生支持 | ❌ 需要额外库 |
| **开发效率** | ✅ 快速开发 | ❌ 需要编译proto |
| **传输效率** | ⚠️ 体积较大 | ✅ 压缩率高(30-60%) |
| **解析性能** | ⚠️ JSON解析较慢 | ✅ 二进制解析快 |
| **类型安全** | ⚠️ 运行时检查 | ✅ 编译时保证 |

#### 推荐方案：

**开发环境**: JSON（便于调试，快速迭代）
```go
parser := NewMessageParser() // 使用JSON
```

**生产环境**: Proto（高性能，小体积）
```go
parser := NewProtoMessageParser() // 使用Proto
```

**混合方案**: 消息头用JSON，消息体用Proto
```go
// 解析消息头（JSON）
header := parseJSON(msg.Header)
// 解析消息体（Proto）
body := parseProto(msg.Body, header.MsgType)
```

这样既保持了开发便利性，又获得了性能优势。

## 最佳实践

### 1. 服务启动架构
```
go-zero REST API (端口8080) ──┐
                                 ├── 同一个进程
WebSocket Gateway (端口8888) ──┘
```

### 2. 消息解析策略
```go
// 开发环境
SerializationFormat: "json"

// 生产环境
SerializationFormat: "proto"
```

### 3. 扩展新消息类型
```go
// 1. 在proto文件中定义消息
message CustomMessage {
    string custom_field = 1;
}

// 2. 实现解析器
type CustomMessageParser struct{}
func (p *CustomMessageParser) Parse(data []byte) (interface{}, error) {
    var msg pb.CustomMessage
    return json.Unmarshal(data, &msg) // 或 proto.Unmarshal
}

// 3. 注册到MessageParser
func (p *MessageParser) registerDefaultParsers() {
    // 添加新解析器
    p.parsers[pb.MessageType_MSG_CUSTOM] = &CustomMessageParser{}
}
```

### 4. 性能监控
- 连接数监控：`connMgr.GetConnectionCount()`
- 消息处理统计：集成Prometheus指标
- 响应时间监控：记录消息处理耗时

### 消息类型

#### 客户端消息 (0-99)
- `MSG_HEARTBEAT` (0): 心跳
- `MSG_LOGIN` (1): 登录
- `MSG_LOGOUT` (2): 登出
- `MSG_JOIN_ROOM` (3): 加入房间
- `MSG_LEAVE_ROOM` (4): 离开房间
- `MSG_GAME_ACTION` (5): 游戏操作
- `MSG_CHAT` (6): 聊天消息
- `MSG_USER_INFO_QUERY` (7): 用户信息查询
- `MSG_ROOM_LIST_QUERY` (8): 房间列表查询

#### 服务端推送消息 (100-199)
- `MSG_PUSH_GAME_STATE` (100): 游戏状态推送
- `MSG_PUSH_ROOM_INFO` (101): 房间信息推送
- `MSG_PUSH_USER_UPDATE` (102): 用户状态更新
- `MSG_PUSH_SYSTEM_MSG` (103): 系统消息
- `MSG_PUSH_CHAT_MSG` (104): 聊天消息推送
- `MSG_PUSH_BROADCAST` (105): 广播消息

## 配置说明

```yaml
Name: gateway_ws-api
Host: 0.0.0.0
Port: 8080
Timeout: 30000

Log:
  Mode: console
  Level: info
  Encoding: json

WebSocket:
  Host: 0.0.0.0
  Port: 8888
  Path: "/ws"
  ReadTimeout: 60
  WriteTimeout: 60
  MaxMessageSize: 65536
  HeartbeatInterval: 30
  HeartbeatTimeout: 90
  MaxConnections: 10000
  EnableCompression: true
  AllowedOrigins:
    - "*"
```

## 使用示例

### 连接WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8888/ws');

// 监听连接打开
ws.onopen = function(event) {
    console.log('WebSocket connected');
};

// 监听消息
ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};

// 发送心跳
setInterval(() => {
    ws.send(JSON.stringify({
        header: {
            msg_type: 0,
            msg_id: Date.now().toString(),
            timestamp: Date.now()
        },
        body: JSON.stringify({
            client_time: Date.now()
        })
    }));
}, 30000);
```

### 登录消息

```javascript
const loginMessage = {
    header: {
        msg_type: 1,
        msg_id: "login_001",
        timestamp: Date.now()
    },
    body: JSON.stringify({
        token: "your_login_token"
    })
};

ws.send(JSON.stringify(loginMessage));
```

### 加入房间

```javascript
const joinRoomMessage = {
    header: {
        msg_type: 3,
        msg_id: "join_001",
        timestamp: Date.now()
    },
    body: JSON.stringify({
        room_id: "room_123"
    })
};

ws.send(JSON.stringify(joinRoomMessage));
```

## 部署运行

### 编译

```bash
cd server/gateway_ws
go build .
```

### 运行

```bash
./gateway_ws -f etc/gatewayws-api.yaml
```

### Docker部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o gateway_ws ./server/gateway_ws

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/gateway_ws .
COPY --from=builder /app/server/gateway_ws/etc ./etc
CMD ["./gateway_ws", "-f", "etc/gatewayws-api.yaml"]
```

## 监控指标

- 连接数量统计
- 房间数量统计
- 消息处理统计
- 错误率统计
- 性能指标监控

## 扩展开发

### 添加新的消息类型

1. 在 `proto/websocket.proto` 中定义新的消息类型
2. 重新生成Go代码: `protoc --go_out=. proto/websocket.proto`
3. 在 `MessageRouter` 中注册新的处理器
4. 实现具体的业务逻辑

### 自定义消息处理器

```go
type CustomHandler struct {
    // 依赖注入
}

func (h *CustomHandler) Handle(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, body interface{}) error {
    // 处理逻辑
    return nil
}
```

## 性能优化

- 使用连接池管理大量并发连接
- 消息异步处理，避免阻塞
- 心跳检测自动清理死连接
- 支持消息压缩减少带宽
- 水平扩展支持负载均衡

## 安全考虑

- Origin检查防止跨域攻击
- 消息大小限制防止DOS攻击
- 连接数限制防止资源耗尽
- Token认证确保用户身份
- 超时机制防止连接泄露
