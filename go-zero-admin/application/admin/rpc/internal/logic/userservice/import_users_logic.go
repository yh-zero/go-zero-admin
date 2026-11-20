package userservicelogic

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxUserImportRows = 100

type ImportUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewImportUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImportUsersLogic {
	return &ImportUsersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ImportUsersLogic) ImportUsers(in *adminpb.ImportUsersRequest) (*adminpb.ImportUsersResponse, error) {
	if in.GetActor().GetUserId() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if len(in.Rows) == 0 || len(in.Rows) > maxUserImportRows || len(in.RowNumbers) > len(in.Rows) || in.Confirm != (len(in.RowNumbers) > 0) {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	rows := make([]model.UserImportCandidate, 0, len(in.Rows))
	rowNumbers := make(map[int64]bool, len(in.Rows))
	for _, row := range in.Rows {
		if row == nil || row.Row < 2 || row.DeptId < 0 || rowNumbers[row.Row] || !validImportUsername(row.Username) ||
			!utf8.ValidString(row.RealName) || utf8.RuneCountInString(row.RealName) > 64 {
			return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
		}
		rowNumbers[row.Row] = true
		rows = append(rows, model.UserImportCandidate{Row: row.Row, Username: row.Username, DeptId: row.DeptId, RealName: row.RealName})
	}
	selection := make(map[int64]bool, len(in.RowNumbers))
	for _, number := range in.RowNumbers {
		if !rowNumbers[number] || selection[number] {
			return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
		}
		selection[number] = true
	}
	actor := model.UserActor{UserId: in.Actor.UserId, DeptId: in.Actor.DeptId}
	issues, err := l.svcCtx.SysUserModel.ImportUsers(l.ctx, actor, rows, in.RowNumbers)
	if !model.MutationCommitted(err) {
		return nil, userMutationError(err)
	}
	cacheSyncFailed := err != nil
	if cacheSyncFailed {
		l.Error("批量用户导入已提交，缓存同步失败")
	}
	response := &adminpb.ImportUsersResponse{}
	for _, row := range rows {
		if in.Confirm && !selection[row.Row] {
			continue
		}
		result := &adminpb.ImportUserResult{Row: row.Row, Username: row.Username, DeptId: row.DeptId, Status: "ready"}
		switch issue := issues[row.Row]; {
		case errors.Is(issue, model.ErrDataScopeDenied):
			result.Status, result.Message = "invalid", "部门不在可管理范围内"
		case errors.Is(issue, model.ErrImportConflict):
			result.Status, result.Message = "failed", issue.Error()
		case issue != nil:
			result.Status, result.Message = "invalid", issue.Error()
		case in.Confirm:
			result.Status, result.Message = "created", "已创建停用账号，请先启用，再重置密码"
			if cacheSyncFailed {
				result.Message += "。缓存同步失败，列表可能暂未更新"
			}
			response.Created++
		}
		response.Rows = append(response.Rows, result)
	}
	return response, nil
}

func validImportUsername(name string) bool {
	length := utf8.RuneCountInString(name)
	return length >= 2 && length <= 64 && utf8.ValidString(name) && strings.TrimSpace(name) == name && !strings.ContainsFunc(name, unicode.IsControl)
}
