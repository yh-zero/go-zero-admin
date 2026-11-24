package file

import (
	"context"
	"io"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadMultipartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadMultipartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadMultipartLogic {
	return &UploadMultipartLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UploadMultipartLogic) UploadMultipart(req *types.UploadMultipartReq, fileName string, content io.Reader) (*types.UploadFileResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	userId, _ := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	if userId <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	username, _ := l.ctx.Value(middleware.CtxKeyUsername).(string)
	deptId, _ := l.ctx.Value(middleware.CtxKeyDeptId).(int64)
	ip, _ := l.ctx.Value(middleware.CtxKeyClientIP).(string)
	userAgent, _ := l.ctx.Value(middleware.CtxKeyUserAgent).(string)
	stream, err := l.svcCtx.FileRpc.UploadFileStream(l.ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.CloseSend() }()
	if err = stream.Send(&filepb.UploadFileChunk{
		Actor:        &filepb.Actor{UserID: userId, Username: username, DeptId: deptId, IP: ip, UserAgent: userAgent},
		OriginalName: fileName, Scene: req.Scene, Visibility: req.Visibility,
	}); err != nil {
		return nil, err
	}
	buffer := make([]byte, 256<<10)
	for {
		n, readErr := content.Read(buffer)
		if n > 0 {
			if err = stream.Send(&filepb.UploadFileChunk{Content: buffer[:n]}); err != nil {
				return nil, err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	response, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	return &types.UploadFileResp{File: mapFileInfo(response.File)}, nil
}
