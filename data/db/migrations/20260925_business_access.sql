-- 2026-09-25：前后端接口接入的增量初始化（MySQL 8）。
-- 在项目已有数据库中执行，必须先备份 sys_apis / casbin_rule /
-- sys_base_menus / sys_authority_menus / sys_base_menu_btns / sys_authority_btns。
-- 可重复执行：按名称和 method+path 查找，不覆盖已有资源/授权，不依赖主键。
-- 只补 admin 账号当前角色的本轮管理权限，其他角色仅修复历史 getMenu POST 错误。
-- 回退：对照执行前备份撤销本次新增记录；不要覆盖执行后新增的业务数据。
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
START TRANSACTION;

SET @business_admin_role = (SELECT authority_id FROM sys_users WHERE username='admin' AND deleted_at IS NULL ORDER BY id LIMIT 1);
SET @business_parent_menu = (SELECT id FROM sys_base_menus WHERE name='superAdmin' AND deleted_at IS NULL LIMIT 1);

-- 新角色旧版本曾将首页误关联到 ID=1：只补正确首页，不删除可能有意分配的管理菜单。
INSERT INTO sys_authority_menus (sys_base_menu_id, sys_authority_authority_id)
SELECT m.id,a.authority_id FROM sys_authorities a JOIN sys_base_menus m ON m.name='index' AND m.deleted_at IS NULL
WHERE a.default_router='index' AND a.deleted_at IS NULL
AND NOT EXISTS (SELECT 1 FROM sys_authority_menus am WHERE am.sys_base_menu_id=m.id AND am.sys_authority_authority_id=a.authority_id);

INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT DISTINCT 'p',old.v0,old.v1,'GET' FROM casbin_rule old
WHERE old.ptype='p' AND old.v1='/v1/sys/menu/getMenu' AND old.v2='POST'
AND NOT EXISTS (SELECT 1 FROM casbin_rule current WHERE current.ptype='p' AND current.v0=old.v0 AND current.v1=old.v1 AND current.v2='GET');
DELETE FROM casbin_rule WHERE ptype='p' AND v1='/v1/sys/menu/getMenu' AND v2='POST';

INSERT INTO sys_base_menus (created_at,updated_at,menu_level,parent_id,path,name,hidden,component,sort,active_name,keep_alive,default_menu,title,icon,close_tab)
SELECT NOW(3),NOW(3),0,COALESCE(@business_parent_menu,0),'business-tools','businessTools',0,'views/business/tools/index.vue',90,'',0,0,'管理工具','lucide:wrench',0
WHERE NOT EXISTS (SELECT 1 FROM sys_base_menus WHERE name='businessTools' AND deleted_at IS NULL);

INSERT INTO sys_authority_menus (sys_base_menu_id,sys_authority_authority_id)
SELECT m.id,@business_admin_role FROM sys_base_menus m
WHERE @business_admin_role IS NOT NULL AND m.deleted_at IS NULL
AND m.name IN ('superAdmin','index','user','authority','menu','api','dictionary','businessTools')
AND NOT EXISTS (SELECT 1 FROM sys_authority_menus am WHERE am.sys_base_menu_id=m.id AND am.sys_authority_authority_id=@business_admin_role);

CREATE TEMPORARY TABLE business_api_seed (path VARCHAR(191), method VARCHAR(16), description VARCHAR(191), PRIMARY KEY(path,method)) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
INSERT INTO business_api_seed (path,method,description) VALUES
('/v1/sys/api/createApi','POST','新增API'),
('/v1/sys/api/deleteApi','DELETE','删除API'),
('/v1/sys/api/deleteApisByIds','DELETE','批量删除API'),
('/v1/sys/api/getAllApiList','GET','全部API'),
('/v1/sys/api/getApiList','GET','API列表'),
('/v1/sys/api/updateApi','PUT','编辑API'),
('/v1/sys/authority/addAuthorityMenu','POST','保存角色菜单'),
('/v1/sys/authority/createAuthority','POST','新增角色'),
('/v1/sys/authority/deleteAuthority','DELETE','删除角色'),
('/v1/sys/authority/getAuthorityList','GET','角色列表'),
('/v1/sys/authority/updateAuthority','PUT','编辑角色'),
('/v1/sys/base/sendEmailCode','POST','发送邮箱验证码'),
('/v1/sys/base/uploadFileImg','POST','上传图片'),
('/v1/sys/casbin/getPathByAuthorityId','GET','角色接口权限'),
('/v1/sys/casbin/updateCasbinData','PUT','按路径保存接口权限'),
('/v1/sys/casbin/updateCasbinDataByApiIds','PUT','按ID保存接口权限'),
('/v1/sys/deleteUser','DELETE','删除用户'),
('/v1/sys/getUserList','GET','用户列表'),
('/v1/sys/register','POST','新增用户'),
('/v1/sys/resetUserPassword','PUT','重置用户密码'),
('/v1/sys/updateUserInfo','PUT','编辑用户'),
('/v1/sys/dictionary/createSysDictionary','POST','新增字典'),
('/v1/sys/dictionary/createSysDictionaryInfo','POST','新增字典项'),
('/v1/sys/dictionary/deleteSysDictionary','DELETE','删除字典'),
('/v1/sys/dictionary/deleteSysDictionaryInfo','DELETE','删除字典项'),
('/v1/sys/dictionary/getSysDictionaryDetails','GET','字典详情'),
('/v1/sys/dictionary/getSysDictionaryInfoList','GET','字典项列表'),
('/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','GET','字典项详情'),
('/v1/sys/dictionary/getSysDictionaryList','GET','字典列表'),
('/v1/sys/dictionary/updateSysDictionary','PUT','编辑字典'),
('/v1/sys/dictionary/updateSysDictionaryInfo','PUT','编辑字典项'),
('/v1/sys/menu/addBaseMenu','POST','新增菜单'),
('/v1/sys/menu/deleteBaseMenu','DELETE','删除菜单'),
('/v1/sys/menu/getAuthorityButtons','GET','角色按钮权限'),
('/v1/sys/menu/getBaseMenuById','GET','菜单详情'),
('/v1/sys/menu/getBaseMenuTree','GET','菜单树'),
('/v1/sys/menu/getMenu','GET','当前菜单'),
('/v1/sys/menu/getMenuAuthority','GET','角色菜单'),
('/v1/sys/menu/getMenuList','GET','菜单管理列表'),
('/v1/sys/menu/updateAuthorityButtons','PUT','保存按钮授权'),
('/v1/sys/menu/updateBaseMenu','PUT','编辑菜单');

