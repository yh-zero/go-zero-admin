// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package files

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddFileReferenceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 登记文件业务引用
func NewAddFileReferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddFileReferenceLogic {
	return &AddFileReferenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddFileReferenceLogic) AddFileReference(req *types.FileReferenceRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletFileRPC.AddFileReference(l.ctx, &pb.FileReferenceRequest{Actor: actorSession(l.ctx), FileID: req.FileId, ObjectType: req.ObjectType, ObjectID: req.ObjectId})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "文件引用已登记"}, nil
}
