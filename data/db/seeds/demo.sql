-- Fresh-install demo data only. This is NOT an upgrade or reset script.
-- Appended after all migrations/operator bootstraps by build-fresh-sql.ps1.
-- admin / 123456; all other demo accounts / Demo@123456.
-- Matching image objects are installed by the RustFS storage-init image.
-- No usable sessions, tokens or email verification codes are seeded.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
START TRANSACTION;
SET @demo_now = UTC_TIMESTAMP(3);
SET @demo_password = '$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy';

-- Keep resource IDs, route names, paths and components stable. Sort is local
-- to siblings, in steps of ten so contributors can insert new menu entries.
UPDATE sys_base_menus SET sort = CASE name
 WHEN 'index' THEN 0
 WHEN 'superAdmin' THEN 10
 WHEN 'app-members-users' THEN 20
 WHEN 'app-memberships' THEN 30
 WHEN 'user' THEN 10
 WHEN 'authority' THEN 20
 WHEN 'organization-departments' THEN 30
 WHEN 'organization-positions' THEN 40
 WHEN 'menu' THEN 50
 WHEN 'api' THEN 60
 WHEN 'dictionary' THEN 70
 WHEN 'files' THEN 80
 WHEN 'sessions' THEN 90
 WHEN 'audit' THEN 100
 WHEN 'ai-agent' THEN 110
 WHEN 'businessTools' THEN 120
 ELSE sort END,
 menu_level = CASE WHEN parent_id=0 THEN 0 ELSE 1 END;
UPDATE sys_base_menus SET title='系统管理',icon='lucide:settings' WHERE name='superAdmin';
UPDATE sys_base_menus SET title='API 管理' WHERE name='api';
UPDATE sys_base_menus SET title='AI 助手' WHERE name='ai-agent';
UPDATE sys_base_menus SET icon=CASE name
 WHEN 'index' THEN 'lucide:house'
 WHEN 'user' THEN 'lucide:users'
 WHEN 'authority' THEN 'lucide:shield-check'
 WHEN 'organization-departments' THEN 'lucide:building-2'
 WHEN 'organization-positions' THEN 'lucide:briefcase-business'
 WHEN 'menu' THEN 'lucide:list-tree'
 WHEN 'api' THEN 'lucide:network'
 WHEN 'dictionary' THEN 'lucide:book-open'
 WHEN 'files' THEN 'lucide:folder-open'
 WHEN 'sessions' THEN 'lucide:monitor'
 WHEN 'audit' THEN 'lucide:clipboard-list'
 WHEN 'ai-agent' THEN 'lucide:bot'
 WHEN 'app-members-users' THEN 'lucide:contact'
 WHEN 'app-memberships' THEN 'lucide:crown'
 ELSE icon END;
UPDATE sys_dictionary_info SET sort=CASE
 WHEN sys_dictionary_id=1 THEN value*10
 WHEN sys_dictionary_id=2 THEN CASE extend
  WHEN 'GET' THEN 10 WHEN 'POST' THEN 20 WHEN 'PUT' THEN 30
  WHEN 'DELETE' THEN 40 WHEN 'PATCH' THEN 50 WHEN 'HEAD' THEN 60
  WHEN 'OPTIONS' THEN 70 ELSE sort END
 ELSE sort END;
UPDATE sys_dictionaries SET name='API 请求方法',`desc`='HTTP 请求方法' WHERE type='method';

-- Three explicit roles; no wildcard policies or role inheritance.
UPDATE sys_authorities SET authority_name='超级管理员',default_router='index' WHERE authority_id=1;
INSERT INTO sys_authorities(created_at,updated_at,authority_id,authority_name,parent_id,default_router) VALUES
 (@demo_now,@demo_now,2,'运营人员',0,'index'),
 (@demo_now,@demo_now,3,'只读观察员',0,'index');
UPDATE sys_users SET nick_name='超级管理员',header_img='/default-avatar.svg',phone='',email='' WHERE id=1;
INSERT INTO sys_users(id,created_at,updated_at,uuid,username,password,nick_name,side_mode,header_img,base_color,active_color,authority_id,phone,email,enable,session_version) VALUES
 (2,@demo_now,@demo_now,'00000000-0000-4000-8000-000000000002','operator',@demo_password,'演示运营','dark','/default-avatar.svg','#000','#fff',2,'','operator@example.test',1,1),
 (3,@demo_now,@demo_now,'00000000-0000-4000-8000-000000000003','viewer',@demo_password,'演示观察员','dark','/default-avatar.svg','#000','#fff',3,'','viewer@example.test',1,1),
 (4,@demo_now,@demo_now,'00000000-0000-4000-8000-000000000004','disabled',@demo_password,'禁用账号示例','dark','/default-avatar.svg','#000','#fff',3,'','disabled-admin@example.test',2,1);
