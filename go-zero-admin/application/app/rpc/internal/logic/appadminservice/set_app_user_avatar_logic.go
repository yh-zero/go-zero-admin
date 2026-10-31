package appadminservicelogic

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/xerr"
)

type SetAppUserAvatarLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetAppUserAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAppUserAvatarLogic {
	return &SetAppUserAvatarLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *SetAppUserAvatarLogic) SetAppUserAvatar(in *apppb.AppSetAvatarRequest) (*apppb.NoDataResponse, error) {
	if err := requireActor(in.GetActor()); err != nil {
		return nil, err
	}
	if in.UserID <= 0 || in.FileID < 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.FileRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	actor := &filepb.Actor{UserID: in.Actor.GetUserId(), Username: in.Actor.GetUsername(), DeptId: in.Actor.GetDeptId(), IP: in.Actor.GetIP(), UserAgent: in.Actor.GetUserAgent()}
	attempted := false
	err := l.svcCtx.AppUserModel.SetAvatar(l.ctx, uint64(in.UserID), in.FileID, func(ctx context.Context, oldId, newId int64) error {
		if newId <= 0 || oldId == newId {
			return nil
		}
		// App 头像只允许公开文件，不能把 App 用户 ID 当成后台文件上传者。
		if _, err := l.svcCtx.FileRpc.GetFileURL(ctx, &filepb.GetFileURLRequest{FileID: newId, PublicOnly: true}); err != nil {
			return avatarFileError(err)
		}
		attempted = true
		_, err := l.svcCtx.FileRpc.AddFileReference(ctx, &filepb.AddFileReferenceRequest{Actor: actor, FileID: newId, RefType: "app_avatar", RefID: in.UserID})
		if err != nil {
			return avatarFileError(err)
		}
		return nil
	})
	if err == nil || attempted {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), 4*time.Second)
		defer cancel()
		cleanupErr := l.svcCtx.AppUserModel.ReconcileAvatar(cleanupCtx, uint64(in.UserID), func(ctx context.Context, keep int64) error {
			_, err := l.svcCtx.FileRpc.RemoveFileReference(ctx, &filepb.RemoveFileReferenceRequest{Actor: actor, RefType: "app_avatar", RefID: in.UserID, KeepFileId: &keep})
			return err
		})
		if cleanupErr != nil {
			return nil, xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "头像引用同步未完成，请重试相同头像操作")
		}
	}
	if err != nil {
		if business, ok := xerr.AsCodeError(err); ok {
			return nil, business
		}
		if errors.Is(err, model.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.NOT_FOUND)
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}

func avatarFileError(err error) error {
	if code, msg, ok := xerr.FromGRPCError(err); ok {
		return xerr.NewErrCodeMsg(code, msg)
	}
	return xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
}
