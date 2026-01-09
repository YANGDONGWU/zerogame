package manager

import (
	"context"
	"fmt"

	"zerogame/pb"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

// MessageHandler 消息处理器接口
type MessageHandler interface {
	Handle(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, body interface{}) error
}

// MessageRouter 消息路由器
type MessageRouter struct {
	handlers map[pb.MessageType]MessageHandler
	logx.Logger
}

// NewMessageRouter 创建消息路由器
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		Logger:   logx.WithContext(context.Background()),
		handlers: make(map[pb.MessageType]MessageHandler),
	}
}

// RegisterHandler 注册消息处理器
func (r *MessageRouter) RegisterHandler(msgType pb.MessageType, handler MessageHandler) {
	r.handlers[msgType] = handler
	r.Infof("Registered handler for message type: %d", msgType)
}

// Route 路由消息
func (r *MessageRouter) Route(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, parser MessageParserInterface) error {
	if msg.Header == nil {
		return fmt.Errorf("message header is nil")
	}

	handler, exists := r.handlers[msg.Header.MsgType]
	if !exists {
		r.Errorf("No handler found for message type: %d", msg.Header.MsgType)
		return fmt.Errorf("unsupported message type: %d", msg.Header.MsgType)
	}

	// 解析消息体
	body, err := parser.ParseMessageBody(msg)
	if err != nil {
		r.Errorf("Failed to parse message body for type %d: %v", msg.Header.MsgType, err)
		return fmt.Errorf("failed to parse message body: %w", err)
	}

	// 调用处理器
	return handler.Handle(ctx, conn, msg, body)
}

// DefaultMessageHandler 默认消息处理器
type DefaultMessageHandler struct {
	connMgr     *ConnectionManager
	broadcaster *Broadcaster
	parser      MessageParserInterface
	logx.Logger
}

// NewDefaultMessageHandler 创建默认消息处理器
func NewDefaultMessageHandler(connMgr *ConnectionManager, broadcaster *Broadcaster, parser MessageParserInterface) *DefaultMessageHandler {
	return &DefaultMessageHandler{
		connMgr:     connMgr,
		broadcaster: broadcaster,
		parser:      parser,
	}
}

// RegisterDefaultHandlers 注册默认处理器
func (r *MessageRouter) RegisterDefaultHandlers(connMgr *ConnectionManager, broadcaster *Broadcaster, parser MessageParserInterface) {
	handler := NewDefaultMessageHandler(connMgr, broadcaster, parser)

	// 注册各种消息类型的处理器
	r.RegisterHandler(pb.MessageType_MSG_HEARTBEAT, handler)
	r.RegisterHandler(pb.MessageType_MSG_LOGIN, handler)
	r.RegisterHandler(pb.MessageType_MSG_LOGOUT, handler)
	r.RegisterHandler(pb.MessageType_MSG_JOIN_ROOM, handler)
	r.RegisterHandler(pb.MessageType_MSG_LEAVE_ROOM, handler)
	r.RegisterHandler(pb.MessageType_MSG_GAME_ACTION, handler)
	r.RegisterHandler(pb.MessageType_MSG_CHAT, handler)
	r.RegisterHandler(pb.MessageType_MSG_USER_INFO_QUERY, handler)
	r.RegisterHandler(pb.MessageType_MSG_ROOM_LIST_QUERY, handler)
}

// Handle 处理消息
func (h *DefaultMessageHandler) Handle(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, body interface{}) error {
	switch msg.Header.MsgType {
	case pb.MessageType_MSG_HEARTBEAT:
		return h.handleHeartbeat(ctx, conn, msg, body.(*pb.Heartbeat))
	case pb.MessageType_MSG_LOGIN:
		return h.handleLogin(ctx, conn, msg, body.(*pb.LoginMessage))
	case pb.MessageType_MSG_LOGOUT:
		return h.handleLogout(ctx, conn, msg, body.(*pb.LogoutMessage))
	case pb.MessageType_MSG_JOIN_ROOM:
		return h.handleJoinRoom(ctx, conn, msg, body.(*pb.JoinRoomMessage))
	case pb.MessageType_MSG_LEAVE_ROOM:
		return h.handleLeaveRoom(ctx, conn, msg, body.(*pb.LeaveRoomMessage))
	case pb.MessageType_MSG_GAME_ACTION:
		return h.handleGameAction(ctx, conn, msg, body.(*pb.GameActionMessage))
	case pb.MessageType_MSG_CHAT:
		return h.handleChat(ctx, conn, msg, body.(*pb.ChatMessage))
	case pb.MessageType_MSG_USER_INFO_QUERY:
		return h.handleUserInfoQuery(ctx, conn, msg, body.(*pb.UserInfoQuery))
	case pb.MessageType_MSG_ROOM_LIST_QUERY:
		return h.handleRoomListQuery(ctx, conn, msg, body.(*pb.RoomListQuery))
	default:
		return fmt.Errorf("unsupported message type: %d", msg.Header.MsgType)
	}
}

