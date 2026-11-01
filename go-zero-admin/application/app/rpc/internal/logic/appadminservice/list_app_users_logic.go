package appadminservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/application/file/rpc/filepb"
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
	list := make([]*apppb.AppAdminUserInfo, 0, len(rows))
	avatarUrls := make(map[int64]string)
	for _, r := range rows {
		info, err := adminUserInfo(l.ctx, l.svcCtx, &r, avatarUrls)
		if err != nil {
			return nil, err
		}
		list = append(list, info)
	}
	return &apppb.AppAdminListResponse{List: list, Page: &apppb.PageResponse{Total: total}}, nil
}

func adminUserInfo(ctx context.Context, s *svc.ServiceContext, r *model.AppUserRow, avatarUrls map[int64]string) (*apppb.AppAdminUserInfo, error) {
	info := &apppb.AppAdminUserInfo{Id: r.Id, Account: r.Account, IdentityKind: r.IdentityKind, Nickname: r.Nickname, AvatarFileId: r.AvatarFileId, Status: r.Status, ReviewReason: r.ReviewReason, EmailVerified: r.EmailVerified, PhoneVerified: r.PhoneVerified, LastLoginAt: r.LastLoginAt, CreatedAt: r.CreatedAt, Remark: r.Remark}
	if r.AvatarFileId <= 0 {
		return info, nil
	}
	if s.FileRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	url, ok := avatarUrls[r.AvatarFileId]
	if !ok {
		resp, err := s.FileRpc.GetFileURL(ctx, &filepb.GetFileURLRequest{PublicOnly: true, FileID: r.AvatarFileId})
		if err != nil {
			return nil, err
		}
		url = resp.URL
		if avatarUrls != nil {
			avatarUrls[r.AvatarFileId] = url
		}
	}
	info.AvatarUrl = url
	return info, nil
}

func appDataError(err error) error {
	if errors.Is(err, model.ErrCommittedCacheInvalidation) {
		return xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "修改已保存，但缓存刷新失败，请重试")
	}
	if errors.Is(err, model.ErrNotFound) {
		return xerr.NewErrCode(xerr.NOT_FOUND)
	}
	return xerr.NewErrCode(xerr.DB_ERROR)
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
