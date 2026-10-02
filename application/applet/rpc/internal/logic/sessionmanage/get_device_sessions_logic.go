package sessionmanagelogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	userlogic "go-zero-admin/application/applet/rpc/internal/logic/user"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/result/xerr"
	"time"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDeviceSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDeviceSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeviceSessionsLogic {
	return &GetDeviceSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDeviceSessionsLogic) GetDeviceSessions(in *pb.DeviceSessionListRequest) (*pb.DeviceSessionListResponse, error) {
	if in == nil || in.Actor == nil || in.PageRequest == nil || in.UserID < 0 {
		return nil, xerr.NewErrCode(xerr.REUQEST_PARAM_ERROR)
	}
	valid, err := userlogic.NewCheckSessionLogic(l.ctx, l.svcCtx).CheckSession(in.Actor)
	if err != nil {
		return nil, err
	}
	if !valid.Valid {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	if in.Actor.AuthorityId != accessutil.AdminAuthorityID && in.UserID != 0 && in.UserID != in.Actor.UserID {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "无权查询其他用户会话")
	}
	target := in.UserID
	if in.Actor.AuthorityId != accessutil.AdminAuthorityID {
		target = in.Actor.UserID
	}
	offset, limit, err := accessutil.Page(in.PageRequest.PageNo, in.PageRequest.PageSize)
	if err != nil {
		return nil, err
	}
	query := l.svcCtx.DB.WithContext(l.ctx).Model(&model.SysDeviceSession{}).
		Joins("JOIN sys_users u ON u.id = sys_device_sessions.user_id AND u.deleted_at IS NULL AND u.enable = 1 AND u.authority_id = sys_device_sessions.authority_id AND u.session_version = sys_device_sessions.session_version").
		Where("sys_device_sessions.revoked_at IS NULL AND sys_device_sessions.expires_at > ?", time.Now().UTC()).
		Where("EXISTS (SELECT 1 FROM sys_user_authority ua JOIN sys_authorities a ON a.authority_id=ua.sys_authority_authority_id AND a.deleted_at IS NULL WHERE ua.sys_user_id=u.id AND ua.sys_authority_authority_id=u.authority_id)")
	if target > 0 {
		query = query.Where("sys_device_sessions.user_id = ?", target)
	}
	var total int64
	if err = query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []struct {
		model.SysDeviceSession
		Username string
	}
	if err = query.Select("sys_device_sessions.*,u.username").Order("sys_device_sessions.created_at DESC, sys_device_sessions.id DESC").Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	response := &pb.DeviceSessionListResponse{Total: total, List: make([]*pb.DeviceSession, 0, len(rows))}
	for _, row := range rows {
		response.List = append(response.List, &pb.DeviceSession{ID: row.ID, UserID: row.UserID, Username: row.Username, IP: row.IP, UserAgent: row.UserAgent, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), ExpiresAt: row.ExpiresAt.UTC().Format(time.RFC3339), Current: row.ID == in.Actor.SessionID})
	}
	return response, nil
}
