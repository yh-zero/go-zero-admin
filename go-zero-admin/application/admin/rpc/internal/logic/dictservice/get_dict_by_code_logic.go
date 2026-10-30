package dictservicelogic

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetDictByCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDictByCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDictByCodeLogic {
	return &GetDictByCodeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetDictByCodeLogic) GetDictByCode(in *adminpb.GetDictByCodeRequest) (*adminpb.GetDictByCodeResponse, error) {
	dict, err := l.svcCtx.SysDictModel.FindOneByCode(l.ctx, in.Code)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DICT_NOT_FOUND)
	}
	var items []*adminpb.DictItem
	_ = json.Unmarshal([]byte(dict.Items), &items)
	return &adminpb.GetDictByCodeResponse{Dict: &adminpb.DictInfo{
		Id: int64(dict.Id), Name: dict.Name, Code: dict.Code, Items: items,
		Description: dict.Description, Status: int32(dict.Status),
		CreatedAt: dict.CreatedAt.Unix(), UpdatedAt: dict.UpdatedAt.Unix(),
	}}, nil
}
