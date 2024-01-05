package authority

import (
	"context"
	"fmt"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetAuthorityListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAuthorityListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthorityListLogic {
	return &GetAuthorityListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAuthorityListLogic) GetAuthorityList(req *types.GetAuthorityListRequest) (resp *types.GetAuthorityListResponse, err error) {
	var page pb.PageRequest
	if req.PageNo == 0 {
		req.PageNo = l.svcCtx.Config.Page.PageNo
	}
	if req.PageSize == 0 {
		req.PageSize = l.svcCtx.Config.Page.PageSize
	}
	page.PageNo = int64(req.PageNo)
	page.PageSize = int64(req.PageSize)

	authorityListGet, err := l.svcCtx.AppletAuthorityRPC.GetAuthorityList(l.ctx, &pb.GetAuthorityListRequest{Page: &page})
	if err != nil {
		logx.Errorf("AuthorityListGet 错误 error: %v", err)
		return nil, err
	}
	fmt.Println("--------- authorityListGet", authorityListGet)

	var typesGetAuthorityListRes types.GetAuthorityListResponse

	typesGetAuthorityListRes.Total = authorityListGet.Total
	typesGetAuthorityListRes.PageNo = req.PageNo
	typesGetAuthorityListRes.PageSize = req.PageSize
	_ = copier.Copy(&typesGetAuthorityListRes.List, authorityListGet.SysAuthority)

	return &typesGetAuthorityListRes, err
}
