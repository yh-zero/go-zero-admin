package fileresourceservicelogic

import (
	"context"
	"strings"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/datascope"
)

type GetFileListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFileListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileListLogic {
	return &GetFileListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetFileListLogic) GetFileList(in *pb.FileListRequest) (*pb.FileListResponse, error) {
	if in == nil || in.PageRequest == nil {
		return nil, fileError("分页参数不能为空")
	}
	offset, limit, err := accessutil.Page(in.PageRequest.PageNo, in.PageRequest.PageSize)
	if err != nil {
		return nil, err
	}
	if in.Status != "" && in.Status != "active" && in.Status != "deleting" && in.Status != "deleted" {
		return nil, fileError("文件状态无效")
	}
	db, err := datascope.Apply(l.ctx, l.svcCtx.DB.Model(&model.SysFileResource{}), in.Actor, "owner_id", "department_id")
	if err != nil {
		return nil, err
	}
	if in.Status == "" {
		db = db.Where("status <> ?", "deleted")
	} else {
		db = db.Where("status = ?", in.Status)
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		if len([]rune(name)) > 255 {
			return nil, fileError("文件名称过滤条件过长")
		}
		db = db.Where("name LIKE ? ESCAPE '!'", "%"+strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(name)+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var files []model.SysFileResource
	if err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&files).Error; err != nil {
		return nil, err
	}
	response := &pb.FileListResponse{List: make([]*pb.FileResource, 0, len(files)), Total: total}
	if len(files) == 0 {
		return response, nil
	}
	ids := make([]int64, 0, len(files))
	for _, file := range files {
		ids = append(ids, file.ID)
	}
	var counts []struct {
		FileID int64
		Total  int64
	}
	if err := l.svcCtx.DB.WithContext(l.ctx).Model(&model.SysFileReference{}).Select("file_id, COUNT(*) AS total").Where("file_id IN ?", ids).Group("file_id").Scan(&counts).Error; err != nil {
		return nil, err
	}
	references := map[int64]int64{}
	for _, count := range counts {
		references[count.FileID] = count.Total
	}
	for _, file := range files {
		response.List = append(response.List, fileProtoWithReferences(&file, references[file.ID]))
	}
	return response, nil
}
