package fileservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	filepb "go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetFileURLLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFileURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileURLLogic {
	return &GetFileURLLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetFileURLLogic) GetFileURL(in *filepb.GetFileURLRequest) (*filepb.GetFileURLResponse, error) {
	var row struct {
		Visibility string `db:"visibility"`
		ObjectKey  string `db:"object_key"`
	}
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &row,
		"SELECT visibility, object_key FROM sys_file WHERE id=? AND deleted_at IS NULL", in.FileID)
	if err != nil {
		// 未找到与数据库故障分开报告，不再把故障伪装成 404
		if errors.Is(err, sqlx.ErrNotFound) || row.ObjectKey == "" {
			return nil, xerr.NewErrCodeMsg(404, "文件不存在")
		}
		l.Errorf("query file error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if row.ObjectKey == "" {
		return nil, xerr.NewErrCodeMsg(404, "文件不存在")
	}
	if l.svcCtx.Store == nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	// 期限以存储层真实签名为准：公开文件永久有效返回 0，私有返回实际签名 TTL
	urlStr, expiresAt, err := l.svcCtx.Store.URL(l.ctx, row.ObjectKey, row.Visibility)
	if err != nil {
		l.Errorf("generate file url error: %v", err)
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	return &filepb.GetFileURLResponse{URL: urlStr, ExpiresAt: expiresAt}, nil
}
