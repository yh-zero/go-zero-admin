package apilogic

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAllApiListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAllApiListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAllApiListLogic {
	return &GetAllApiListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取全部API列表
func (l *GetAllApiListLogic) GetAllApiList(in *pb.NoDataResponse) (*pb.GetAllApiListResponse, error) {
	fmt.Println("========= GetAllApiList ======= ")
	var apiList []model.SysApi
	err := l.svcCtx.DB.Find(&apiList).Error
	//var pbSysApi []pb.SysApi
	var AllApiListRes pb.GetAllApiListResponse
	_ = copier.Copy(&AllApiListRes.ApiList, apiList)

	return &AllApiListRes, err
}
