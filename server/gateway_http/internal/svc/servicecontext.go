// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	loginpb "zerogame/pb/login"
	userpb "zerogame/pb/user"
	"zerogame/server/gateway_http/internal/config"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	LoginRpc loginpb.LoginServiceClient
	UserRpc  userpb.UserServiceClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		LoginRpc: loginpb.NewLoginServiceClient(
			zrpc.MustNewClient(c.LoginRpc).Conn(),
		),
		UserRpc: userpb.NewUserServiceClient(
			zrpc.MustNewClient(c.UserRpc).Conn(),
		),
	}
}
