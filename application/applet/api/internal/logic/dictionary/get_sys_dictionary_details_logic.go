package dictionary

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSysDictionaryDetailsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSysDictionaryDetailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryDetailsLogic {
	return &GetSysDictionaryDetailsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSysDictionaryDetailsLogic) GetSysDictionaryDetails(req *types.GetSysDictionaryDetailsRequest) (resp *types.GetSysDictionaryDetailsResponse, err error) {
	fmt.Println("req=", req)
	if req.Status == 0 {
		req.Status = 1
	}
	sysDictionaryDetails, err := l.svcCtx.AppletDictionaryRPC.GetSysDictionaryDetails(l.ctx, &pb.GetSysDictionaryDetailsRequest{
		ID:     req.ID,
		Type:   req.Type,
		Status: req.Status,
	})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.GetSysDictionaryDetails err:%v", err)
		return nil, err
	}
	var typesSysDictionary types.SysDictionary
	_ = copier.Copy(&typesSysDictionary, sysDictionaryDetails.SysDictionary)
	fmt.Println("typesSysDictionary", typesSysDictionary)
	fmt.Println("sysDictionaryDetails", sysDictionaryDetails)

	return &types.GetSysDictionaryDetailsResponse{SysDictionary: typesSysDictionary}, nil
}