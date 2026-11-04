// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package file

import (
	"context"
	"encoding/base64"
	"strings"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileLogic {
	return &UploadFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadFileLogic) UploadFile(req *types.UploadFileReq) (resp *types.UploadFileResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	userId, _ := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	if userId <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	const maxBytes = 10 << 20
	if len(req.ContentBase64) == 0 || len(req.ContentBase64) > base64.StdEncoding.EncodedLen(maxBytes) || strings.ContainsAny(req.ContentBase64, "\r\n") {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文件编码为空或超过 10 MiB 限制")
	}
	content, err := base64.StdEncoding.Strict().DecodeString(req.ContentBase64)
	if err != nil || len(content) == 0 || len(content) > maxBytes {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文件 Base64 内容不合法")
	}
	username, _ := l.ctx.Value(middleware.CtxKeyUsername).(string)
	deptId, _ := l.ctx.Value(middleware.CtxKeyDeptId).(int64)
	ip, _ := l.ctx.Value(middleware.CtxKeyClientIP).(string)
	userAgent, _ := l.ctx.Value(middleware.CtxKeyUserAgent).(string)
	rpcResp, err := l.svcCtx.FileRpc.UploadFile(l.ctx, &filepb.UploadFileRequest{
		Actor:   &filepb.Actor{UserID: userId, Username: username, DeptId: deptId, IP: ip, UserAgent: userAgent},
		Content: content, OriginalName: req.Name, Scene: req.Scene, Visibility: req.Visibility,
	})
	if err != nil {
		return nil, err
	}
	return &types.UploadFileResp{File: mapFileInfo(rpcResp.File)}, nil
}