// handleHeartbeat 处理心跳消息
func (h *DefaultMessageHandler) handleHeartbeat(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, heartbeat *pb.Heartbeat) error {
	// 更新心跳时间
	if clientConn := h.connMgr.GetClientConnection(conn); clientConn != nil {
		clientConn.UpdateHeartbeat()
	}

	// 发送心跳响应
	return h.broadcaster.SendHeartbeatResponse(conn, heartbeat.ClientTime)
}

// handleLogin 处理登录消息
func (h *DefaultMessageHandler) handleLogin(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, loginMsg *pb.LoginMessage) error {
	// TODO: 调用登录服务验证token
	// 这里暂时模拟登录成功，设置用户ID为1
	userID := int32(1)

	// 添加连接到管理器
	clientConn := h.connMgr.AddConnection(conn, userID)
	if clientConn == nil {
		return h.broadcaster.SendErrorResponse(conn, msg, 1001, "Connection limit reached")
	}

	// 发送登录成功响应
	resp, err := h.parser.CreateResponse(msg, 0, "Login successful", map[string]interface{}{
		"user_id": userID,
	})
	if err != nil {
		return err
	}

	data, err := h.parser.SerializeMessage(resp)
	if err != nil {
		return err
	}

	return h.broadcaster.SendMessage(conn, data)
}

// handleLogout 处理登出消息
func (h *DefaultMessageHandler) handleLogout(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, logoutMsg *pb.LogoutMessage) error {
	// 移除连接
	h.connMgr.RemoveConnection(conn)

	// 发送登出成功响应
	resp, err := h.parser.CreateResponse(msg, 0, "Logout successful", nil)
	if err != nil {
		return err
	}

	data, err := h.parser.SerializeMessage(resp)
	if err != nil {
		return err
	}

	return h.broadcaster.SendMessage(conn, data)
}

// handleJoinRoom 处理加入房间消息
func (h *DefaultMessageHandler) handleJoinRoom(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, joinMsg *pb.JoinRoomMessage) error {
	// 加入房间
	h.connMgr.JoinRoom(conn, joinMsg.RoomId)

	// 发送加入房间成功响应
	resp, err := h.parser.CreateResponse(msg, 0, "Joined room successfully", map[string]interface{}{
		"room_id": joinMsg.RoomId,
	})
	if err != nil {
		return err
	}

	data, err := h.parser.SerializeMessage(resp)
	if err != nil {
		return err
	}

	err = h.broadcaster.SendMessage(conn, data)
	if err != nil {
		return err
	}

	// 广播用户加入房间消息给房间内其他用户
	pushMsg, err := h.parser.CreatePushMessage(pb.MessageType_MSG_PUSH_USER_UPDATE, msg.Header.UserId, joinMsg.RoomId, "", &pb.UserUpdatePush{
		UserId:   msg.Header.UserId,
		Status:   1, // 1表示加入房间
		Location: joinMsg.RoomId,
	})
	if err != nil {
		return err
	}

	h.broadcaster.BroadcastToRoom(joinMsg.RoomId, pushMsg, msg.Header.UserId)
	return nil
}

// handleLeaveRoom 处理离开房间消息
func (h *DefaultMessageHandler) handleLeaveRoom(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, leaveMsg *pb.LeaveRoomMessage) error {
	// 离开房间
	h.connMgr.LeaveRoom(conn)

	// 发送离开房间成功响应
	resp, err := h.parser.CreateResponse(msg, 0, "Left room successfully", map[string]interface{}{
		"room_id": leaveMsg.RoomId,
	})
	if err != nil {
		return err
	}

	data, err := h.parser.SerializeMessage(resp)
	if err != nil {
		return err
	}

	err = h.broadcaster.SendMessage(conn, data)
	if err != nil {
		return err
	}

	// 广播用户离开房间消息
	pushMsg, err := h.parser.CreatePushMessage(pb.MessageType_MSG_PUSH_USER_UPDATE, msg.Header.UserId, leaveMsg.RoomId, "", &pb.UserUpdatePush{
		UserId:   msg.Header.UserId,
		Status:   0, // 0表示离开房间
		Location: "",
	})
	if err != nil {
		return err
	}

	h.broadcaster.BroadcastToRoom(leaveMsg.RoomId, pushMsg, 0) // 不排除任何人
	return nil
}

