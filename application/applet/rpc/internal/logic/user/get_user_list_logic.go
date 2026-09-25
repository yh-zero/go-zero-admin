package userlogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"strings"
)

type GetUserListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserListLogic {
	return &GetUserListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *GetUserListLogic) GetUserList(in *pb.GetUserListRequest) (*pb.GetUserListResponse, error) {
	page := in.GetPageRequest()
	if page == nil || page.PageNo < 1 || page.PageSize < 1 || page.PageSize > 500 {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "分页参数无效，pageSize 应为 1 至 500")
	}
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&model.SysUser{})
	if keyword := strings.TrimSpace(page.Keyword); keyword != "" {
		db = db.Where("username LIKE ? OR nick_name LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var users []model.SysUser
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := db.Order("id DESC").Limit(int(page.PageSize)).Offset(int((page.PageNo - 1) * page.PageSize)).Preload("Authorities").Preload("Authority").Find(&users).Error; err != nil {
		return nil, err
	}
	output := &pb.GetUserListResponse{Total: total, UserInfoList: []*pb.UserInfo{}}
	if err := copier.Copy(&output.UserInfoList, users); err != nil {
		return nil, err
	}
	for _, user := range output.UserInfoList {
		user.Password = ""
	}
	return output, nil
}
