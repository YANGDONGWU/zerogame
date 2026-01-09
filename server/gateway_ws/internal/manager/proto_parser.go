package manager

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"
	"zerogame/pb"
)

// ProtoMessageParser proto消息解析器（性能优化版本）
type ProtoMessageParser struct {
	logx.Logger
}

// NewProtoMessageParser 创建proto消息解析器
func NewProtoMessageParser() *ProtoMessageParser {
	return &ProtoMessageParser{}
}

// ParseMessage 解析proto格式的WebSocket消息
func (p *ProtoMessageParser) ParseMessage(data []byte) (*pb.WebSocketMessage, error) {
	var msg pb.WebSocketMessage
	if err := proto.Unmarshal(data, &msg); err != nil {
		p.Errorf("Failed to unmarshal proto message: %v", err)
		return nil, fmt.Errorf("invalid proto message format: %w", err)
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

// SerializeMessage 序列化proto消息
func (p *ProtoMessageParser) SerializeMessage(msg *pb.WebSocketMessage) ([]byte, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		p.Errorf("Failed to marshal proto message: %v", err)
		return nil, fmt.Errorf("failed to serialize proto message: %w", err)
	}
	return data, nil
}

// ParseMessageBody 根据消息类型解析消息体（proto版本）
func (p *ProtoMessageParser) ParseMessageBody(msg *pb.WebSocketMessage) (interface{}, error) {
	if msg.Header == nil {
		return nil, fmt.Errorf("message header is nil")
	}

	// 根据消息类型反序列化body
	switch msg.Header.MsgType {
	case pb.MessageType_MSG_HEARTBEAT:
		var body pb.Heartbeat
		if err := proto.Unmarshal(msg.Body, &body); err != nil {
			return nil, fmt.Errorf("failed to parse heartbeat: %w", err)
		}
		return &body, nil

	case pb.MessageType_MSG_LOGIN:
		var body pb.LoginMessage
		if err := proto.Unmarshal(msg.Body, &body); err != nil {
			return nil, fmt.Errorf("failed to parse login message: %w", err)
		}
		return &body, nil

	case pb.MessageType_MSG_JOIN_ROOM:
		var body pb.JoinRoomMessage
		if err := proto.Unmarshal(msg.Body, &body); err != nil {
			return nil, fmt.Errorf("failed to parse join room message: %w", err)
		}
		return &body, nil

	// 添加其他消息类型的处理...
	default:
		return nil, fmt.Errorf("unsupported message type: %d", msg.Header.MsgType)
	}
}

// SerializeMessageBody 根据消息类型序列化消息体（proto版本）
func (p *ProtoMessageParser) SerializeMessageBody(msgType pb.MessageType, body interface{}) ([]byte, error) {
	switch msgType {
	case pb.MessageType_MSG_HEARTBEAT:
		data, err := proto.Marshal(body.(*pb.Heartbeat))
		if err != nil {
			return nil, fmt.Errorf("failed to marshal heartbeat: %w", err)
		}
		return data, nil

	case pb.MessageType_MSG_LOGIN:
		data, err := proto.Marshal(body.(*pb.LoginMessage))
		if err != nil {
			return nil, fmt.Errorf("failed to marshal login message: %w", err)
		}
		return data, nil

	case pb.MessageType_MSG_JOIN_ROOM:
		data, err := proto.Marshal(body.(*pb.JoinRoomMessage))
		if err != nil {
			return nil, fmt.Errorf("failed to marshal join room message: %w", err)
		}
		return data, nil

	// 添加其他消息类型的处理...
	default:
		return nil, fmt.Errorf("unsupported message type for proto marshal: %d", msgType)
	}
}

// CreateResponse 创建proto响应消息
func (p *ProtoMessageParser) CreateResponse(reqMsg *pb.WebSocketMessage, code int32, msg string, data interface{}) (*pb.WebSocketMessage, error) {
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
		bodyData, err := p.SerializeMessageBody(pb.MessageType(code+1000), data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response data: %w", err)
		}
		resp.Body = bodyData
	}

	return resp, nil
}

// CreatePushMessage 创建proto推送消息
func (p *ProtoMessageParser) CreatePushMessage(msgType pb.MessageType, userID int32, roomID, gameID string, data interface{}) (*pb.WebSocketMessage, error) {
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
// 使用说明：
// ============================================================================
//
// 要使用proto序列化替代JSON，需要：
//
// 1. 前端使用proto编码发送消息：
//    const message = {
//      header: { msgType: 1, msgId: "001", timestamp: Date.now() },
//      body: LoginMessage.encode({ token: "xxx" }).finish()
//    }
//
// 2. 替换MessageParser为ProtoMessageParser：
//    parser := NewProtoMessageParser()
//
// 3. WebSocket消息类型设置为二进制：
//    conn.WriteMessage(websocket.BinaryMessage, protoData)
//
// 4. 性能对比：
//    - JSON: 易调试，可读性好，兼容性好
//    - Proto: 体积小(30-60%减少)，解析快，类型安全
//
// 推荐：开发环境用JSON，生产环境用Proto
// ============================================================================
