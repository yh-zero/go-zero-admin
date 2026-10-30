package fileservicelogic

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	filepb "go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type DeleteFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFileLogic {
	return &DeleteFileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteFileLogic) DeleteFile(in *filepb.DeleteFileRequest) (*filepb.NoDataResponse, error) {
	var refCount int32
	var visibility, objectKey string
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &refCount,
		"SELECT ref_count FROM sys_file WHERE id=? AND deleted_at IS NULL", in.FileID)
	if err == nil && refCount > 0 {
		return nil, xerr.NewErrCodeMsg(400, fmt.Sprintf("文件被 %d 处引用，无法删除", refCount))
	}
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &visibility,
		"SELECT visibility FROM sys_file WHERE id=?", in.FileID)
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &objectKey,
		"SELECT object_key FROM sys_file WHERE id=?", in.FileID)
	if strings.TrimSpace(objectKey) == "" {
		return nil, xerr.NewErrCodeMsg(404, "文件不存在或已删除")
	}

	if l.svcCtx.Store != nil {
		if err := l.svcCtx.Store.Delete(l.ctx, objectKey, visibility); err != nil {
			l.Errorf("store delete error: %v", err)
			return nil, xerr.NewErrCodeMsg(500, "存储删除失败")
		}
	}
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE sys_file SET deleted_at=NOW() WHERE id=?", in.FileID)
	return &filepb.NoDataResponse{}, nil
}
