-- Explicit operator step, after 20261008_permission_revision_history.sql.
-- Stop API/RPC instances first; review backup and role 1's legacy recovery
-- grants. This script only repairs access to the replacement editor and move
-- preview. History, rollback and ordinary roles require explicit API grants.
START TRANSACTION;
SELECT authority_id FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL FOR UPDATE;
SELECT revision FROM sys_permission_versions WHERE id=1 FOR UPDATE;

INSERT INTO sys_apis(created_at,updated_at,path,description,api_group,method)
SELECT NOW(3),NOW(3),'/v1/sys/permissions/edit','读取四类授权的统一资源与选择快照','permission','GET'
WHERE NOT EXISTS(SELECT 1 FROM sys_apis WHERE path='/v1/sys/permissions/edit' AND method='GET' AND deleted_at IS NULL);
INSERT INTO sys_apis(created_at,updated_at,path,description,api_group,method)
SELECT NOW(3),NOW(3),'/v1/sys/menu/previewMove','预览菜单移动新增可见入口','menu','POST'
WHERE NOT EXISTS(SELECT 1 FROM sys_apis WHERE path='/v1/sys/menu/previewMove' AND method='POST' AND deleted_at IS NULL);

INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5)
SELECT 'p','1',replacement.path,replacement.method,'','',''
FROM (SELECT '/v1/sys/permissions/edit' AS path,'GET' AS method UNION ALL SELECT '/v1/sys/menu/previewMove','POST') replacement
WHERE EXISTS(SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND EXISTS(SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1='/v1/sys/casbin/getPathByAuthorityId' AND v2='GET')
AND EXISTS(SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1='/v1/sys/casbin/updateCasbinData' AND v2='PUT')
AND NOT EXISTS(SELECT 1 FROM casbin_rule p WHERE p.ptype='p' AND p.v0='1' AND p.v1=replacement.path AND p.v2=replacement.method);

INSERT INTO sys_policy_versions(id,version,updated_at) VALUES(1,1,NOW(3))
ON DUPLICATE KEY UPDATE version=version+1,updated_at=NOW(3);
UPDATE sys_permission_versions SET revision=revision+1 WHERE id=1;
COMMIT;
-- Verify both explicit role-1 grants before restarting the upgraded services.
SELECT ptype,v0,v1,v2 FROM casbin_rule WHERE ptype='p' AND v0='1'
AND v1 IN('/v1/sys/permissions/edit','/v1/sys/menu/previewMove');
