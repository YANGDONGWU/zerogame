package logic

import (
	"context"

	"zerogame/pb/login"
	"zerogame/server/login/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BindQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBindQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindQueryLogic {
	return &BindQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 账号绑定查询
func (l *BindQueryLogic) BindQuery(in *login.BindQueryRequest) (*login.BindQueryResponse, error) {
	// todo: add your logic here and delete this line

	return &login.BindQueryResponse{}, nil
}