// handleGameAction 处理游戏操作消息
func (h *DefaultMessageHandler) handleGameAction(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, actionMsg *pb.GameActionMessage) error {
	// TODO: 根据游戏类型路由到对应的游戏服务
	h.Infof("Game action received: type=%s, room=%s", actionMsg.ActionType, msg.Header.RoomId)

	// 暂时返回成功响应
	resp, err := h.parser.CreateResponse(msg, 0, "Game action processed", nil)
	if err != nil {
		return err
	}

	data, err := h.parser.SerializeMessage(resp)
	if err != nil {
		return err
	}

	return h.broadcaster.SendMessage(conn, data)
}

// handleChat 处理聊天消息
func (h *DefaultMessageHandler) handleChat(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, chatMsg *pb.ChatMessage) error {
	// 创建聊天推送消息
	pushMsg, err := h.parser.CreatePushMessage(pb.MessageType_MSG_PUSH_CHAT_MSG, 0, "", "", &pb.ChatMessagePush{
		SenderId:   msg.Header.UserId,
		SenderName: fmt.Sprintf("User_%d", msg.Header.UserId), // TODO: 从用户服务获取用户名
		ChatType:   chatMsg.ChatType,
		TargetId:   chatMsg.TargetId,
		Content:    chatMsg.Content,
		SendTime:   msg.Header.Timestamp,
	})
	if err != nil {
		return err
	}

	// 根据聊天类型广播消息
	switch chatMsg.ChatType {
	case 0: // 世界聊天
		h.broadcaster.BroadcastToAll(pushMsg)
	case 1: // 房间聊天
		h.broadcaster.BroadcastToRoom(msg.Header.RoomId, pushMsg, 0)
	case 2: // 私聊
		h.broadcaster.BroadcastToUser(chatMsg.TargetId, pushMsg)
	}

	// 发送成功响应
	resp, err := h.parser.CreateResponse(msg, 0, "Chat message sent", nil)
	if err != nil {
		return err
	}

	data, err := h.parser.SerializeMessage(resp)
	if err != nil {
		return err
	}

	return h.broadcaster.SendMessage(conn, data)
}

// handleUserInfoQuery 处理用户信息查询
func (h *DefaultMessageHandler) handleUserInfoQuery(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, query *pb.UserInfoQuery) error {
	userID := query.UserId
	if userID == 0 {
		userID = msg.Header.UserId // 查询自己的信息
	}

	// TODO: 调用用户服务获取用户信息
	// 这里暂时返回模拟数据
	userInfo := map[string]interface{}{
		"user_id":  userID,
		"nickname": fmt.Sprintf("User_%d", userID),
		"level":    1,
		"coins":    1000,
		"status":   "online",
	}

	resp, err := h.parser.CreateResponse(msg, 0, "User info retrieved", userInfo)
	if err != nil {
		return err
	}

	data, err := h.parser.SerializeMessage(resp)
	if err != nil {
		return err
	}

	return h.broadcaster.SendMessage(conn, data)
}

// handleRoomListQuery 处理房间列表查询
func (h *DefaultMessageHandler) handleRoomListQuery(ctx context.Context, conn *websocket.Conn, msg *pb.WebSocketMessage, query *pb.RoomListQuery) error {
	// TODO: 调用大厅服务获取房间列表
	// 这里暂时返回模拟数据
	rooms := []*pb.RoomInfo{
		{
			RoomId:      "room_001",
			RoomName:    "德州扑克初级房",
			GameType:    "texas_poker",
			PlayerCount: 5,
			MaxPlayers:  9,
			RoomStatus:  1,
			CreateTime:  "2024-01-01 12:00:00",
		},
		{
			RoomId:      "room_002",
			RoomName:    "牛牛中级房",
			GameType:    "niu_niu",
			PlayerCount: 3,
			MaxPlayers:  6,
			RoomStatus:  1,
			CreateTime:  "2024-01-01 12:30:00",
		},
	}

	response := &pb.RoomListResponse{
		Rooms:      rooms,
		TotalCount: 2,
		Page:       query.Page,
		PageSize:   query.PageSize,
	}

	resp, err := h.parser.CreateResponse(msg, 0, "Room list retrieved", response)
	if err != nil {
		return err
	}

	data, err := h.parser.SerializeMessage(resp)
	if err != nil {
		return err
	}

	return h.broadcaster.SendMessage(conn, data)
}
