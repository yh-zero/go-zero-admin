package audit

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

const auditExportLimit = 10000

type ExportAuditLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExportAuditLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExportAuditLogsLogic {
	return &ExportAuditLogsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ExportAuditLogsLogic) ExportAuditLogs(req *types.GetAuditLogListReq) ([]*types.AuditLogInfo, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	rows := make([]*types.AuditLogInfo, 0)
	for page := int64(1); ; page++ {
		pageReq := *req
		pageReq.Page = page
		pageReq.PageSize = 100
		resp, err := NewGetAuditLogListLogic(l.ctx, l.svcCtx).GetAuditLogList(&pageReq)
		if err != nil {
			return nil, err
		}
		if page == 1 && resp.Total > auditExportLimit {
			return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "导出超过 10000 行，请缩小筛选范围")
		}
		rows = append(rows, resp.List...)
		if len(rows) >= int(resp.Total) || len(resp.List) == 0 {
			return rows, nil
		}
	}
}
