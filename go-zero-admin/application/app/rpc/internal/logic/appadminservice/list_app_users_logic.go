package appadminservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"strings"
)

type ListAppUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAppUsersLogic(ctx context.Context, s *svc.ServiceContext) *ListAppUsersLogic {
	return &ListAppUsersLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *ListAppUsersLogic) ListAppUsers(in *apppb.AppAdminListRequest) (*apppb.AppAdminListResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := pagination(in.GetPage())
	if err != nil {
		return nil, err
	}
	if in.Status != "" && !validStatus(in.Status) {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if len(in.Account) > 128 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	rows, total, err := l.svcCtx.AppUserModel.List(l.ctx, in.Status, strings.TrimSpace(in.Account), page, size)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*apppb.AppUserInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &apppb.AppUserInfo{Id: r.Id, Account: r.Account, IdentityKind: r.IdentityKind, Nickname: r.Nickname, AvatarFileId: r.AvatarFileId, Status: r.Status, ReviewReason: r.ReviewReason, EmailVerified: r.EmailVerified, PhoneVerified: r.PhoneVerified, LastLoginAt: r.LastLoginAt, CreatedAt: r.CreatedAt})
	}
	return &apppb.AppAdminListResponse{List: list, Page: &apppb.PageResponse{Total: total}}, nil
}

func requireActor(a *apppb.Actor) error {
	if a.GetUserId() <= 0 {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	return nil
}
func validStatus(s string) bool {
	switch s {
	case "pending", "active", "disabled", "rejected":
		return true
	}
	return false
}
func pagination(p *apppb.PageRequest) (int64, int64, error) {
	page, size := p.GetPage(), p.GetPageSize()
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	if page > 1000000 {
		return 0, 0, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	return page, size, nil
}
func liveAppUser(ctx context.Context, s *svc.ServiceContext, id int64) (*model.AppUser, error) {
	if id <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	u, err := s.AppUserModel.FindOne(ctx, uint64(id))
	if errors.Is(err, model.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.NOT_FOUND)
	}
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if u.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.NOT_FOUND)
	}
	return u, nil
}
