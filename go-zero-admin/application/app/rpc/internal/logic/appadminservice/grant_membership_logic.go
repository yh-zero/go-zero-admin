package appadminservicelogic

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"strings"
	"time"
)

type GrantMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGrantMembershipLogic(ctx context.Context, s *svc.ServiceContext) *GrantMembershipLogic {
	return &GrantMembershipLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *GrantMembershipLogic) GrantMembership(in *apppb.GrantMembershipRequest) (*apppb.NoDataResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	if in.DurationDays < 1 || in.DurationDays > 3650 || in.RequestID == "" || len(in.RequestID) > 64 || in.PlanCode == "" || len(in.PlanCode) > 32 || len(in.Benefits) > 512 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if _, err := liveAppUser(l.ctx, l.svcCtx, in.UserID); err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(strings.ToLower(in.RequestID)))
	lock := redis.NewRedisLock(l.svcCtx.Redis, fmt.Sprintf("app:membership:grant:%x", hash))
	lock.SetExpire(30)
	acquired, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if !acquired {
		return nil, xerr.NewErrCodeMsg(400, "授权正在处理，请稍后重试")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), time.Second)
		defer cancel()
		_, _ = lock.ReleaseCtx(ctx)
	}()
	existing, err := l.svcCtx.AppMembershipGrantModel.FindOneByRequestId(l.ctx, in.RequestID)
	if err == nil {
		if existing.RequestId != in.RequestID || existing.UserId != uint64(in.UserID) || existing.PlanCode != in.PlanCode || existing.Benefits != in.Benefits || !existing.GrantedBy.Valid || existing.GrantedBy.Int64 != in.Actor.UserId || existing.ExpiresAt.Sub(existing.StartsAt) != time.Duration(in.DurationDays)*24*time.Hour {
			return nil, xerr.NewErrCodeMsg(400, "请求编号已用于其他授权")
		}
		return &apppb.NoDataResponse{}, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	now := time.Now().Truncate(time.Second)
	_, err = l.svcCtx.AppMembershipGrantModel.Insert(l.ctx, &model.AppMembershipGrant{UserId: uint64(in.UserID), PlanCode: in.PlanCode, Benefits: in.Benefits, RequestId: in.RequestID, GrantedBy: sql.NullInt64{Int64: in.Actor.UserId, Valid: true}, StartsAt: now, ExpiresAt: now.Add(time.Duration(in.DurationDays) * 24 * time.Hour), CreatedAt: now, UpdatedAt: now})
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}
