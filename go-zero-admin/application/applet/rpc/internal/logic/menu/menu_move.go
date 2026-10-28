package menulogic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"sort"
	"strconv"
)

func menuMovePreview(tx *gorm.DB, id, parentID int64) (*pb.PreviewMenuMoveResponse, error) {
	var menus []model.SysBaseMenu
	if err := tx.Order("id").Find(&menus).Error; err != nil {
		return nil, err
	}
	parents := map[int64]int64{}
	for _, m := range menus {
		parents[m.ID] = m.ParentId
	}
	if _, ok := parents[id]; !ok {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单不存在")
	}
	if err := accessutil.ValidateParent(id, parentID, parents); err != nil {
		return nil, err
	}
	out := &pb.PreviewMenuMoveResponse{AuthorityIds: []int64{}, AncestorIds: []int64{}, AddedLinks: []*pb.PermissionMenuLink{}}
	for p := parentID; p != 0; p = parents[p] {
		out.AncestorIds = append(out.AncestorIds, p)
	}
	subtree := map[int64]bool{id: true}
	for changed := true; changed; {
		changed = false
		for _, m := range menus {
			if !subtree[m.ID] && subtree[m.ParentId] {
				subtree[m.ID] = true
				changed = true
			}
		}
	}
	var links []model.SysAuthorityMenu
	if err := tx.Order("sys_authority_authority_id,sys_base_menu_id").Find(&links).Error; err != nil {
		return nil, err
	}
	assigned := map[int64]map[int64]bool{}
	affected := map[int64]bool{}
	for _, link := range links {
		role, err := strconv.ParseInt(link.AuthorityId, 10, 64)
		if err != nil {
			return nil, err
		}
		menu, err := strconv.ParseInt(link.MenuId, 10, 64)
		if err != nil {
			return nil, err
		}
		if assigned[role] == nil {
			assigned[role] = map[int64]bool{}
		}
		assigned[role][menu] = true
		if subtree[menu] {
			affected[role] = true
		}
	}
	for role := range affected {
		out.AuthorityIds = append(out.AuthorityIds, role)
	}
	sort.Slice(out.AuthorityIds, func(i, j int) bool { return out.AuthorityIds[i] < out.AuthorityIds[j] })
	for _, role := range out.AuthorityIds {
		for _, ancestor := range out.AncestorIds {
			if !assigned[role][ancestor] {
				out.AddedLinks = append(out.AddedLinks, &pb.PermissionMenuLink{AuthorityId: role, MenuId: ancestor})
			}
		}
	}
	revision, err := accessutil.PermissionRevision(tx)
	if err != nil {
		return nil, err
	}
	children := []int64{}
	for child := range subtree {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool { return children[i] < children[j] })
	data, err := json.Marshal(struct {
		ID, ParentID int64
		Revision     string
		Subtree      []int64
		Impact       *pb.PreviewMenuMoveResponse
	}{id, parentID, revision, children, out})
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	out.Version = hex.EncodeToString(hash[:])
	return out, nil
}

func applyMenuMove(tx *gorm.DB, id, parentID int64, version string) error {
	preview, err := menuMovePreview(tx, id, parentID)
	if err != nil {
		return err
	}
	if version == "" || version != preview.Version {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单移动影响已变更或缺少预览，请重新预览并确认")
	}
	for _, link := range preview.AddedLinks {
		if err := tx.Create(&model.SysAuthorityMenu{AuthorityId: strconv.FormatInt(link.AuthorityId, 10), MenuId: strconv.FormatInt(link.MenuId, 10)}).Error; err != nil {
			return err
		}
	}
	return nil
}
