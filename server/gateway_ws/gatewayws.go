// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"zerogame/server/gateway_ws/internal/config"
	"zerogame/server/gateway_ws/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "/Users/o/work/go/zerogame/server/gateway_ws/etc/gatewayws-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 初始化日志
	logx.MustSetup(c.Log)

	// 创建服务上下文
	ctx := svc.NewServiceContext(c)

	// 启动WebSocket服务器
	if err := ctx.Start(); err != nil {
		logx.Errorf("Failed to start server: %v", err)
		os.Exit(1)
	}

	fmt.Printf("WebSocket Gateway started successfully!\n")
	fmt.Printf("WebSocket server listening on: %s:%d%s\n", c.WebSocket.Host, c.WebSocket.Port, c.WebSocket.Path)
	fmt.Printf("Max connections: %d\n", c.WebSocket.MaxConnections)
	fmt.Printf("Heartbeat interval: %d seconds\n", c.WebSocket.HeartbeatInterval)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nShutting down server...")

	// 优雅关闭
	ctx.Stop()

	fmt.Println("Server stopped.")
}
