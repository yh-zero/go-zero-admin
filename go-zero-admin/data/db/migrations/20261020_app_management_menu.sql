-- Group App modules without changing their URLs or operation permissions.
-- Run with API/RPC writers stopped, like other permission migrations.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
START TRANSACTION;
SELECT revision FROM sys_permission_versions WHERE id=1 FOR UPDATE;

INSERT INTO sys_base_menus(created_at,updated_at,menu_level,parent_id,path,name,hidden,component,sort,active_name,keep_alive,default_menu,title,icon,close_tab)
SELECT NOW(3),NOW(3),0,0,'app','app-management',0,'views/superAdmin/index.vue',20,'',0,0,'App 管理','lucide:smartphone',0
WHERE NOT EXISTS(SELECT 1 FROM sys_base_menus WHERE name='app-management' AND deleted_at IS NULL);
SET @app_management_menu_id=(SELECT id FROM sys_base_menus WHERE name='app-management' AND deleted_at IS NULL LIMIT 1);

UPDATE sys_base_menus SET parent_id=@app_management_menu_id,menu_level=1,
 path=CASE name WHEN 'app-members-users' THEN '/app-members/users' ELSE '/app-members/memberships' END,
 sort=CASE name WHEN 'app-members-users' THEN 10 ELSE 20 END
WHERE name IN('app-members-users','app-memberships') AND deleted_at IS NULL;

-- A directory grant only makes previously granted children reachable. No new
-- leaf, button or API grants are created for any role.
INSERT INTO sys_authority_menus(sys_base_menu_id,sys_authority_authority_id)
SELECT DISTINCT @app_management_menu_id,a.sys_authority_authority_id
FROM sys_authority_menus a
JOIN sys_base_menus m ON m.id=a.sys_base_menu_id AND m.deleted_at IS NULL
JOIN sys_authorities r ON r.authority_id=a.sys_authority_authority_id AND r.deleted_at IS NULL
WHERE m.name IN('app-members-users','app-memberships') AND m.parent_id=@app_management_menu_id
AND NOT EXISTS(SELECT 1 FROM sys_authority_menus p WHERE p.sys_base_menu_id=@app_management_menu_id AND p.sys_authority_authority_id=a.sys_authority_authority_id);

UPDATE sys_permission_versions SET revision=revision+1 WHERE id=1;
COMMIT;
SET @app_management_menu_id=NULL;
