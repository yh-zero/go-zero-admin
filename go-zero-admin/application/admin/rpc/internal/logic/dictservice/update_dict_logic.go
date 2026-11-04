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
	var items *string
	if in.UpdateItems {
		values := in.Items
		if values == nil {
			values = []*adminpb.DictItem{}
		}
		encoded, err := json.Marshal(values)
		if err != nil {
			return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
		}
		value := string(encoded)
		items = &value
	}
	err := l.svcCtx.SysDictModel.UpdatePartial(l.ctx, uint64(in.DictId), model.DictPatch{Name: in.Name, Code: in.Code, Items: items, Description: in.Description, Status: in.Status})
	if errors.Is(err, sqlx.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.DICT_NOT_FOUND)
	}
	if errors.Is(err, model.ErrInvalidMutation) {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, err.Error())
	}
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.NoDataResponse{}, nil
}
