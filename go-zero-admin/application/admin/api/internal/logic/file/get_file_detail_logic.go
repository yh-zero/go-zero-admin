package file

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetFileDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileDetailLogic {
	return &GetFileDetailLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetFileDetailLogic) GetFileDetail(req *types.GetFileReq) (*types.GetFileDetailResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	userId, _ := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	username, _ := l.ctx.Value(middleware.CtxKeyUsername).(string)
	if userId <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	resp, err := l.svcCtx.FileRpc.GetFileDetail(l.ctx, &filepb.GetFileDetailRequest{
		Actor: &filepb.Actor{UserID: userId, Username: username}, FileID: req.Id,
	})
	if err != nil {
		return nil, err
	}
	references := make([]*types.FileReferenceInfo, 0, len(resp.References))
	for _, ref := range resp.References {
		references = append(references, &types.FileReferenceInfo{Id: ref.Id, RefType: ref.RefType, RefId: ref.RefId, CreatedBy: ref.CreatedBy, CreatedAt: ref.CreatedAt})
	}
	return &types.GetFileDetailResp{File: mapFileInfo(resp.File), References: references}, nil
}
