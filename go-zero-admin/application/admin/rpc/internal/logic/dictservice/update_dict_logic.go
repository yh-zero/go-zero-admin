package dictservicelogic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type UpdateDictLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDictLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDictLogic {
	return &UpdateDictLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateDictLogic) UpdateDict(in *adminpb.UpdateDictRequest) (*adminpb.NoDataResponse, error) {
	dict, err := l.svcCtx.SysDictModel.FindOne(l.ctx, uint64(in.DictId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DICT_NOT_FOUND)
	}
	itemsJson, _ := json.Marshal(in.Items)
	dict.Name = in.Name
	dict.Code = in.Code
	dict.Items = string(itemsJson)
	dict.Description = in.Description
	dict.Status = int64(in.Status)
	dict.UpdatedAt = time.Now()
	err = l.svcCtx.SysDictModel.Update(l.ctx, dict)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.NoDataResponse{}, nil
}