INSERT INTO sys_apis (created_at,updated_at,path,method,description,api_group)
SELECT NOW(3),NOW(3),s.path,s.method,s.description,'系统管理' FROM business_api_seed s
WHERE NOT EXISTS (SELECT 1 FROM sys_apis a WHERE a.path=s.path AND a.method=s.method AND a.deleted_at IS NULL);

INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p',CAST(@business_admin_role AS CHAR),s.path,s.method FROM business_api_seed s
WHERE @business_admin_role IS NOT NULL
AND NOT EXISTS (
  SELECT 1 FROM casbin_rule p WHERE p.ptype='p'
  AND p.v0 COLLATE utf8mb4_general_ci=CAST(@business_admin_role AS CHAR)
  AND p.v1 COLLATE utf8mb4_general_ci=s.path
  AND p.v2 COLLATE utf8mb4_general_ci=s.method
);

CREATE TEMPORARY TABLE business_button_seed (menu_name VARCHAR(191), name VARCHAR(191), description VARCHAR(191)) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
INSERT INTO business_button_seed (menu_name,name,description) VALUES
('user','create','新增'),
('user','update','编辑'),
('user','delete','删除'),
('authority','create','新增'),
('authority','update','编辑'),
('authority','delete','删除'),
('menu','create','新增'),
('menu','update','编辑'),
('menu','delete','删除'),
('api','create','新增'),
('api','update','编辑'),
('api','delete','删除'),
('dictionary','create','新增'),
('dictionary','update','编辑'),
('dictionary','delete','删除'),
('user','resetPassword','重置密码'),
('authority','menus','菜单授权'),
('authority','apis','接口授权'),
('authority','buttons','按钮授权'),
('dictionary','items','字典项管理'),
('businessTools','upload','上传图片'),
('businessTools','sendEmail','发送邮箱验证码');

INSERT INTO sys_base_menu_btns (created_at,updated_at,name,`desc`,sys_base_menu_id)
SELECT NOW(3),NOW(3),s.name,s.description,m.id FROM business_button_seed s JOIN sys_base_menus m ON m.name=s.menu_name AND m.deleted_at IS NULL
WHERE NOT EXISTS (SELECT 1 FROM sys_base_menu_btns b WHERE b.sys_base_menu_id=m.id AND b.name=s.name AND b.deleted_at IS NULL);

INSERT INTO sys_authority_btns (authority_id,sys_menu_id,sys_base_menu_btn_id)
SELECT @business_admin_role,m.id,b.id FROM business_button_seed s
JOIN sys_base_menus m ON m.name=s.menu_name AND m.deleted_at IS NULL
JOIN sys_base_menu_btns b ON b.sys_base_menu_id=m.id AND b.name=s.name AND b.deleted_at IS NULL
WHERE @business_admin_role IS NOT NULL
AND NOT EXISTS (SELECT 1 FROM sys_authority_btns ab WHERE ab.authority_id=@business_admin_role AND ab.sys_menu_id=m.id AND ab.sys_base_menu_btn_id=b.id);

COMMIT;
DROP TEMPORARY TABLE business_api_seed;
DROP TEMPORARY TABLE business_button_seed;
-- 完成后重启 RPC（或重载权限策略）并刷新前端权限，以加载本次增量初始化。
