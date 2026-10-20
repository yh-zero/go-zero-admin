package fileresourceservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/datascope"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileLogic {
	return &UploadFileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// UploadFile validates the image, stores it, then registers metadata and audit.
// A failed metadata transaction triggers best-effort object cleanup.
func (l *UploadFileLogic) UploadFile(in *pb.UploadFileRequest) (*pb.UploadFileResponse, error) {
	if in == nil || in.Actor == nil {
		return nil, xerr.NewErrCodeMsg(300002, "上传请求无效")
	}
	// Reject invalid sessions before making the storage request.
	if _, err := datascope.Apply(l.ctx, l.svcCtx.DB.Model(&model.SysFileResource{}), in.Actor, "owner_id", "department_id"); err != nil {
		return nil, err
	}
	if in.Visibility != "public" && in.Visibility != "private" {
		return nil, xerr.NewErrCodeMsg(300002, "文件可见性只允许public或private")
	}
	name, err := normalizeUploadFilename(in.Filename)
	if err != nil {
		return nil, err
	}
	mime, err := detectAndValidateImage(in.Data)
	if err != nil {
		return nil, err
	}
	if in.Mime != "" && in.Mime != mime {
		return nil, xerr.NewErrCodeMsg(300002, "文件类型与实际内容不一致")
	}
	key, err := newImageKey(mime)
	if err != nil {
		return nil, err
	}
	storage, err := configuredStorage(l.svcCtx)
	if err != nil {
		return nil, err
	}
	if err = storage.Put(l.ctx, key, mime, in.Data, in.Visibility); err != nil {
		l.Error("object storage upload failed")
		return nil, xerr.NewErrCodeMsg(300002, "图片上传失败，请检查文件存储服务")
	}

	file := model.SysFileResource{ObjectKey: key, Name: name, Mime: mime,
		Size: int64(len(in.Data)), OwnerID: in.Actor.UserID, Visibility: in.Visibility, Status: "active"}
	if err := registerFileRecord(l.ctx, l.svcCtx.DB.DB, in.Actor, &file); err != nil {
		if delErr := storage.Delete(context.WithoutCancel(l.ctx), key, in.Visibility); delErr != nil {
			l.Errorf("orphan storage object requires cleanup key=%s: %v", key, delErr)
		}
		return nil, err
	}
	resp := &pb.UploadFileResponse{ExpiresIn: 0}
	resp.File, err = fileProto(l.ctx, l.svcCtx.DB.DB, &file)
	if err != nil {
		return nil, err
	}
	resp.Url, resp.ExpiresIn, err = storage.URL(l.ctx, key, in.Visibility)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300002, "图片已保存，访问地址生成失败")
	}
	return resp, nil
}
