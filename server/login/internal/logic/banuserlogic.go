package logic

import (
	"context"

	"zerogame/pb/login"
	"zerogame/server/login/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BanUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBanUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BanUserLogic {
	return &BanUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 封禁用户
func (l *BanUserLogic) BanUser(in *login.BanUserRequest) (*login.BanUserResponse, error) {
	// todo: add your logic here and delete this line

	return &login.BanUserResponse{}, nil
}
