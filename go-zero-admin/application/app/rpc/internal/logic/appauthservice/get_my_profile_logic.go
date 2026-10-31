package appauthservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/xerr"
)

type GetMyProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyProfileLogic(ctx context.Context, s *svc.ServiceContext) *GetMyProfileLogic {
	return &GetMyProfileLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *GetMyProfileLogic) GetMyProfile(in *apppb.AppProfileRequest) (*apppb.AppProfileResponse, error) {
	u, _, err := CurrentAppUser(l.ctx, l.svcCtx, in.AccessToken)
	if err != nil {
		return nil, err
	}
	info, err := profileInfo(l.ctx, l.svcCtx, u)
	if err != nil {
		return nil, err
	}
	return &apppb.AppProfileResponse{User: info}, nil
}
func profileInfo(ctx context.Context, s *svc.ServiceContext, u *model.AppUser) (*apppb.AppUserInfo, error) {
	info := mapAppUser(u)
	if info.AvatarFileId > 0 && s.FileRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	if info.AvatarFileId > 0 && s.FileRpc != nil {
		// App 身份不映射到后台账户，只允许解析公开头像。
		resp, err := s.FileRpc.GetFileURL(ctx, &filepb.GetFileURLRequest{PublicOnly: true, FileID: info.AvatarFileId})
		if err != nil {
			return nil, err
		}
		info.AvatarURL = resp.URL
	}
	return info, nil
}
