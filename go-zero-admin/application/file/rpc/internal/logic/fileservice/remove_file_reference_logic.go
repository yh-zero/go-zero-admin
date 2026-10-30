package fileservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	filepb "go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type RemoveFileReferenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveFileReferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFileReferenceLogic {
	return &RemoveFileReferenceLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RemoveFileReferenceLogic) RemoveFileReference(in *filepb.RemoveFileReferenceRequest) (*filepb.NoDataResponse, error) {
	_, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"UPDATE sys_file SET ref_count = GREATEST(ref_count - 1, 0) WHERE id=? AND deleted_at IS NULL", in.FileID)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &filepb.NoDataResponse{}, nil
}
