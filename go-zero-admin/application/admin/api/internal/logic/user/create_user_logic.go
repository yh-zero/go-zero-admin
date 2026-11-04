package user

import (
	"context"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserLogic {
	return &CreateUserLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateUserLogic) CreateUser(req *types.CreateUserReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	_, err := l.svcCtx.UserRpc.CreateUser(l.ctx, &adminpb.CreateUserRequest{
		Actor: middleware.ActorFromContext(l.ctx), RoleIds: req.RoleIds,
		Username: req.Username, Password: req.Password, Nickname: req.Nickname,
		RealName: req.RealName, Email: req.Email, Phone: req.Phone,
		DeptId: req.DeptId, AvatarFileId: req.AvatarFileId,
		Gender: req.Gender, Status: req.Status, Remark: req.Remark,
	})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
