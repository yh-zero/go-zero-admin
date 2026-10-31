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
	if in.KeepFileId != nil {
		if in.FileID != 0 || *in.KeepFileId < 0 {
			return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
		}
		if err := validateReference(in.GetActor().GetUserID(), 1, in.RefType, in.RefID); err != nil {
			return nil, err
		}
		if err := l.svcCtx.SysFileReferenceModel.ReconcileReferences(l.ctx, in.GetActor().GetUserID(), in.RefType, in.RefID, *in.KeepFileId); err != nil {
			return nil, fileModelError(err)
		}
		return &filepb.NoDataResponse{}, nil
	}
	if err := validateReference(in.GetActor().GetUserID(), in.FileID, in.RefType, in.RefID); err != nil {
		return nil, err
	}
	if err := l.svcCtx.SysFileReferenceModel.RemoveReference(l.ctx, in.FileID, in.Actor.UserID, in.RefType, in.RefID); err != nil {
		return nil, fileModelError(err)
	}
	return &filepb.NoDataResponse{}, nil
}
