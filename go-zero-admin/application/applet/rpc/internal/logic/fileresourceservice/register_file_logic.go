package fileresourceservicelogic

import (
	"context"
	"path"
	"strings"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/datascope"
)

type RegisterFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterFileLogic {
	return &RegisterFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterFileLogic) RegisterFile(in *pb.RegisterFileRequest) (*pb.FileResource, error) {
	if in == nil {
		return nil, fileError("文件登记参数不能为空")
	}
	if err := validateFile(in.File); err != nil {
		return nil, err
	}
	if _, err := datascope.Apply(l.ctx, l.svcCtx.DB.Model(&model.SysFileResource{}), in.Actor, "owner_id", "department_id"); err != nil {
		return nil, err
	}
	file := model.SysFileResource{ObjectKey: in.File.ObjectKey, Name: path.Base(strings.ReplaceAll(in.File.Name, "\\", "/")), Mime: in.File.Mime,
		Size: in.File.Size, OwnerID: in.Actor.UserID, Visibility: in.File.Visibility, Status: "active"}
	if err := registerFileRecord(l.ctx, l.svcCtx.DB.DB, in.Actor, &file); err != nil {
		return nil, err
	}
	return fileProto(l.ctx, l.svcCtx.DB.DB, &file)
}
