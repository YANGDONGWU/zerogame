// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	loginpb "zerogame/pb/login"

	"zerogame/server/gateway_http/internal/svc"
	"zerogame/server/gateway_http/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	// 调用 login-rpc
	rpcResp, err := l.svcCtx.LoginRpc.Logon(l.ctx, &loginpb.LogonRequest{
		Accounts: req.Accounts,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}

	return &types.LoginResp{
		UserId: int64(rpcResp.UserId),
		Token:  rpcResp.Token,
	}, nil
}
