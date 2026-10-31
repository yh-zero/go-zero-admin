package fileservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	filepb "go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type AddFileReferenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddFileReferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddFileReferenceLogic {
	return &AddFileReferenceLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *AddFileReferenceLogic) AddFileReference(in *filepb.AddFileReferenceRequest) (*filepb.NoDataResponse, error) {
	if err := validateReference(in.GetActor().GetUserID(), in.FileID, in.RefType, in.RefID); err != nil {
		return nil, err
	}
	if err := l.svcCtx.SysFileReferenceModel.AddReference(l.ctx, in.FileID, in.Actor.UserID, in.RefType, in.RefID); err != nil {
		return nil, fileModelError(err)
	}
	return &filepb.NoDataResponse{}, nil
}

func validateReference(actorId, fileId int64, refType string, refId int64) error {
	if actorId <= 0 {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if fileId <= 0 || refId <= 0 || (refType != "admin_avatar" && refType != "app_avatar") {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文件引用参数不合法")
	}
	return nil
}
