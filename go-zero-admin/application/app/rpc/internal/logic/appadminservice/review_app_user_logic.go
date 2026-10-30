package appadminservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ReviewAppUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReviewAppUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviewAppUserLogic {
	return &ReviewAppUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ReviewAppUserLogic) ReviewAppUser(in *apppb.AppReviewRequest) (*apppb.NoDataResponse, error) {
	// 状态白名单校验，防止写入非法状态值
	switch in.Status {
	case "active", "disabled", "rejected":
	default:
		return nil, xerr.NewErrCodeMsg(400, "非法审核状态："+in.Status)
	}

	user, err := l.svcCtx.AppUserModel.FindOne(l.ctx, uint64(in.UserID))
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "用户不存在")
	}
	user.Status = in.Status
	user.ReviewReason = in.Reason
	user.UpdatedAt = time.Now()
	err = l.svcCtx.AppUserModel.Update(l.ctx, user)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}
