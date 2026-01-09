package manager

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

// ClientConnection WebSocket客户端连接
type ClientConnection struct {
	Conn        *websocket.Conn
	UserID      int32
	RoomID      string
	GameID      string
	LastHeartbeat time.Time
	ConnectedAt   time.Time
	mutex        sync.RWMutex
}

// UpdateHeartbeat 更新心跳时间
func (c *ClientConnection) UpdateHeartbeat() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.LastHeartbeat = time.Now()
}

// IsAlive 检查连接是否存活
func (c *ClientConnection) IsAlive(timeout time.Duration) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return time.Since(c.LastHeartbeat) < timeout
}

// ConnectionManager 连接管理器
type ConnectionManager struct {
	connections    map[*websocket.Conn]*ClientConnection // 连接映射
	userConnections map[int32]*websocket.Conn           // 用户ID到连接的映射
	roomConnections map[string]map[*websocket.Conn]bool // 房间到连接的映射
	mutex          sync.RWMutex
	maxConnections int
	logx.Logger

	// 性能优化
	connPool       sync.Pool // 对象池复用
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(maxConnections int) *ConnectionManager {
	cm := &ConnectionManager{
		connections:     make(map[*websocket.Conn]*ClientConnection),
		userConnections: make(map[int32]*websocket.Conn),
		roomConnections: make(map[string]map[*websocket.Conn]bool),
		maxConnections:  maxConnections,
	}

	// 初始化对象池
	cm.connPool.New = func() interface{} {
		return &ClientConnection{}
	}

	return cm
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(conn *websocket.Conn, userID int32) *ClientConnection {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 检查连接数限制
	if len(cm.connections) >= cm.maxConnections {
		cm.Errorf("Connection limit reached: %d", cm.maxConnections)
		return nil
	}

	clientConn := &ClientConnection{
		Conn:          conn,
		UserID:        userID,
		ConnectedAt:   time.Now(),
		LastHeartbeat: time.Now(),
	}

	cm.connections[conn] = clientConn
	cm.userConnections[userID] = conn

	cm.Infof("Added connection for user %d, total connections: %d", userID, len(cm.connections))
	return clientConn
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	clientConn, exists := cm.connections[conn]
	if !exists {
		return
	}

	// 从用户映射中移除
	delete(cm.userConnections, clientConn.UserID)

	// 从房间映射中移除
	if clientConn.RoomID != "" {
		if roomConns, exists := cm.roomConnections[clientConn.RoomID]; exists {
			delete(roomConns, conn)
			if len(roomConns) == 0 {
				delete(cm.roomConnections, clientConn.RoomID)
			}
		}
	}

	// 从连接映射中移除
	delete(cm.connections, conn)

	cm.Infof("Removed connection for user %d, total connections: %d", clientConn.UserID, len(cm.connections))
}

// GetConnection 获取用户连接
func (cm *ConnectionManager) GetConnection(userID int32) *websocket.Conn {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.userConnections[userID]
}

// GetClientConnection 获取客户端连接信息
func (cm *ConnectionManager) GetClientConnection(conn *websocket.Conn) *ClientConnection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.connections[conn]
}

// JoinRoom 用户加入房间
func (cm *ConnectionManager) JoinRoom(conn *websocket.Conn, roomID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	clientConn, exists := cm.connections[conn]
	if !exists {
		return
	}

	// 离开之前的房间
	if clientConn.RoomID != "" && clientConn.RoomID != roomID {
		cm.leaveRoomInternal(conn, clientConn.RoomID)
	}

	// 加入新房间
	clientConn.RoomID = roomID
	if cm.roomConnections[roomID] == nil {
		cm.roomConnections[roomID] = make(map[*websocket.Conn]bool)
	}
	cm.roomConnections[roomID][conn] = true

	cm.Infof("User %d joined room %s", clientConn.UserID, roomID)
}

// LeaveRoom 用户离开房间
func (cm *ConnectionManager) LeaveRoom(conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	clientConn, exists := cm.connections[conn]
	if !exists || clientConn.RoomID == "" {
		return
	}

	cm.leaveRoomInternal(conn, clientConn.RoomID)
	clientConn.RoomID = ""
}

// leaveRoomInternal 内部离开房间方法
func (cm *ConnectionManager) leaveRoomInternal(conn *websocket.Conn, roomID string) {
	if roomConns, exists := cm.roomConnections[roomID]; exists {
		delete(roomConns, conn)
		if len(roomConns) == 0 {
			delete(cm.roomConnections, roomID)
		}
	}
}

// GetRoomConnections 获取房间内所有连接
func (cm *ConnectionManager) GetRoomConnections(roomID string) []*websocket.Conn {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	roomConns, exists := cm.roomConnections[roomID]
	if !exists {
		return nil
	}

	connections := make([]*websocket.Conn, 0, len(roomConns))
	for conn := range roomConns {
		connections = append(connections, conn)
	}
	return connections
}

// GetAllConnections 获取所有连接
func (cm *ConnectionManager) GetAllConnections() []*websocket.Conn {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	connections := make([]*websocket.Conn, 0, len(cm.connections))
	for conn := range cm.connections {
		connections = append(connections, conn)
	}
	return connections
}

// GetConnectionCount 获取连接数量
func (cm *ConnectionManager) GetConnectionCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.connections)
}

// GetRoomCount 获取房间数量
func (cm *ConnectionManager) GetRoomCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.roomConnections)
}

// CleanupDeadConnections 清理死连接
func (cm *ConnectionManager) CleanupDeadConnections(timeout time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	deadConnections := make([]*websocket.Conn, 0)

	for conn, clientConn := range cm.connections {
		if now.Sub(clientConn.LastHeartbeat) > timeout {
			deadConnections = append(deadConnections, conn)
		}
	}

	for _, conn := range deadConnections {
		cm.removeConnectionInternal(conn)
	}

	if len(deadConnections) > 0 {
		cm.Infof("Cleaned up %d dead connections", len(deadConnections))
	}
}

// removeConnectionInternal 内部移除连接方法
func (cm *ConnectionManager) removeConnectionInternal(conn *websocket.Conn) {
	clientConn, exists := cm.connections[conn]
	if !exists {
		return
	}

	// 从用户映射中移除
	delete(cm.userConnections, clientConn.UserID)

	// 从房间映射中移除
	if clientConn.RoomID != "" {
		cm.leaveRoomInternal(conn, clientConn.RoomID)
	}

	// 从连接映射中移除
	delete(cm.connections, conn)

	// 关闭连接
	conn.Close()
}

// StartHeartbeatChecker 启动心跳检查器
func (cm *ConnectionManager) StartHeartbeatChecker(ctx context.Context, interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.CleanupDeadConnections(timeout)
		}
	}
}
