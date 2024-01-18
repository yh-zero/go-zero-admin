package dictionary

import (
	"context"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/jinzhu/copier"
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
	if req.Status == 0 {
		req.Status = 1
	}
	if req.Type == "" && req.ID == 0 {
		return nil, xerr.NewErrCodeMsg(900000, "参数错误,id或者type 要有一个！")
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

	return &types.GetSysDictionaryDetailsResponse{SysDictionary: typesSysDictionary}, nil
}
