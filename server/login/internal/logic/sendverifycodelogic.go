package logic

import (
	"context"

	"zerogame/pb/login"
	"zerogame/server/login/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendVerifyCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendVerifyCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendVerifyCodeLogic {
	return &SendVerifyCodeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取验证码
func (l *SendVerifyCodeLogic) SendVerifyCode(in *login.VerifyCodeRequest) (*login.VerifyCodeResponse, error) {
	// todo: add your logic here and delete this line

	return &login.VerifyCodeResponse{}, nil
}
