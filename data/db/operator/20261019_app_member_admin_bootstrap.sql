-- Optional, explicit grant of the App management module to administrator role 1.
-- Run after 20261019_app_members.sql with API/RPC stopped. Other roles are unchanged.
START TRANSACTION;
SELECT authority_id FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL FOR UPDATE;
SELECT revision FROM sys_permission_versions WHERE id=1 FOR UPDATE;

INSERT INTO sys_authority_menus(sys_base_menu_id,sys_authority_authority_id)
SELECT m.id,1 FROM sys_base_menus m
WHERE m.deleted_at IS NULL AND m.name IN('app-members-users','app-memberships')
AND m.component IN('views/business/app-members/users.vue','views/business/app-members/memberships.vue')
AND EXISTS(SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS(SELECT 1 FROM sys_authority_menus a WHERE a.sys_base_menu_id=m.id AND a.sys_authority_authority_id=1);

INSERT INTO sys_authority_btns(authority_id,sys_menu_id,sys_base_menu_btn_id)
SELECT 1,b.sys_base_menu_id,b.id FROM sys_base_menu_btns b
JOIN sys_authority_menus a ON a.sys_base_menu_id=b.sys_base_menu_id AND a.sys_authority_authority_id=1
WHERE b.deleted_at IS NULL AND b.permission_key IN('app-members-users:review','app-members-users:status','app-members-users:revoke-session','app-memberships:grant','app-memberships:revoke')
AND NOT EXISTS(SELECT 1 FROM sys_authority_btns existing WHERE existing.authority_id=1 AND existing.sys_base_menu_btn_id=b.id);

INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5)
SELECT 'p','1',a.path,a.method,'','','' FROM sys_apis a
WHERE a.deleted_at IS NULL AND a.api_group='appmembers'
AND (a.path,a.method) IN (
 ('/v1/sys/app-members/users','GET'),
 ('/v1/sys/app-members/status','PUT'),
 ('/v1/sys/app-members/sessions','GET'),
 ('/v1/sys/app-members/session','DELETE'),
 ('/v1/sys/app-members/plans','GET'),
 ('/v1/sys/app-members/grants','GET'),
 ('/v1/sys/app-members/grants','POST'),
 ('/v1/sys/app-members/grant','DELETE')
)
AND EXISTS(SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
-- Historical Casbin and API tables use different utf8mb4 collations.
AND NOT EXISTS(SELECT 1 FROM casbin_rule p WHERE p.ptype='p' AND p.v0='1'
 AND p.v1 COLLATE utf8mb4_bin=a.path COLLATE utf8mb4_bin
 AND p.v2 COLLATE utf8mb4_bin=a.method COLLATE utf8mb4_bin);

INSERT INTO sys_policy_versions(id,version,updated_at) VALUES(1,1,NOW(3))
ON DUPLICATE KEY UPDATE version=version+1,updated_at=NOW(3);
UPDATE sys_permission_versions SET revision=revision+1 WHERE id=1;
COMMIT;

