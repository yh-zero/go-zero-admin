// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package user

import (
	"context"
	"fmt"
	"sort"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ImportUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewImportUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImportUsersLogic {
	return &ImportUsersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ImportUsersLogic) ImportUsers(req *types.ImportUsersReq) (*types.ImportUsersResp, error) {
	actor := middleware.ActorFromContext(l.ctx)
	if actor.GetUserId() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	allowed, err := l.svcCtx.Enforcer.Enforce(fmt.Sprintf("user:%d", actor.GetUserId()), fmt.Sprintf("dept:%d", actor.GetDeptId()), "/v1/sys/user/create", "POST")
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if !allowed {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil || req.Confirm != (len(req.RowNumbers) > 0) {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	entries, err := parseUserImportCSV(req.Csv)
	if err != nil {
		return nil, err
	}
	response := &types.ImportUsersResp{Rows: make([]*types.ImportUserRow, 0, len(entries))}
	input := &adminpb.ImportUsersRequest{Actor: actor, Confirm: req.Confirm}
	selected := make(map[int64]bool, len(req.RowNumbers))
	for _, row := range req.RowNumbers {
		if _, exists := selected[row]; exists {
			return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
		}
		selected[row] = false
	}
	for _, entry := range entries {
		_, chosen := selected[entry.row]
		if chosen {
			selected[entry.row] = true
		}
		if entry.message == "" {
			input.Rows = append(input.Rows, &adminpb.ImportUserInput{Row: entry.row, Username: entry.username, DeptId: entry.deptId, RealName: entry.realName})
			if chosen {
				input.RowNumbers = append(input.RowNumbers, entry.row)
			}
		} else if !req.Confirm || chosen {
			response.Rows = append(response.Rows, &types.ImportUserRow{Row: entry.row, Username: entry.username, DeptId: entry.deptId, Status: "invalid", Message: entry.message})
		}
	}
	for _, found := range selected {
		if !found {
			return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
		}
	}
	if len(input.Rows) > 0 && (!req.Confirm || len(input.RowNumbers) > 0) {
		result, err := l.svcCtx.UserRpc.ImportUsers(l.ctx, input)
		if err != nil {
			return nil, err
		}
		response.Created = result.Created
		for _, row := range result.Rows {
			response.Rows = append(response.Rows, &types.ImportUserRow{Row: row.Row, Username: row.Username, DeptId: row.DeptId, Status: row.Status, Message: row.Message})
		}
	}

	sort.Slice(response.Rows, func(i, j int) bool { return response.Rows[i].Row < response.Rows[j].Row })
	return response, nil
}
