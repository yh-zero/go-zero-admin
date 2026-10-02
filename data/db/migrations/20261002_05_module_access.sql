-- New administrative resources. Personal session endpoints use Session middleware.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
START TRANSACTION;
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/departments','GET','organization','查询部门树'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/departments' AND method='GET' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/departments','GET'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/departments' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='GET' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/departments','POST','organization','创建部门'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/departments' AND method='POST' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/departments','POST'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/departments' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='POST' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/departments','PUT','organization','更新部门'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/departments' AND method='PUT' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/departments','PUT'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/departments' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='PUT' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/departments','DELETE','organization','删除部门'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/departments' AND method='DELETE' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/departments','DELETE'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/departments' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='DELETE' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/positions','GET','organization','查询岗位'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/positions' AND method='GET' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/positions','GET'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/positions' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='GET' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/positions','POST','organization','创建岗位'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/positions' AND method='POST' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/positions','POST'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/positions' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='POST' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/positions','PUT','organization','更新岗位'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/positions' AND method='PUT' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/positions','PUT'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/positions' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='PUT' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/positions','DELETE','organization','删除岗位'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/positions' AND method='DELETE' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/positions','DELETE'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/positions' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='DELETE' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/membership','GET','organization','查询用户部门岗位'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/membership' AND method='GET' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/membership','GET'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/membership' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='GET' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/membership','PUT','organization','更新用户部门岗位'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/membership' AND method='PUT' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/membership','PUT'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/membership' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='PUT' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/dataScope','GET','organization','查询角色数据范围'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/dataScope' AND method='GET' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/dataScope','GET'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/dataScope' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='GET' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/organization/dataScope','PUT','organization','更新角色数据范围'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/organization/dataScope' AND method='PUT' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/organization/dataScope','PUT'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/organization/dataScope' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='PUT' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/files/list','GET','files','查询文件资源'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/files/list' AND method='GET' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/files/list','GET'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/files/list' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='GET' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/files/url','GET','files','获取受控文件地址'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/files/url' AND method='GET' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/files/url','GET'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/files/url' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='GET' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/files/resource','DELETE','files','删除无引用文件资源'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/files/resource' AND method='DELETE' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/files/resource','DELETE'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/files/resource' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='DELETE' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/files/reference','POST','files','登记文件引用'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/files/reference' AND method='POST' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/files/reference','POST'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/files/reference' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='POST' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/files/reference','DELETE','files','移除文件引用'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/files/reference' AND method='DELETE' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/files/reference','DELETE'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/files/reference' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='DELETE' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/session/admin/devices','GET','session','查询用户设备会话'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/session/admin/devices' AND method='GET' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/session/admin/devices','GET'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/session/admin/devices' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='GET' COLLATE utf8mb4_general_ci);
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),'/v1/sys/session/admin/device','DELETE','session','撤销用户设备会话'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/session/admin/device' AND method='DELETE' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1','/v1/sys/session/admin/device','DELETE'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
AND NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 COLLATE utf8mb4_general_ci='/v1/sys/session/admin/device' COLLATE utf8mb4_general_ci AND v2 COLLATE utf8mb4_general_ci='DELETE' COLLATE utf8mb4_general_ci);
UPDATE sys_policy_versions SET version=version+1,updated_at=NOW(3) WHERE id=1;
COMMIT;