INSERT INTO sys_user_authority(sys_user_id,sys_authority_authority_id) VALUES (2,2),(3,3),(4,3);

INSERT INTO sys_departments(id,name,code,parent_id,sort,status,leader) VALUES
 (1,'示例组织','DEMO',0,10,1,''),
 (2,'运营部','DEMO_OPS',1,10,1,''),
 (3,'研发部','DEMO_DEV',1,20,1,'');
INSERT INTO sys_positions(id,name,code,sort,status) VALUES
 (1,'运营专员','DEMO_OPERATOR',10,1),
 (2,'观察员','DEMO_VIEWER',20,1);
INSERT INTO sys_user_departments(user_id,department_id) VALUES (1,1),(2,2),(3,3),(4,3);
INSERT INTO sys_user_positions(user_id,position_id) VALUES (2,1),(3,2),(4,2);
INSERT INTO sys_role_data_scopes(authority_id,scope) VALUES (1,'all'),(2,'self'),(3,'self');

-- Administrator gets all currently registered resources explicitly at install
-- time. Future APIs/buttons still need an explicit grant after an upgrade.
INSERT INTO sys_authority_menus(sys_base_menu_id,sys_authority_authority_id)
 SELECT m.id,1 FROM sys_base_menus m WHERE m.deleted_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM sys_authority_menus a WHERE a.sys_base_menu_id=m.id AND a.sys_authority_authority_id=1);
INSERT INTO sys_authority_btns(authority_id,sys_menu_id,sys_base_menu_btn_id)
 SELECT 1,b.sys_base_menu_id,b.id FROM sys_base_menu_btns b WHERE b.deleted_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM sys_authority_btns a WHERE a.authority_id=1 AND a.sys_base_menu_btn_id=b.id);
INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5)
 SELECT 'p','1',a.path,a.method,'','','' FROM sys_apis a WHERE a.deleted_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM casbin_rule p WHERE p.ptype='p' AND p.v0='1'
  AND p.v1 COLLATE utf8mb4_bin=a.path COLLATE utf8mb4_bin
  AND p.v2 COLLATE utf8mb4_bin=a.method COLLATE utf8mb4_bin);

-- Operator: App account review/status/avatar and own file uploads. No system
-- configuration, role changes, device revocation or membership grant/revoke.
INSERT INTO sys_authority_menus(sys_base_menu_id,sys_authority_authority_id)
 SELECT id,2 FROM sys_base_menus WHERE name IN('index','superAdmin','files','businessTools','app-members-users','app-memberships');
INSERT INTO sys_authority_btns(authority_id,sys_menu_id,sys_base_menu_btn_id)
 SELECT 2,sys_base_menu_id,id FROM sys_base_menu_btns
 WHERE permission_key IN('app-members-users:review','app-members-users:status','app-members-users:avatar','files:upload','businessTools:upload');
INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5)
 SELECT 'p','2',a.path,a.method,'','','' FROM sys_apis a WHERE a.deleted_at IS NULL AND (
  (a.method='GET' AND a.path IN('/v1/sys/menu/getMenu','/v1/sys/app-members/users','/v1/sys/app-members/sessions','/v1/sys/app-members/plans','/v1/sys/app-members/grants','/v1/sys/files/list','/v1/sys/files/url'))
  OR (a.method='PUT' AND a.path IN('/v1/sys/app-members/status','/v1/sys/app-members/avatar'))
  OR (a.method='POST' AND a.path='/v1/sys/base/uploadFileImg')
 );

-- Viewer: App users/memberships read-only; no action buttons and no writes.
INSERT INTO sys_authority_menus(sys_base_menu_id,sys_authority_authority_id)
 SELECT id,3 FROM sys_base_menus WHERE name IN('index','app-members-users','app-memberships');
INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5)
 SELECT 'p','3',a.path,a.method,'','','' FROM sys_apis a WHERE a.deleted_at IS NULL
 AND a.method='GET' AND a.path IN('/v1/sys/menu/getMenu','/v1/sys/app-members/users','/v1/sys/app-members/sessions','/v1/sys/app-members/plans','/v1/sys/app-members/grants');

