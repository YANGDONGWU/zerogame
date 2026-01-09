// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import "github.com/zeromicro/go-zero/rest"

// WebSocket配置
type WebSocketConfig struct {
	Host               string `json:",default=0.0.0.0"`     // WebSocket服务器主机
	Port               int    `json:",default=8888"`        // WebSocket服务器端口
	Path               string `json:",default=/ws"`         // WebSocket路径
	ReadTimeout        int    `json:",default=60"`          // 读取超时时间（秒）
	WriteTimeout       int    `json:",default=60"`          // 写入超时时间（秒）
	MaxMessageSize     int64  `json:",default=65536"`       // 最大消息大小（字节）
	HeartbeatInterval  int    `json:",default=30"`          // 心跳间隔（秒）
	HeartbeatTimeout   int    `json:",default=90"`          // 心跳超时时间（秒）
	MaxConnections     int    `json:",default=10000"`       // 最大连接数
	EnableCompression  bool   `json:",default=true"`        // 启用压缩
	AllowedOrigins     []string `json:",optional"`          // 允许的源域名
	SerializationFormat string `json:",default=json"`       // 序列化方式: "json" 或 "proto"
}

type Config struct {
	rest.RestConf
	WebSocket WebSocketConfig `json:",optional"`
}
