package logic

import (
	"context"

	"zerogame/pb/login"
	"zerogame/server/login/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResetPasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 密码重置
func (l *ResetPasswordLogic) ResetPassword(in *login.PasswordResetRequest) (*login.PasswordResetResponse, error) {
	// todo: add your logic here and delete this line

	return &login.PasswordResetResponse{}, nil
}
