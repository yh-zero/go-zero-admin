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

type TransferUserResourcesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTransferUserResourcesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TransferUserResourcesLogic {
	return &TransferUserResourcesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TransferUserResourcesLogic) TransferUserResources(req *types.TransferUserResourcesRequest) (resp *types.TransferUserResourcesResponse, err error) {
	result, err := l.svcCtx.AppletUserRPC.TransferUserResources(l.ctx, &pb.TransferUserResourcesRequest{Actor: resourceActor(l.ctx), UserID: req.UserID, TargetUserID: req.TargetUserID, IncludeFiles: req.IncludeFiles, IncludeAI: req.IncludeAI})
	if err != nil {
		return nil, err
	}
	return &types.TransferUserResourcesResponse{Files: result.Files, AIConversations: result.AIConversations, AIMessages: result.AIMessages, AIRuns: result.AIRuns, TransactionMode: result.TransactionMode}, nil
}
