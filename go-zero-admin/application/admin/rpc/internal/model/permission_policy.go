package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// rebuildRoleMenuPolicies 按当前角色绑定和显式映射重建 API 策略，同时清理历史不一致规则。
func rebuildRoleMenuPolicies(ctx context.Context, s sqlx.Session, roleId uint64) error {
	var role struct {
		Code string `db:"code"`
	}
	if err := s.QueryRowCtx(ctx, &role, "SELECT code FROM sys_role WHERE id=? AND deleted_at IS NULL", roleId); err != nil {
		return err
	}
	if role.Code == "super_admin" {
		return nil
	}
	if _, err := s.ExecCtx(ctx, "DELETE FROM casbin_rule WHERE ptype='p' AND v0=?", role.Code); err != nil {
		return err
	}
	var rows []struct {
		Domain string `db:"domain"`
		Method string `db:"method"`
		Path   string `db:"path"`
	}
	if err := s.QueryRowsCtx(ctx, &rows, `SELECT DISTINCT g.v2 AS domain,a.method,a.path
FROM casbin_rule g
JOIN sys_role_menu rm ON rm.role_id=?
JOIN sys_permission sp ON sp.menu_id=rm.menu_id
JOIN sys_api a ON a.id=sp.api_id
JOIN sys_menu m ON m.id=rm.menu_id AND m.deleted_at IS NULL AND m.status=1
WHERE g.ptype='g' AND g.v1=? ORDER BY domain,method,path`, roleId, role.Code); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := s.ExecCtx(ctx, "INSERT INTO casbin_rule (ptype,v0,v1,v2,v3,v4,v5) VALUES ('p',?,?,?,?, '', '')", role.Code, row.Domain, row.Path, row.Method); err != nil {
			return fmt.Errorf("sync role menu policy: %w", err)
		}
	}
	return nil
}

func replaceMenuPermissions(ctx context.Context, s sqlx.Session, menuID int64, apiIDs []int64) error {
	if _, err := s.ExecCtx(ctx, "DELETE FROM sys_permission WHERE menu_id=?", menuID); err != nil {
		return err
	}
	seen := make(map[int64]bool, len(apiIDs))
	for _, apiID := range apiIDs {
		if apiID <= 0 {
			return ErrInvalidMutation
		}
		if seen[apiID] {
			continue
		}
		seen[apiID] = true
		result, err := s.ExecCtx(ctx, "INSERT INTO sys_permission (menu_id,api_id) SELECT ?,id FROM sys_api WHERE id=?", menuID, apiID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return fmt.Errorf("%w: API 不存在", ErrInvalidMutation)
		}
	}
	return nil
}

func roleCodesFromBindings(bindings map[string]bool) []string {
	codes := make([]string, 0, len(bindings))
	for code := range bindings {
		codes = append(codes, code)
	}
	return codes
}

func rebuildRolePoliciesByCodes(ctx context.Context, s sqlx.Session, codes []string) error {
	for _, code := range codes {
		if code == "super_admin" {
			continue
		}
		var id uint64
		if err := s.QueryRowCtx(ctx, &id, "SELECT id FROM sys_role WHERE code=? AND deleted_at IS NULL", code); err != nil {
			return err
		}
		if err := rebuildRoleMenuPolicies(ctx, s, id); err != nil {
			return err
		}
	}
	return nil
}

func rebuildAllRoleMenuPolicies(ctx context.Context, s sqlx.Session) error {
	var ids []uint64
	if err := s.QueryRowsCtx(ctx, &ids, "SELECT id FROM sys_role WHERE deleted_at IS NULL ORDER BY id"); err != nil {
		return err
	}
	for _, id := range ids {
		if err := rebuildRoleMenuPolicies(ctx, s, id); err != nil {
			return err
		}
	}
	return nil
}
