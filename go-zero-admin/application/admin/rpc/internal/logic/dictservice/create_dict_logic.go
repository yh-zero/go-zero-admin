package dictservicelogic

import (
	"context"
	"encoding/json"
	"time"

	"go-zero-admin/application/admin/rpc/internal/model"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type CreateDictLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateDictLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDictLogic {
	return &CreateDictLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateDictLogic) CreateDict(in *adminpb.CreateDictRequest) (*adminpb.NoDataResponse, error) {
	_, err := l.svcCtx.SysDictModel.FindOneByCode(l.ctx, in.Code)
	if err == nil {
		return nil, xerr.NewErrCode(xerr.DICT_CODE_EXISTS)
	}
	itemsJson, _ := json.Marshal(in.Items)
	now := time.Now()
	_, err = l.svcCtx.SysDictModel.Insert(l.ctx, &model.SysDict{
		Name: in.Name, Code: in.Code, Items: string(itemsJson),
		Description: in.Description, Status: int64(in.Status),
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.NoDataResponse{}, nil
}
