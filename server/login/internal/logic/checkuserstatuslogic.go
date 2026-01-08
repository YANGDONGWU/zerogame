package logic

import (
	"context"

	"zerogame/pb/login"
	"zerogame/server/login/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckUserStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckUserStatusLogic {
	return &CheckUserStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 检查用户状态
func (l *CheckUserStatusLogic) CheckUserStatus(in *login.CheckUserStatusRequest) (*login.CheckUserStatusResponse, error) {
	// todo: add your logic here and delete this line

	return &login.CheckUserStatusResponse{}, nil
}
