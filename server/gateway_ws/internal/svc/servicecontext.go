// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"context"
	"sync"

	"zerogame/server/gateway_ws/internal/config"
	"zerogame/server/gateway_ws/internal/manager"
)

type ServiceContext struct {
	Config        config.Config
	WsServer      *manager.WebSocketServer
	serverCtx     context.Context
	cancelFunc    context.CancelFunc
	wg            sync.WaitGroup
}

func NewServiceContext(c config.Config) *ServiceContext {
	ctx, cancel := context.WithCancel(context.Background())

	// 根据配置选择序列化方式
	var wsServer *manager.WebSocketServer
	if c.WebSocket.SerializationFormat == "proto" {
		wsServer = manager.NewWebSocketServerWithParser(&c.WebSocket, manager.NewProtoMessageParser())
	} else {
		wsServer = manager.NewWebSocketServer(&c.WebSocket) // 默认JSON
	}

	return &ServiceContext{
		Config:     c,
		WsServer:   wsServer,
		serverCtx:  ctx,
		cancelFunc: cancel,
	}
}

// Start 启动服务
func (s *ServiceContext) Start() error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.WsServer.Start(s.serverCtx); err != nil {
			panic("Failed to start WebSocket server: " + err.Error())
		}
	}()
	return nil
}

// Stop 停止服务
func (s *ServiceContext) Stop() {
	s.cancelFunc()
	if err := s.WsServer.Stop(); err != nil {
		// 记录错误但不panic
		println("Error stopping WebSocket server:", err.Error())
	}
	s.wg.Wait()
}
