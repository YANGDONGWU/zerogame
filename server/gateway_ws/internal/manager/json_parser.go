package manager

import (
	"encoding/json"
	"fmt"

	"zerogame/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

// MessageParserInterface 消息解析器接口
type MessageParserInterface interface {
	ParseMessage(data []byte) (*pb.WebSocketMessage, error)
	SerializeMessage(msg *pb.WebSocketMessage) ([]byte, error)
	ParseMessageBody(msg *pb.WebSocketMessage) (interface{}, error)
	SerializeMessageBody(msgType pb.MessageType, body interface{}) ([]byte, error)
	CreateResponse(reqMsg *pb.WebSocketMessage, code int32, msg string, data interface{}) (*pb.WebSocketMessage, error)
	CreatePushMessage(msgType pb.MessageType, userID int32, roomID, gameID string, data interface{}) (*pb.WebSocketMessage, error)
}

// MessageParser 消息解析器
type MessageParser struct {
	logx.Logger
	parsers map[pb.MessageType]MessageBodyParser
}

// MessageBodyParser 消息体解析器接口
type MessageBodyParser interface {
	Parse(data []byte) (interface{}, error)
	GetMessageType() pb.MessageType
}

// NewMessageParser 创建消息解析器
func NewMessageParser() *MessageParser {
	parser := &MessageParser{
		parsers: make(map[pb.MessageType]MessageBodyParser),
	}
	parser.registerDefaultParsers()
	return parser
}

// registerDefaultParsers 注册默认的解析器
func (p *MessageParser) registerDefaultParsers() {
	// 注册所有消息类型的解析器
	parsers := []MessageBodyParser{
		&HeartbeatParser{},
		&LoginMessageParser{},
		&LogoutMessageParser{},
		&JoinRoomMessageParser{},
		&LeaveRoomMessageParser{},
		&GameActionMessageParser{},
		&ChatMessageParser{},
		&UserInfoQueryParser{},
		&RoomListQueryParser{},
	}

	for _, parser := range parsers {
		p.parsers[parser.GetMessageType()] = parser
	}
}

// ParseMessage 解析WebSocket消息
func (p *MessageParser) ParseMessage(data []byte) (*pb.WebSocketMessage, error) {
	var msg pb.WebSocketMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		p.Errorf("Failed to unmarshal message: %v", err)
		return nil, fmt.Errorf("invalid message format: %w", err)
	}

	// 验证消息头
	if msg.Header == nil {
		return nil, fmt.Errorf("message header is required")
	}

	if msg.Header.MsgType < 0 || msg.Header.MsgType > 105 {
		return nil, fmt.Errorf("invalid message type: %d", msg.Header.MsgType)
	}

	return &msg, nil
}

// SerializeMessage 序列化消息
func (p *MessageParser) SerializeMessage(msg *pb.WebSocketMessage) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		p.Errorf("Failed to marshal message: %v", err)
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}
	return data, nil
}

// ParseMessageBody 根据消息类型解析消息体
func (p *MessageParser) ParseMessageBody(msg *pb.WebSocketMessage) (interface{}, error) {
	if msg.Header == nil {
		return nil, fmt.Errorf("message header is nil")
	}

	parser, exists := p.parsers[msg.Header.MsgType]
	if !exists {
		return nil, fmt.Errorf("unsupported message type: %d", msg.Header.MsgType)
	}

	return parser.Parse(msg.Body)
}

// SerializeMessageBody 根据消息类型序列化消息体
func (p *MessageParser) SerializeMessageBody(msgType pb.MessageType, body interface{}) ([]byte, error) {
	switch msgType {
	case pb.MessageType_MSG_HEARTBEAT:
		return json.Marshal(body.(*pb.Heartbeat))
	case pb.MessageType_MSG_LOGIN:
		return json.Marshal(body.(*pb.LoginMessage))
	case pb.MessageType_MSG_LOGOUT:
		return json.Marshal(body.(*pb.LogoutMessage))
	case pb.MessageType_MSG_JOIN_ROOM:
		return json.Marshal(body.(*pb.JoinRoomMessage))
	case pb.MessageType_MSG_LEAVE_ROOM:
		return json.Marshal(body.(*pb.LeaveRoomMessage))
	case pb.MessageType_MSG_GAME_ACTION:
		return json.Marshal(body.(*pb.GameActionMessage))
	case pb.MessageType_MSG_CHAT:
		return json.Marshal(body.(*pb.ChatMessage))
	case pb.MessageType_MSG_USER_INFO_QUERY:
		return json.Marshal(body.(*pb.UserInfoQuery))
	case pb.MessageType_MSG_ROOM_LIST_QUERY:
		return json.Marshal(body.(*pb.RoomListQuery))
	case pb.MessageType_MSG_PUSH_GAME_STATE:
		return json.Marshal(body.(*pb.GameStatePush))
	case pb.MessageType_MSG_PUSH_ROOM_INFO:
		return json.Marshal(body.(*pb.RoomInfoPush))
	case pb.MessageType_MSG_PUSH_USER_UPDATE:
		return json.Marshal(body.(*pb.UserUpdatePush))
	case pb.MessageType_MSG_PUSH_SYSTEM_MSG:
		return json.Marshal(body.(*pb.SystemMessagePush))
	case pb.MessageType_MSG_PUSH_CHAT_MSG:
		return json.Marshal(body.(*pb.ChatMessagePush))
	case pb.MessageType_MSG_PUSH_BROADCAST:
		return json.Marshal(body.(*pb.BroadcastMessage))
	default:
		return nil, fmt.Errorf("unsupported message type: %d", msgType)
	}
}

