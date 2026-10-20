package appaccountservicelogic

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/applet/rpc/internal/logic/avatarutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SetAppUserAvatarLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetAppUserAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAppUserAvatarLogic {
	return &SetAppUserAvatarLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SetAppUserAvatarLogic) SetAppUserAvatar(in *pb.AppAvatarRequest) (*pb.NoDataResponse, error) {
	if in == nil || in.UserID <= 0 {
		return nil, badInput("App用户参数无效")
	}
	nextURL := strings.TrimSpace(in.AvatarURL)
	if nextURL == "/default-avatar.svg" {
		nextURL = ""
	}
	if !utf8.ValidString(nextURL) || utf8.RuneCountInString(nextURL) > 1024 || strings.ContainsAny(nextURL, "\x00\r\n") {
		return nil, badInput("头像地址无效或超过1024个字符")
	}
	if nextURL != "" {
		parsed, err := url.Parse(nextURL)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
			return nil, badInput("头像地址须为HTTP或HTTPS地址")
		}
	}
	err := withAdmin(l.ctx, l.svcCtx, in.Actor, adminRoot+"avatar", "PUT", func(tx *gorm.DB, caller *model.SysUser) error {
		var user model.AppUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, in.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return badInput("App用户不存在")
			}
			return err
		}
		if err := avatarutil.Sync(l.ctx, tx, l.svcCtx.Config.Storage, avatarutil.AppUser, user.ID, user.AvatarURL, nextURL); err != nil {
			return err
		}
		if err := tx.Model(&user).Update("avatar_url", nextURL).Error; err != nil {
			return err
		}
		return audit.Record(l.ctx, tx, audit.Event{ActorID: caller.ID, ActorName: caller.Username, AuthorityID: in.Actor.AuthorityId,
			Module: "app-members", Action: "setAppUserAvatar", Object: "app-user:" + strconv.FormatInt(user.ID, 10), Path: adminRoot + "avatar", Method: "PUT"})
	})
	if err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
