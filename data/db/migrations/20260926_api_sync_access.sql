-- Idempotent resources for API synchronization. Built-in administrator role = 1.
-- Keep existing resource IDs/descriptions and all existing role grants.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
START TRANSACTION;

INSERT INTO sys_apis (created_at, updated_at, path, method, api_group, description)
SELECT NOW(3), NOW(3), '/v1/sys/api/previewSync', 'GET', 'api', '预览Swagger与API资源差异，失效资源仅提示'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/api/previewSync' AND method='GET' AND deleted_at IS NULL);
INSERT INTO sys_apis (created_at, updated_at, path, method, api_group, description)
SELECT NOW(3), NOW(3), '/v1/sys/api/applySync', 'POST', 'api', '按预览版本同步选中API，仅新增资源或更新分组描述，保留授权'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/api/applySync' AND method='POST' AND deleted_at IS NULL);

INSERT INTO casbin_rule (ptype, v0, v1, v2)
SELECT 'p', '1', a.path, a.method FROM sys_apis a
WHERE a.deleted_at IS NULL
  AND ((a.path='/v1/sys/api/previewSync' AND a.method='GET') OR (a.path='/v1/sys/api/applySync' AND a.method='POST'))
  AND EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
  -- Legacy resource tables use general_ci while Casbin may inherit MySQL 8's
  -- 0900_ai_ci default. Compare explicitly without altering stored collations.
  AND NOT EXISTS (SELECT 1 FROM casbin_rule c WHERE c.ptype='p' AND c.v0='1'
    AND c.v1 COLLATE utf8mb4_general_ci = a.path COLLATE utf8mb4_general_ci
    AND c.v2 COLLATE utf8mb4_general_ci = a.method COLLATE utf8mb4_general_ci);

INSERT INTO sys_base_menu_btns (created_at, updated_at, name, `desc`, sys_base_menu_id)
SELECT NOW(3), NOW(3), 'sync', '同步接口资源', m.id FROM sys_base_menus m
WHERE m.name='api' AND m.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM sys_base_menu_btns b WHERE b.sys_base_menu_id=m.id AND b.name='sync' AND b.deleted_at IS NULL);

INSERT INTO sys_authority_btns (authority_id, sys_menu_id, sys_base_menu_btn_id)
SELECT 1, m.id, b.id FROM sys_base_menus m JOIN sys_base_menu_btns b ON b.sys_base_menu_id=m.id AND b.name='sync' AND b.deleted_at IS NULL
WHERE m.name='api' AND m.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
  AND NOT EXISTS (SELECT 1 FROM sys_authority_btns ab WHERE ab.authority_id=1 AND ab.sys_menu_id=m.id AND ab.sys_base_menu_btn_id=b.id);
COMMIT;
