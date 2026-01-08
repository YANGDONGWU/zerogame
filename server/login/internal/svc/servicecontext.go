package svc

import (
	userpb "zerogame/pb/user"
	"zerogame/server/login/internal/config"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config  config.Config
	UserRpc userpb.UserServiceClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		UserRpc: userpb.NewUserServiceClient(
			zrpc.MustNewClient(c.UserRpc).Conn(),
		),
	}
}