// CreateResponse 创建响应消息
func (p *MessageParser) CreateResponse(reqMsg *pb.WebSocketMessage, code int32, msg string, data interface{}) (*pb.WebSocketMessage, error) {
	resp := &pb.WebSocketMessage{
		Header: &pb.MessageHeader{
			MsgType:   pb.MessageType(code + 1000), // 响应消息类型 = 请求类型 + 1000
			MsgId:     reqMsg.Header.MsgId,
			Timestamp: reqMsg.Header.Timestamp,
			UserId:    reqMsg.Header.UserId,
			RoomId:    reqMsg.Header.RoomId,
			GameId:    reqMsg.Header.GameId,
		},
	}

	// 如果有响应数据，序列化
	if data != nil {
		bodyData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response data: %w", err)
		}
		resp.Body = bodyData
	}

	return resp, nil
}

// CreatePushMessage 创建推送消息
func (p *MessageParser) CreatePushMessage(msgType pb.MessageType, userID int32, roomID, gameID string, data interface{}) (*pb.WebSocketMessage, error) {
	pushMsg := &pb.WebSocketMessage{
		Header: &pb.MessageHeader{
			MsgType:   msgType,
			Timestamp: 0, // 由调用方设置
			UserId:    userID,
			RoomId:    roomID,
			GameId:    gameID,
		},
	}

	// 序列化推送数据
	if data != nil {
		bodyData, err := p.SerializeMessageBody(msgType, data)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize push data: %w", err)
		}
		pushMsg.Body = bodyData
	}

	return pushMsg, nil
}

// ============================================================================
// 消息体解析器实现
// ============================================================================

type HeartbeatParser struct{}

func (p *HeartbeatParser) GetMessageType() pb.MessageType { return pb.MessageType_MSG_HEARTBEAT }
func (p *HeartbeatParser) Parse(data []byte) (interface{}, error) {
	var msg pb.Heartbeat
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse heartbeat: %w", err)
	}
	return &msg, nil
}

type LoginMessageParser struct{}

func (p *LoginMessageParser) GetMessageType() pb.MessageType { return pb.MessageType_MSG_LOGIN }
func (p *LoginMessageParser) Parse(data []byte) (interface{}, error) {
	var msg pb.LoginMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse login message: %w", err)
	}
	return &msg, nil
}

type LogoutMessageParser struct{}

func (p *LogoutMessageParser) GetMessageType() pb.MessageType { return pb.MessageType_MSG_LOGOUT }
func (p *LogoutMessageParser) Parse(data []byte) (interface{}, error) {
	var msg pb.LogoutMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse logout message: %w", err)
	}
	return &msg, nil
}

type JoinRoomMessageParser struct{}

func (p *JoinRoomMessageParser) GetMessageType() pb.MessageType { return pb.MessageType_MSG_JOIN_ROOM }
func (p *JoinRoomMessageParser) Parse(data []byte) (interface{}, error) {
	var msg pb.JoinRoomMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse join room message: %w", err)
	}
	return &msg, nil
}

type LeaveRoomMessageParser struct{}

func (p *LeaveRoomMessageParser) GetMessageType() pb.MessageType {
	return pb.MessageType_MSG_LEAVE_ROOM
}
func (p *LeaveRoomMessageParser) Parse(data []byte) (interface{}, error) {
	var msg pb.LeaveRoomMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse leave room message: %w", err)
	}
	return &msg, nil
}

type GameActionMessageParser struct{}

func (p *GameActionMessageParser) GetMessageType() pb.MessageType {
	return pb.MessageType_MSG_GAME_ACTION
}
func (p *GameActionMessageParser) Parse(data []byte) (interface{}, error) {
	var msg pb.GameActionMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse game action message: %w", err)
	}
	return &msg, nil
}

type ChatMessageParser struct{}

func (p *ChatMessageParser) GetMessageType() pb.MessageType { return pb.MessageType_MSG_CHAT }
func (p *ChatMessageParser) Parse(data []byte) (interface{}, error) {
	var msg pb.ChatMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse chat message: %w", err)
	}
	return &msg, nil
}

type UserInfoQueryParser struct{}

func (p *UserInfoQueryParser) GetMessageType() pb.MessageType {
	return pb.MessageType_MSG_USER_INFO_QUERY
}
func (p *UserInfoQueryParser) Parse(data []byte) (interface{}, error) {
	var msg pb.UserInfoQuery
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse user info query: %w", err)
	}
	return &msg, nil
}

type RoomListQueryParser struct{}

func (p *RoomListQueryParser) GetMessageType() pb.MessageType {
	return pb.MessageType_MSG_ROOM_LIST_QUERY
}
func (p *RoomListQueryParser) Parse(data []byte) (interface{}, error) {
	var msg pb.RoomListQuery
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse room list query: %w", err)
	}
	return &msg, nil
}
