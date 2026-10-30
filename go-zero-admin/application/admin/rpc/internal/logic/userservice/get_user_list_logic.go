package userservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetUserListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserListLogic {
	return &GetUserListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

type userRow struct {
	Id           int64  `db:"id"`
	Username     string `db:"username"`
	Nickname     string `db:"nickname"`
	RealName     string `db:"real_name"`
	Email        string `db:"email"`
	Phone        string `db:"phone"`
	DeptId       int64  `db:"dept_id"`
	AvatarFileId int64  `db:"avatar_file_id"`
	Gender       int32  `db:"gender"`
	Status       int32  `db:"status"`
	Remark       string `db:"remark"`
	LastLoginAt  int64  `db:"last_login_at"`
	LastLoginIp  string `db:"last_login_ip"`
	CreatedAt    int64  `db:"created_at"`
	UpdatedAt    int64  `db:"updated_at"`
}

func (l *GetUserListLogic) GetUserList(in *adminpb.GetUserListRequest) (*adminpb.GetUserListResponse, error) {
	where := "1=1"
	var args []interface{}
	if in.Username != "" {
		where += " AND username LIKE ?"
		args = append(args, "%"+in.Username+"%")
	}
	if in.Nickname != "" {
		where += " AND nickname LIKE ?"
		args = append(args, "%"+in.Nickname+"%")
	}
	if in.DeptId > 0 {
		where += " AND dept_id = ?"
		args = append(args, in.DeptId)
	}
	if in.Status > 0 {
		where += " AND status = ?"
		args = append(args, in.Status)
	}

	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total,
		"SELECT COUNT(*) FROM sys_user WHERE "+where+" AND deleted_at IS NULL", args...)

	page := in.Page.Page
	pageSize := in.Page.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var rows []userRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT id, username, nickname, real_name, email, phone, COALESCE(dept_id,0), COALESCE(avatar_file_id,0), gender, status, remark, COALESCE(UNIX_TIMESTAMP(last_login_at),0), last_login_ip, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM sys_user WHERE "+where+" AND deleted_at IS NULL ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		l.Errorf("query users error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}

	list := make([]*adminpb.UserInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &adminpb.UserInfo{
			Id: r.Id, Username: r.Username, Nickname: r.Nickname,
			RealName: r.RealName, Email: r.Email, Phone: r.Phone,
			DeptId: r.DeptId, AvatarFileId: r.AvatarFileId,
			Gender: r.Gender, Status: r.Status, Remark: r.Remark,
			LastLoginAt: r.LastLoginAt, LastLoginIP: r.LastLoginIp,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	return &adminpb.GetUserListResponse{
		List: list,
		Page: &adminpb.PageResponse{Total: total},
	}, nil
}
