package manager

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"zerogame/pb"
)

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	Message     *pb.WebSocketMessage
	TargetUsers []int32  // 指定用户ID列表，为空表示广播到所有用户
	TargetRooms []string // 指定房间ID列表，为空表示不限制房间
	ExcludeUser int32    // 排除的用户ID
}

// Broadcaster 广播器
type Broadcaster struct {
	connMgr    *ConnectionManager
	parser     MessageParserInterface
	messageCh  chan *BroadcastMessage
	workerPool *WorkerPool
	logx.Logger
}

// WorkerPool 工作池
type WorkerPool struct {
	workers   int
	taskQueue chan func()
	wg        sync.WaitGroup
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	wp := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan func(), queueSize),
	}

	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

// worker 工作协程
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for task := range wp.taskQueue {
		task()
	}
}

// Submit 提交任务
func (wp *WorkerPool) Submit(task func()) {
	select {
	case wp.taskQueue <- task:
	default:
		// 队列满时直接执行，避免阻塞
		go task()
	}
}

// Shutdown 关闭工作池
func (wp *WorkerPool) Shutdown() {
	close(wp.taskQueue)
	wp.wg.Wait()
}

// NewBroadcaster 创建广播器
func NewBroadcaster(connMgr *ConnectionManager, parser MessageParserInterface, workers int) *Broadcaster {
	return &Broadcaster{
		connMgr:    connMgr,
		parser:     parser,
		messageCh:  make(chan *BroadcastMessage, 1000),
		workerPool: NewWorkerPool(workers, 10000),
	}
}

// Start 启动广播器
func (b *Broadcaster) Start(ctx context.Context) {
	go b.processMessages(ctx)
}

// Stop 停止广播器
func (b *Broadcaster) Stop() {
	close(b.messageCh)
	b.workerPool.Shutdown()
}

// Broadcast 广播消息
func (b *Broadcaster) Broadcast(msg *BroadcastMessage) {
	select {
	case b.messageCh <- msg:
	default:
		b.Errorf("Broadcast message queue full, dropping message")
	}
}

// BroadcastToUser 广播给指定用户
func (b *Broadcaster) BroadcastToUser(userID int32, msg *pb.WebSocketMessage) {
	b.Broadcast(&BroadcastMessage{
		Message:     msg,
		TargetUsers: []int32{userID},
	})
}

// BroadcastToUsers 广播给多个用户
func (b *Broadcaster) BroadcastToUsers(userIDs []int32, msg *pb.WebSocketMessage) {
	b.Broadcast(&BroadcastMessage{
		Message:     msg,
		TargetUsers: userIDs,
	})
}

// BroadcastToRoom 广播给房间内所有用户
func (b *Broadcaster) BroadcastToRoom(roomID string, msg *pb.WebSocketMessage, excludeUser int32) {
	b.Broadcast(&BroadcastMessage{
		Message:     msg,
		TargetRooms: []string{roomID},
		ExcludeUser: excludeUser,
	})
}

// BroadcastToRooms 广播给多个房间
func (b *Broadcaster) BroadcastToRooms(roomIDs []string, msg *pb.WebSocketMessage) {
	b.Broadcast(&BroadcastMessage{
		Message:     msg,
		TargetRooms: roomIDs,
	})
}

// BroadcastToAll 广播给所有用户
func (b *Broadcaster) BroadcastToAll(msg *pb.WebSocketMessage) {
	b.Broadcast(&BroadcastMessage{
		Message: msg,
	})
}

// processMessages 处理广播消息
func (b *Broadcaster) processMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-b.messageCh:
			if !ok {
				return
			}
			b.workerPool.Submit(func() {
				b.processBroadcastMessage(msg)
			})
		}
	}
}

// processBroadcastMessage 处理单个广播消息
func (b *Broadcaster) processBroadcastMessage(broadcastMsg *BroadcastMessage) {
	// 序列化消息
	data, err := b.parser.SerializeMessage(broadcastMsg.Message)
	if err != nil {
		b.Errorf("Failed to serialize broadcast message: %v", err)
		return
	}

	var targetConns []*websocket.Conn

	// 根据广播类型获取目标连接
	if len(broadcastMsg.TargetUsers) > 0 {
		// 指定用户广播
		for _, userID := range broadcastMsg.TargetUsers {
			if conn := b.connMgr.GetConnection(userID); conn != nil {
				targetConns = append(targetConns, conn)
			}
		}
	} else if len(broadcastMsg.TargetRooms) > 0 {
		// 房间广播
		roomConns := make(map[*websocket.Conn]bool)
		for _, roomID := range broadcastMsg.TargetRooms {
			if conns := b.connMgr.GetRoomConnections(roomID); conns != nil {
				for _, conn := range conns {
					roomConns[conn] = true
				}
			}
		}

		for conn := range roomConns {
			// 检查是否需要排除用户
			if broadcastMsg.ExcludeUser > 0 {
				if clientConn := b.connMgr.GetClientConnection(conn); clientConn != nil {
					if clientConn.UserID == broadcastMsg.ExcludeUser {
						continue
					}
				}
			}
			targetConns = append(targetConns, conn)
		}
	} else {
		// 全员广播
		targetConns = b.connMgr.GetAllConnections()
		// 排除指定用户
		if broadcastMsg.ExcludeUser > 0 {
			filteredConns := make([]*websocket.Conn, 0, len(targetConns))
			for _, conn := range targetConns {
				if clientConn := b.connMgr.GetClientConnection(conn); clientConn != nil {
					if clientConn.UserID != broadcastMsg.ExcludeUser {
						filteredConns = append(filteredConns, conn)
					}
				}
			}
			targetConns = filteredConns
		}
	}

	// 发送消息给所有目标连接
	sentCount := 0
	for _, conn := range targetConns {
		if err := b.sendMessage(conn, data); err != nil {
			b.Errorf("Failed to send message to connection: %v", err)
			continue
		}
		sentCount++
	}

	b.Infof("Broadcast message sent to %d/%d connections", sentCount, len(targetConns))
}

// sendMessage 发送消息到单个连接
func (b *Broadcaster) sendMessage(conn *websocket.Conn, data []byte) error {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, data)
}

// SendHeartbeatResponse 发送心跳响应
func (b *Broadcaster) SendHeartbeatResponse(conn *websocket.Conn, clientTime int64) error {
	heartbeat := &pb.Heartbeat{
		ClientTime: clientTime,
	}

	msg, err := b.parser.CreatePushMessage(pb.MessageType_MSG_HEARTBEAT, 0, "", "", heartbeat)
	if err != nil {
		return err
	}

	data, err := b.parser.SerializeMessage(msg)
	if err != nil {
		return err
	}

	return b.sendMessage(conn, data)
}

// SendErrorResponse 发送错误响应
func (b *Broadcaster) SendErrorResponse(conn *websocket.Conn, reqMsg *pb.WebSocketMessage, code int32, errMsg string) error {
	resp, err := b.parser.CreateResponse(reqMsg, code, errMsg, nil)
	if err != nil {
		return err
	}

	data, err := b.parser.SerializeMessage(resp)
	if err != nil {
		return err
	}

	return b.sendMessage(conn, data)
}

// SendMessage 发送消息到指定连接
func (b *Broadcaster) SendMessage(conn *websocket.Conn, data []byte) error {
	return b.sendMessage(conn, data)
}
