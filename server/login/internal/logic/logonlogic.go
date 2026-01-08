package logic

import (
	"context"
	"fmt"
	userpb "zerogame/pb/user"

	"zerogame/pb/login"
	"zerogame/server/login/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogonLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogonLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogonLogic {
	return &LogonLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 登录
func (l *LogonLogic) Logon(in *login.LogonRequest) (*login.LogonResponse, error) {
	fmt.Printf("Logon req:%+v", in)

	req := &userpb.GetUserInfoRequest{}
	rsp, err := l.svcCtx.UserRpc.GetUserInfo(l.ctx, req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	fmt.Println("rsp:", rsp)

	return &login.LogonResponse{
		UserId:      123456,
		ErrorCode:   0,
		ConfineTime: "",
		Token:       rsp.Nickname,
		Skin:        "",
		UiType:      "",
		Versions:    "",
		LoginCount:  0,
	}, nil
}
