package appadminservicelogic

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/apptoken"
	"go-zero-admin/pkg/xerr"
)

type GrantMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGrantMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GrantMembershipLogic {
	return &GrantMembershipLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GrantMembershipLogic) GrantMembership(in *apppb.GrantMembershipRequest) (*apppb.NoDataResponse, error) {
	user, err := l.svcCtx.AppUserModel.FindOne(l.ctx, uint64(in.UserID))
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "用户不存在")
	}
	_ = user

	// request_id 幂等：已存在则直接成功
	var existingID int64
	err = l.svcCtx.DB.QueryRowCtx(l.ctx, &existingID,
		"SELECT id FROM app_membership_grant WHERE request_id=?", in.RequestID)
	if err == nil && existingID > 0 {
		return &apppb.NoDataResponse{}, nil
	}

	requestID := in.RequestID
	if requestID == "" {
		requestID = fmt.Sprintf("grant-%d-%d", in.UserID, time.Now().Unix())
	}
	_, err = l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO app_membership_grant (user_id, plan_code, benefits, request_id, granted_by, starts_at, expires_at) VALUES (?,?,?,?,?,NOW(),DATE_ADD(NOW(), INTERVAL ? DAY))",
		in.UserID, in.PlanCode, in.Benefits, requestID, in.Actor.UserId, in.DurationDays)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	// Phase 3: 主动失效会员缓存
	apptoken.DelVipCache(l.ctx, l.svcCtx.Redis, in.UserID)
	return &apppb.NoDataResponse{}, nil
}
