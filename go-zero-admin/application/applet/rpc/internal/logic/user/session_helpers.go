package userlogic

import (
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"time"
)

func sessionExpired() error { return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR) }

// Direct database checks make revocation effective on the next request.
func sessionQuery(db *gorm.DB, in *pb.SessionRequest) *gorm.DB {
	if in == nil {
		in = &pb.SessionRequest{}
	}
	query := db.Model(&model.SysUser{}).
		Where("sys_users.id = ? AND sys_users.enable = 1 AND sys_users.session_version = ? AND sys_users.authority_id = ?", in.UserID, in.SessionVersion, in.AuthorityId).
		Where("? > 0 AND ? > 0", in.UserID, in.SessionVersion).
		Where("EXISTS (SELECT 1 FROM sys_user_authority ua JOIN sys_authorities a ON a.authority_id = ua.sys_authority_authority_id AND a.deleted_at IS NULL WHERE ua.sys_user_id = sys_users.id AND ua.sys_authority_authority_id = sys_users.authority_id)")
	if in.SessionID != "" {
		query = query.Where("EXISTS (SELECT 1 FROM sys_device_sessions ds WHERE ds.id = ? AND ds.user_id = sys_users.id AND ds.authority_id = sys_users.authority_id AND ds.session_version = sys_users.session_version AND ds.revoked_at IS NULL AND ds.expires_at > ?)", in.SessionID, time.Now().UTC())
	}
	return query
}
