// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package user

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserResourcePreviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserResourcePreviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserResourcePreviewLogic {
	return &GetUserResourcePreviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserResourcePreviewLogic) GetUserResourcePreview(req *types.UserResourcePreviewRequest) (resp *types.UserResourcePreviewResponse, err error) {
	result, err := l.svcCtx.AppletUserRPC.GetUserResourcePreview(l.ctx, &pb.UserResourcePreviewRequest{Actor: resourceActor(l.ctx), UserID: req.UserID})
	if err != nil {
		return nil, err
	}
	return &types.UserResourcePreviewResponse{UserID: result.UserID, Username: result.Username, Files: result.Files, AIConversations: result.AIConversations, AIMessages: result.AIMessages, AIRuns: result.AIRuns, AIAvailable: result.AIAvailable, AIReason: result.AIReason, TransferMode: result.TransferMode}, nil
}