-- App accounts are separate from administrator accounts. example.test is
-- deliberately non-deliverable; identity_verified remains false.
INSERT INTO app_users(id,nickname,status,session_version,review_reason,reviewed_by,reviewed_at,created_at,updated_at,registration_ip,registration_region,avatar_url) VALUES
 (1,'普通用户示例','active',1,'初始化演示账号',1,@demo_now,@demo_now,@demo_now,'','',''),
 (2,'会员用户示例','active',1,'初始化演示账号',1,@demo_now,@demo_now,@demo_now,'','',''),
 (3,'待审核用户示例','pending',1,'',0,NULL,@demo_now,@demo_now,'','',''),
 (4,'禁用用户示例','disabled',1,'内置演示：账号已禁用',1,@demo_now,@demo_now,@demo_now,'','',''),
 (5,'审核拒绝示例','rejected',1,'内置演示：注册资料待完善',1,@demo_now,@demo_now,@demo_now,'','',''),
 (6,'会员过期示例','active',1,'初始化演示账号',1,@demo_now,@demo_now,@demo_now,'','','');
INSERT INTO app_user_identities(id,user_id,kind,identifier,password_hash,verified_at,created_at) VALUES
 (1,1,'email','member@example.test',@demo_password,NULL,@demo_now),
 (2,2,'email','vip@example.test',@demo_password,NULL,@demo_now),
 (3,3,'email','pending@example.test',@demo_password,NULL,@demo_now),
 (4,4,'email','disabled@example.test',@demo_password,NULL,@demo_now),
 (5,5,'email','rejected@example.test',@demo_password,NULL,@demo_now),
 (6,6,'email','expired@example.test',@demo_password,NULL,@demo_now);
INSERT INTO app_membership_grants(id,user_id,plan_id,plan_code,plan_version,entitlements,request_id,days,starts_at,ends_at,granted_by,reason,created_at)
 SELECT 1,2,id,code,version,entitlements,'demo-vip-initial-grant',30,@demo_now,DATE_ADD(@demo_now,INTERVAL 30 DAY),1,'初始化演示会员，导入时起有效30天',@demo_now
 FROM app_membership_plans WHERE code='vip' AND version=1 AND enabled=1;
INSERT INTO app_membership_grants(id,user_id,plan_id,plan_code,plan_version,entitlements,request_id,days,starts_at,ends_at,granted_by,reason,created_at)
 SELECT 2,6,id,code,version,entitlements,'demo-vip-expired-grant',30,DATE_SUB(@demo_now,INTERVAL 60 DAY),DATE_SUB(@demo_now,INTERVAL 30 DAY),1,'内置演示：会员已过期',DATE_SUB(@demo_now,INTERVAL 60 DAY)
 FROM app_membership_plans WHERE code='vip' AND version=1 AND enabled=1;
INSERT INTO app_membership_grants(id,user_id,plan_id,plan_code,plan_version,entitlements,request_id,days,starts_at,ends_at,revoked_at,granted_by,revoked_by,reason,revoke_reason,created_at)
 SELECT 3,1,id,code,version,entitlements,'demo-vip-revoked-grant',30,DATE_SUB(@demo_now,INTERVAL 10 DAY),DATE_ADD(@demo_now,INTERVAL 20 DAY),DATE_SUB(@demo_now,INTERVAL 1 DAY),1,1,'内置演示：曾获会员授予','内置演示：会员已撤销',DATE_SUB(@demo_now,INTERVAL 10 DAY)
 FROM app_membership_plans WHERE code='vip' AND version=1 AND enabled=1;

-- Files match the bundled data/demo-files PNGs installed into RustFS. Store
-- object keys rather than machine-specific URLs; API builds the current URL.
INSERT INTO sys_file_resources(id,object_key,name,mime,size,owner_id,department_id,visibility,status,created_at,updated_at) VALUES
 (1,'go-zero-admin/00000000-0000-4000-8000-000000000101.png','演示图片-引用保护.png','image/png',703,2,2,'public','active',@demo_now,@demo_now),
 (2,'go-zero-admin/00000000-0000-4000-8000-000000000102.png','演示图片-可删除.png','image/png',611,1,1,'public','active',@demo_now,@demo_now),
 (3,'go-zero-admin/00000000-0000-4000-8000-000000000103.png','演示图片-私有下载.png','image/png',525,2,2,'private','active',@demo_now,@demo_now);
INSERT INTO sys_file_references(file_id,object_type,object_id,created_at)
 VALUES (1,'demo_attachment','quickstart',@demo_now);

UPDATE sys_policy_versions SET version=version+1,updated_at=@demo_now WHERE id=1;
UPDATE sys_permission_versions SET revision=revision+1 WHERE id=1;
COMMIT;
SET @demo_password=NULL;
SET @demo_now=NULL;
