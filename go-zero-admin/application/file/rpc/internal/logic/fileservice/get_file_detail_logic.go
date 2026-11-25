package fileservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetFileDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFileDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileDetailLogic {
	return &GetFileDetailLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetFileDetailLogic) GetFileDetail(in *filepb.GetFileDetailRequest) (*filepb.GetFileDetailResponse, error) {
	if in.GetActor().GetUserID() <= 0 || in.FileID <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	row, err := l.svcCtx.SysFileModel.FindLive(l.ctx, in.FileID)
	if err != nil {
		return nil, fileModelError(err)
	}
	if !row.UploaderId.Valid || row.UploaderId.Int64 != in.Actor.UserID {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	refs, err := l.svcCtx.SysFileReferenceModel.ListFileReferences(l.ctx, int64(row.Id))
	if err != nil {
		l.Errorf("query file references error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	references := make([]*filepb.FileReferenceInfo, 0, len(refs))
	for _, ref := range refs {
		references = append(references, &filepb.FileReferenceInfo{Id: ref.Id, RefType: ref.RefType, RefId: ref.RefId, CreatedBy: ref.CreatedBy, CreatedAt: ref.CreatedAt})
	}
	return &filepb.GetFileDetailResponse{File: &filepb.FileInfo{
		Id: int64(row.Id), Bucket: row.Bucket, ObjectKey: row.ObjectKey, OriginalName: row.OriginalName,
		Ext: row.Ext, Mime: row.Mime, Size: int64(row.Size), Sha256: row.Sha256, Visibility: row.Visibility,
		RefCount: int32(row.RefCount), UploaderID: row.UploaderId.Int64, UploaderName: in.Actor.Username,
		CreatedAt: row.CreatedAt.Unix(),
	}, References: references}, nil
}
