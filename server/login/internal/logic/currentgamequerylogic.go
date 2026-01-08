package logic

import (
	"context"

	"zerogame/pb/login"
	"zerogame/server/login/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CurrentGameQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCurrentGameQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CurrentGameQueryLogic {
	return &CurrentGameQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询玩家当前游戏
func (l *CurrentGameQueryLogic) CurrentGameQuery(in *login.CurrentGameQueryRequest) (*login.CurrentGameQueryResponse, error) {
	// todo: add your logic here and delete this line

	return &login.CurrentGameQueryResponse{}, nil
}
