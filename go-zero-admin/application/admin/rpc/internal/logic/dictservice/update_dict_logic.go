package dictservicelogic

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
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
	itemsJson, _ := json.Marshal(in.Items)
	err := l.svcCtx.SysDictModel.UpdateActive(l.ctx, &model.SysDict{
		Id: uint64(in.DictId), Name: in.Name, Code: in.Code, Items: string(itemsJson),
		Description: in.Description, Status: int64(in.Status),
	})
	if errors.Is(err, sqlx.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.DICT_NOT_FOUND)
	}
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.NoDataResponse{}, nil
}
