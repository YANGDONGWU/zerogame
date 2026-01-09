package manager

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"zerogame/pb"

	"zerogame/server/gateway_ws/internal/config"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

// WebSocketServer WebSocket服务器
type WebSocketServer struct {
	config      *config.WebSocketConfig
	connMgr     *ConnectionManager
	broadcaster *Broadcaster
	router      *MessageRouter
	parser      MessageParserInterface
	upgrader    *websocket.Upgrader
	server      *http.Server
	logx.Logger
}

// NewWebSocketServer 创建WebSocket服务器（默认JSON解析器）
func NewWebSocketServer(cfg *config.WebSocketConfig) *WebSocketServer {
	return NewWebSocketServerWithParser(cfg, NewMessageParser())
}

// NewWebSocketServerWithParser 创建WebSocket服务器（指定解析器）
func NewWebSocketServerWithParser(cfg *config.WebSocketConfig, parser MessageParserInterface) *WebSocketServer {
	// 创建升级器
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if len(cfg.AllowedOrigins) == 0 || cfg.AllowedOrigins[0] == "*" {
				return true
			}
			for _, allowed := range cfg.AllowedOrigins {
				if allowed == origin {
					return true
				}
			}
			return false
		},
	}

	// 启用压缩
	if cfg.EnableCompression {
		upgrader.EnableCompression = true
	}

	// 创建管理器
	connMgr := NewConnectionManager(cfg.MaxConnections)
	broadcaster := NewBroadcaster(connMgr, parser, 10) // 10个工作协程
	router := NewMessageRouter()

	// 注册默认处理器
	router.RegisterDefaultHandlers(connMgr, broadcaster, parser)

	return &WebSocketServer{
		Logger:      logx.WithContext(context.Background()),
		config:      cfg,
		connMgr:     connMgr,
		broadcaster: broadcaster,
		router:      router,
		parser:      parser,
		upgrader:    upgrader,
	}
}

// Start 启动WebSocket服务器
func (s *WebSocketServer) Start(ctx context.Context) error {
	// 启动广播器
	s.broadcaster.Start(ctx)

	// 启动心跳检查器
	go s.connMgr.StartHeartbeatChecker(ctx,
		time.Duration(s.config.HeartbeatInterval)*time.Second,
		time.Duration(s.config.HeartbeatTimeout)*time.Second)

	// 创建HTTP服务器
	mux := http.NewServeMux()
	mux.HandleFunc(s.config.Path, s.handleWebSocket)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler: mux,
	}

	s.Infof("Starting WebSocket server on %s:%d%s", s.config.Host, s.config.Port, s.config.Path)

	// 启动服务器
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Errorf("WebSocket server error: %v", err)
		}
	}()

	return nil
}

// Stop 停止WebSocket服务器
func (s *WebSocketServer) Stop() error {
	s.Infof("Stopping WebSocket server...")

	// 停止广播器
	s.broadcaster.Stop()

	// 关闭所有连接
	connections := s.connMgr.GetAllConnections()
	for _, conn := range connections {
		conn.Close()
	}

	// 停止HTTP服务器
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}

	return nil
}

// handleWebSocket 处理WebSocket连接
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Errorf("Failed to upgrade connection: %v", err)
		return
	}

	s.Infof("New WebSocket connection established from %s", r.RemoteAddr)

	// 设置连接参数
	conn.SetReadLimit(s.config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(time.Duration(s.config.ReadTimeout) * time.Second))
	conn.SetWriteDeadline(time.Now().Add(time.Duration(s.config.WriteTimeout) * time.Second))

	// 设置pong处理器来处理心跳
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(time.Duration(s.config.ReadTimeout) * time.Second))
		return nil
	})

	// 设置close处理器
	conn.SetCloseHandler(func(code int, text string) error {
		s.Infof("WebSocket connection closed: code=%d, text=%s", code, text)
		s.connMgr.RemoveConnection(conn)
		return nil
	})

	// 启动消息处理协程
	go s.handleConnection(conn)
}

// handleConnection 处理单个连接的消息
func (s *WebSocketServer) handleConnection(conn *websocket.Conn) {
	defer func() {
		conn.Close()
		s.connMgr.RemoveConnection(conn)
	}()

	for {
		// 读取消息
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.Errorf("WebSocket error: %v", err)
			}
			break
		}

		// 只处理文本消息
		if messageType != websocket.TextMessage {
			s.Infof("Received non-text message, type: %d", messageType)
			continue
		}

		// 处理消息
		if err := s.handleMessage(conn, data); err != nil {
			s.Errorf("Failed to handle message: %v", err)
			// 发送错误响应
			s.sendError(conn, "Failed to process message", err)
		}
	}
}

// handleMessage 处理消息
func (s *WebSocketServer) handleMessage(conn *websocket.Conn, data []byte) error {
	// 解析消息
	msg, err := s.parser.ParseMessage(data)
	if err != nil {
		return err
	}

	s.Infof("Received message: type=%d, user=%d, room=%s", msg.Header.MsgType, msg.Header.UserId, msg.Header.RoomId)

	// 创建上下文
	ctx := context.Background()

	// 路由消息
	return s.router.Route(ctx, conn, msg, s.parser)
}

// sendError 发送错误消息
func (s *WebSocketServer) sendError(conn *websocket.Conn, message string, err error) {
	errorMsg := map[string]interface{}{
		"error":   message,
		"details": err.Error(),
	}

	data, jsonErr := s.parser.SerializeMessageBody(pb.MessageType(0), errorMsg)
	if jsonErr != nil {
		s.Errorf("Failed to serialize error message: %v", jsonErr)
		return
	}

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if writeErr := conn.WriteMessage(websocket.TextMessage, data); writeErr != nil {
		s.Errorf("Failed to send error message: %v", writeErr)
	}
}

// GetStats 获取服务器统计信息
func (s *WebSocketServer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"connections":        s.connMgr.GetConnectionCount(),
		"rooms":              s.connMgr.GetRoomCount(),
		"max_connections":    s.config.MaxConnections,
		"heartbeat_interval": s.config.HeartbeatInterval,
		"heartbeat_timeout":  s.config.HeartbeatTimeout,
	}
}

// BroadcastToRoom 广播到房间
func (s *WebSocketServer) BroadcastToRoom(roomID string, message interface{}, excludeUser int32) error {
	pushMsg, err := s.parser.CreatePushMessage(pb.MessageType_MSG_PUSH_SYSTEM_MSG, 0, roomID, "", message)
	if err != nil {
		return err
	}

	s.broadcaster.BroadcastToRoom(roomID, pushMsg, excludeUser)
	return nil
}

// BroadcastToAll 广播到所有用户
func (s *WebSocketServer) BroadcastToAll(message interface{}) error {
	pushMsg, err := s.parser.CreatePushMessage(pb.MessageType_MSG_PUSH_BROADCAST, 0, "", "", message)
	if err != nil {
		return err
	}

	s.broadcaster.BroadcastToAll(pushMsg)
	return nil
}
