package userservicelogic

import (
	"context"
	"errors"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/xerr"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
)

type UpdateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateUserLogic) UpdateUser(in *adminpb.UpdateUserRequest) (*adminpb.NoDataResponse, error) {
	err := UpdateUserWithAvatar(l.ctx, l.svcCtx, in.UserId, in.GetActor(), model.UserPatch{
		Nickname: in.Nickname, RealName: in.RealName, Email: in.Email, Phone: in.Phone, DeptId: in.DeptId, AvatarFileId: in.AvatarFileId,
		Gender: in.Gender, Status: in.Status, Remark: in.Remark, UpdateRoles: in.UpdateRoles, RoleIds: in.RoleIds,
	})
	if err != nil {
		return nil, userMutationError(err)
	}
	return &adminpb.NoDataResponse{}, nil
}

func fileActor(actor *adminpb.Actor) *filepb.Actor {
	return &filepb.Actor{UserID: actor.GetUserId(), Username: actor.GetUsername(), DeptId: actor.GetDeptId(), IP: actor.GetIp(), UserAgent: actor.GetUserAgent()}
}

func prepareAvatar(s *svc.ServiceContext, actor *adminpb.Actor, attempted *bool) model.AvatarPrepare {
	return func(ctx context.Context, userId, oldFileId, newFileId int64) error {
		if newFileId <= 0 || newFileId == oldFileId {
			return nil
		}
		*attempted = true
		_, err := s.FileRpc.AddFileReference(ctx, &filepb.AddFileReferenceRequest{Actor: fileActor(actor), FileID: newFileId, RefType: "admin_avatar", RefID: userId})
		if err != nil {
			return fileBusinessError(err)
		}
		return nil
	}
}

func fileBusinessError(err error) error {
	if code, msg, ok := xerr.FromGRPCError(err); ok {
		return xerr.NewErrCodeMsg(code, msg)
	}
	return xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "文件服务暂不可用，请重试")
}

func ReconcileUserAvatar(ctx context.Context, s *svc.ServiceContext, userId int64, actor *adminpb.Actor) error {
	if userId <= 0 || actor.GetUserId() <= 0 {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	// 补偿不复用已取消的 HTTP/RPC deadline，但独立截止时间限制持锁时长。
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
	defer cancel()
	err := s.SysUserModel.ReconcileAvatar(cleanupCtx, userId, func(ctx context.Context, keep int64) error {
		_, err := s.FileRpc.RemoveFileReference(ctx, &filepb.RemoveFileReferenceRequest{Actor: fileActor(actor), RefType: "admin_avatar", RefID: userId, KeepFileId: &keep})
		return err
	})
	if err != nil {
		return xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "头像引用同步未完成，请重试相同头像操作")
	}
	return nil
}

func UpdateUserWithAvatar(ctx context.Context, s *svc.ServiceContext, userId int64, actor *adminpb.Actor, p model.UserPatch) error {
	attempted := false
	p.PrepareAvatar = prepareAvatar(s, actor, &attempted)
	p.PrepareStatus = func(ctx context.Context, targetId int64) error {
		if err := s.Sessions.RevokeAll(ctx, targetId); err != nil {
			return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		return nil
	}
	err := s.SysUserModel.UpdatePartial(ctx, userId, model.UserActor{UserId: actor.GetUserId(), DeptId: actor.GetDeptId(), OwnAccount: p.OwnAccount}, p)
	if !p.OwnAccount && model.MutationCommitted(err) {
		s.PublishPolicyChanged(ctx)
	}
	if p.AvatarFileId != nil && (err == nil || attempted) {
		if cleanupErr := ReconcileUserAvatar(ctx, s, userId, actor); cleanupErr != nil {
			return errors.Join(cleanupErr, err)
		}
	}
	return err
}

// URL 仅按真实调用者解析；私有文件无权限或文件服务暂不可用时不泄露地址，也不影响认证读取用户状态。
func ResolveAvatarURL(ctx context.Context, s *svc.ServiceContext, fileId int64, actor *adminpb.Actor) string {
	if fileId <= 0 || actor.GetUserId() <= 0 {
		return ""
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	result, err := s.FileRpc.GetFileURL(lookupCtx, &filepb.GetFileURLRequest{Actor: fileActor(actor), FileID: fileId})
	if err != nil {
		return ""
	}
	return result.URL
}
