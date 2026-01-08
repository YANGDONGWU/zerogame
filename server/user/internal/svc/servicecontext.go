package svc

import (
	loginpb "zerogame/pb/login"
	"zerogame/server/user/internal/config"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	LoginRpc loginpb.LoginServiceClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		LoginRpc: loginpb.NewLoginServiceClient(
			zrpc.MustNewClient(c.LoginRpc).Conn(),
		),
	}
}
