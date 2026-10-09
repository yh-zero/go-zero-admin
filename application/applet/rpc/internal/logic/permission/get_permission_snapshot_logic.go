package permissionlogic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/logic/datascope"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"
	"sort"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPermissionSnapshotLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPermissionSnapshotLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPermissionSnapshotLogic {
	return &GetPermissionSnapshotLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPermissionSnapshotLogic) GetPermissionSnapshot(in *pb.GetPermissionSnapshotRequest) (*pb.GetPermissionSnapshotResponse, error) {
	out := &pb.GetPermissionSnapshotResponse{Menus: []*pb.SysMenu{}, Codes: []string{}}
	err := accessutil.PermissionRead(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		if _, err := datascope.Apply(l.ctx, tx, in.Actor, "owner_id", "department_id"); err != nil {
			return err
		}
		edit, err := accessutil.LoadPermissionEdit(tx, in.Actor.AuthorityId, "menu")
		if err != nil {
			return err
		}
		out.AuthorityId = in.Actor.AuthorityId
		out.Revision = edit.Revision
		var role model.SysAuthority
		if err := tx.First(&role, "authority_id = ?", out.AuthorityId).Error; err != nil {
			return err
		}
		out.DefaultRouter = role.DefaultRouter
		assigned := map[int64]bool{}
		for _, id := range edit.MenuIds {
			assigned[id] = true
		}
		buttons := map[int64]bool{}
		for _, id := range edit.MenuBtnIds {
			buttons[id] = true
		}
		codes := map[string]bool{}
		var build func([]*pb.SysBaseMenu) []*pb.SysMenu
		build = func(menus []*pb.SysBaseMenu) []*pb.SysMenu {
			list := []*pb.SysMenu{}
			for _, m := range menus {
				if !assigned[m.ID] {
					continue
				}
				children := m.Children
				m.Children = nil
				n := &pb.SysMenu{SysBaseMenu: m, ID: m.ID, MenuId: strconv.FormatInt(m.ID, 10), AuthorityId: out.AuthorityId, Parameters: m.Parameters, Btns: map[string]int64{}}
				for _, b := range m.MenuBtn {
					if buttons[b.ID] {
						n.Btns[b.Name] = out.AuthorityId
						codes[m.Name+":"+b.Name] = true
						if b.PermissionKey != "" {
							codes[b.PermissionKey] = true
						}
					}
				}
				n.Children = build(children)
				list = append(list, n)
			}
			return list
		}
		out.Menus = build(edit.Menus)
		for code := range codes {
			out.Codes = append(out.Codes, code)
		}
		sort.Strings(out.Codes)
		data, err := json.Marshal(struct {
			AuthorityId   int64
			DefaultRouter string
			Menus         []*pb.SysMenu
			Codes         []string
		}{out.AuthorityId, out.DefaultRouter, out.Menus, out.Codes})
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		out.Fingerprint = hex.EncodeToString(sum[:])
		return nil
	})
	return out, err
}
