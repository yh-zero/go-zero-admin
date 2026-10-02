package userlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"gorm.io/gorm"
	"strconv"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *LogoutLogic) Logout(in *pb.SessionRequest) (*pb.NoDataResponse, error) {
	// Only revoke the version that the caller actually owns.
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		change := sessionQuery(tx, in).Update("session_version", gorm.Expr("session_version + 1"))
		if change.Error != nil {
			return change.Error
		}
		if change.RowsAffected == 0 {
			return nil
		}
		return audit.Record(l.ctx, tx, audit.Event{Module: "session", Action: "logoutAllDevices", Object: strconv.FormatInt(in.UserID, 10)})
	})
	return &pb.NoDataResponse{}, err
}
