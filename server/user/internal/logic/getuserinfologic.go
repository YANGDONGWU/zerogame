package logic

import (
	"context"
	"fmt"
	"zerogame/pb/user"
	"zerogame/server/user/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserInfoLogic) GetUserInfo(in *user.GetUserInfoRequest) (*user.GetUserInfoResponse, error) {
	fmt.Printf("GetUserInfo req:%+v", in)

	//req := &loginpb.LogonRequest{
	//	Accounts: "allen",
	//}
	//rsp, err := l.svcCtx.LoginRpc.Logon(l.ctx, req)
	//if err != nil {
	//	fmt.Println(err)
	//	return nil, err
	//}
	//
	//fmt.Println("rsp:", rsp)

	return &user.GetUserInfoResponse{
		UserId:   123456,
		Nickname: "allen",
		Gold:     1111,
	}, nil
}
