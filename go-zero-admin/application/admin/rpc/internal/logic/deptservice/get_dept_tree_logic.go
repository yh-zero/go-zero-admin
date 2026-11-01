package deptservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetDeptTreeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDeptTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeptTreeLogic {
	return &GetDeptTreeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetDeptTreeLogic) GetDeptTree(in *adminpb.GetDeptTreeRequest) (*adminpb.GetDeptTreeResponse, error) {
	rows, err := l.svcCtx.SysDeptModel.ListDeptTree(l.ctx, int64(in.Status))
	if err != nil {
		l.Errorf("dept tree error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	var counts map[int64]int64
	membersVisible := false
	if in.IncludeMembers {
		counts, err = l.svcCtx.SysDeptModel.ListDepartmentMemberCounts(l.ctx, model.UserActor{
			UserId: in.GetActor().GetUserId(), DeptId: in.GetActor().GetDeptId(),
		})
		if err != nil && !errors.Is(err, model.ErrDataScopeDenied) {
			l.Errorf("dept member count error: %v", err)
			return nil, xerr.NewErrCode(xerr.DB_ERROR)
		}
		membersVisible = err == nil
	}
	list := make([]*adminpb.DeptInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &adminpb.DeptInfo{
			Id: r.Id, ParentId: r.ParentId, Name: r.Name, Leader: r.Leader,
			Phone: r.Phone, Email: r.Email, Sort: r.Sort, Status: r.Status,
			Path: r.Path, CreatedAt: r.CreatedAt,
			LeaderUserId: r.LeaderUserId, MemberCount: counts[r.Id], MembersVisible: membersVisible,
		})
	}
	return &adminpb.GetDeptTreeResponse{List: list}, nil
}
