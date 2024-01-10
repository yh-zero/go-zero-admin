package dictionary

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSysDictionaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSysDictionaryLogic {
	return &UpdateSysDictionaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSysDictionaryLogic) UpdateSysDictionary(req *types.UpdateSysDictionaryRequest) (resp *types.MessageResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
