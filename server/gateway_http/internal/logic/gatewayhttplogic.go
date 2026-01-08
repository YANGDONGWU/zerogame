// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"zerogame/server/gateway_http/internal/svc"
	"zerogame/server/gateway_http/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type Gateway_httpLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGateway_httpLogic(ctx context.Context, svcCtx *svc.ServiceContext) *Gateway_httpLogic {
	return &Gateway_httpLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *Gateway_httpLogic) Gateway_http(req *types.Request) (resp *types.Response, err error) {
	// todo: add your logic here and delete this line

	return
}
