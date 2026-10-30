-- 本机前端联调数据；仅用于 gozero-new-mysql / gozero_admin。
-- 固定 fixture ID 位于 9001000–9001999；只检查实际写入/引用的 ID，不认领整个区间。
-- 测试账号沿用管理员 id=1 的现有密码哈希，脚本不含密码或哈希常量。
-- 原管理员、原角色分组及原 Casbin 规则不修改；已有 fixture 的可编辑内容保留。
-- 固定 ID 与自然身份双向校验，不能仅凭 demo_ 前缀认领已有记录。
-- 文件、App、AI 等本脚本不写入/不引用的记录，以及区间空洞中的记录均不影响重跑。

SET NAMES utf8mb4;
START TRANSACTION;
CREATE TEMPORARY TABLE demo_fixture_checks (check_name VARCHAR(64) PRIMARY KEY);
INSERT INTO demo_fixture_checks VALUES ('required_base_records');
-- 重复主键是有意的保护：mysql 不带 --force，任一检查失败即退出并回滚整个事务。
INSERT INTO demo_fixture_checks SELECT 'required_base_records' WHERE
  NOT EXISTS (SELECT 1 FROM sys_user WHERE id=1 AND username='admin' AND CHAR_LENGTH(password_hash)>0)
  OR NOT EXISTS (SELECT 1 FROM sys_role WHERE id=1 AND code='super_admin')
  OR NOT EXISTS (SELECT 1 FROM sys_dept WHERE id=1);

-- 先在与真实表结构一致的临时表中准备预期行；全部身份通过检查后才写入真实表。
CREATE TEMPORARY TABLE demo_fixture_sys_dept LIKE sys_dept;
CREATE TEMPORARY TABLE demo_fixture_sys_user LIKE sys_user;
CREATE TEMPORARY TABLE demo_fixture_sys_role LIKE sys_role;
CREATE TEMPORARY TABLE demo_fixture_sys_menu LIKE sys_menu;
CREATE TEMPORARY TABLE demo_fixture_sys_role_menu LIKE sys_role_menu;
CREATE TEMPORARY TABLE demo_fixture_sys_api LIKE sys_api;
CREATE TEMPORARY TABLE demo_fixture_sys_dict LIKE sys_dict;
CREATE TEMPORARY TABLE demo_fixture_casbin_rule LIKE casbin_rule;
CREATE TEMPORARY TABLE demo_fixture_sys_audit_log LIKE sys_audit_log;

-- 组织层级：两个一级示例部门，三个下级，包含停用部门。
INSERT INTO demo_fixture_sys_dept (id,parent_id,name,leader,phone,email,sort,status,path,created_at,updated_at) VALUES
  (9001000, 1, 'demo_研发中心', '示例负责人甲', '000-0000-0001', 'demo.rd@example.test', 10, 1, '/1/9001000/', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001001, 9001000, 'demo_平台研发组', '示例负责人乙', '000-0000-0002', 'demo.platform@example.test', 20, 1, '/1/9001000/9001001/', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001002, 9001000, 'demo_质量保障组', '示例负责人丙', '000-0000-0003', 'demo.qa@example.test', 30, 1, '/1/9001000/9001002/', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001003, 1, 'demo_客户服务中心', '示例负责人丁', '000-0000-0004', 'demo.service@example.test', 40, 1, '/1/9001003/', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001004, 9001003, 'demo_停用示例组', '示例负责人戊', '000-0000-0005', 'demo.disabled@example.test', 50, 2, '/1/9001003/9001004/', '2026-10-30 09:00:00', '2026-10-30 09:00:00');

-- 28 个测试用户；每第 7 个停用，用于分页和状态筛选。
INSERT INTO demo_fixture_sys_user (id,username,password_hash,nickname,real_name,email,phone,dept_id,avatar_file_id,gender,status,remark,last_login_at,last_login_ip,created_at,updated_at)
SELECT 9001100,'demo_user_01',administrator.password_hash,'demo_联调用户01','示例用户1','demo.user01@example.test','000-0000-0001',9001000,NULL,0,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:01:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001101,'demo_user_02',administrator.password_hash,'demo_联调用户02','示例用户2','demo.user02@example.test','000-0000-0002',9001001,NULL,1,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:02:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001102,'demo_user_03',administrator.password_hash,'demo_联调用户03','示例用户3','demo.user03@example.test','000-0000-0003',9001002,NULL,2,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:03:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001103,'demo_user_04',administrator.password_hash,'demo_联调用户04','示例用户4','demo.user04@example.test','000-0000-0004',9001003,NULL,0,1,'demo_前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001104,'demo_user_05',administrator.password_hash,'demo_联调用户05','示例用户5','demo.user05@example.test','000-0000-0005',9001004,NULL,1,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:05:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001105,'demo_user_06',administrator.password_hash,'demo_联调用户06','示例用户6','demo.user06@example.test','000-0000-0006',9001000,NULL,2,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:06:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001106,'demo_user_07',administrator.password_hash,'demo_联调用户07','示例用户7','demo.user07@example.test','000-0000-0007',9001001,NULL,0,2,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:07:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001107,'demo_user_08',administrator.password_hash,'demo_联调用户08','示例用户8','demo.user08@example.test','000-0000-0008',9001002,NULL,1,1,'demo_前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001108,'demo_user_09',administrator.password_hash,'demo_联调用户09','示例用户9','demo.user09@example.test','000-0000-0009',9001003,NULL,2,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:09:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001109,'demo_user_10',administrator.password_hash,'demo_联调用户10','示例用户10','demo.user10@example.test','000-0000-0010',9001004,NULL,0,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:10:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001110,'demo_user_11',administrator.password_hash,'demo_联调用户11','示例用户11','demo.user11@example.test','000-0000-0011',9001000,NULL,1,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:11:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001111,'demo_user_12',administrator.password_hash,'demo_联调用户12','示例用户12','demo.user12@example.test','000-0000-0012',9001001,NULL,2,1,'demo_前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001112,'demo_user_13',administrator.password_hash,'demo_联调用户13','示例用户13','demo.user13@example.test','000-0000-0013',9001002,NULL,0,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:13:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001113,'demo_user_14',administrator.password_hash,'demo_联调用户14','示例用户14','demo.user14@example.test','000-0000-0014',9001003,NULL,1,2,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:14:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001114,'demo_user_15',administrator.password_hash,'demo_联调用户15','示例用户15','demo.user15@example.test','000-0000-0015',9001004,NULL,2,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:15:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001115,'demo_user_16',administrator.password_hash,'demo_联调用户16','示例用户16','demo.user16@example.test','000-0000-0016',9001000,NULL,0,1,'demo_前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001116,'demo_user_17',administrator.password_hash,'demo_联调用户17','示例用户17','demo.user17@example.test','000-0000-0017',9001001,NULL,1,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:17:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001117,'demo_user_18',administrator.password_hash,'demo_联调用户18','示例用户18','demo.user18@example.test','000-0000-0018',9001002,NULL,2,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:18:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001118,'demo_user_19',administrator.password_hash,'demo_联调用户19','示例用户19','demo.user19@example.test','000-0000-0019',9001003,NULL,0,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:19:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001119,'demo_user_20',administrator.password_hash,'demo_联调用户20','示例用户20','demo.user20@example.test','000-0000-0020',9001004,NULL,1,1,'demo_前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001120,'demo_user_21',administrator.password_hash,'demo_联调用户21','示例用户21','demo.user21@example.test','000-0000-0021',9001000,NULL,2,2,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:21:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001121,'demo_user_22',administrator.password_hash,'demo_联调用户22','示例用户22','demo.user22@example.test','000-0000-0022',9001001,NULL,0,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:22:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001122,'demo_user_23',administrator.password_hash,'demo_联调用户23','示例用户23','demo.user23@example.test','000-0000-0023',9001002,NULL,1,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:23:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001123,'demo_user_24',administrator.password_hash,'demo_联调用户24','示例用户24','demo.user24@example.test','000-0000-0024',9001003,NULL,2,1,'demo_前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001124,'demo_user_25',administrator.password_hash,'demo_联调用户25','示例用户25','demo.user25@example.test','000-0000-0025',9001004,NULL,0,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:25:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001125,'demo_user_26',administrator.password_hash,'demo_联调用户26','示例用户26','demo.user26@example.test','000-0000-0026',9001000,NULL,1,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:26:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001126,'demo_user_27',administrator.password_hash,'demo_联调用户27','示例用户27','demo.user27@example.test','000-0000-0027',9001001,NULL,2,1,'demo_前端联调账号，请勿用于真实业务','2026-10-30 08:27:00','127.0.0.1','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1
UNION ALL
SELECT 9001127,'demo_user_28',administrator.password_hash,'demo_联调用户28','示例用户28','demo.user28@example.test','000-0000-0028',9001002,NULL,0,2,'demo_前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-30 09:00:00' FROM sys_user administrator WHERE administrator.id=1;

INSERT INTO demo_fixture_sys_role (id,name,code,description,data_scope,sort,status,created_at,updated_at) VALUES
  (9001200, 'demo_全域查看', 'demo_all', 'demo_前端联调角色，仅配置 GET 权限；数据范围以实际后端实现为准', 'ALL', 10, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001201, 'demo_部门及下级', 'demo_dept_children', 'demo_前端联调角色，仅配置 GET 权限；数据范围以实际后端实现为准', 'DEPT_AND_CHILDREN', 11, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001202, 'demo_部门查看', 'demo_dept', 'demo_前端联调角色，仅配置 GET 权限；数据范围以实际后端实现为准', 'DEPT', 12, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001203, 'demo_本人查看', 'demo_self', 'demo_前端联调角色，仅配置 GET 权限；数据范围以实际后端实现为准', 'SELF', 13, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00');

INSERT INTO demo_fixture_sys_menu (id,parent_id,title,name,type,path,component,perm_code,icon,sort,visible,status,created_at,updated_at) VALUES
  (9001300, 0, 'demo_系统管理', 'demo_system', 1, '/sys', '', '', 'system', 10, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001301, 0, 'demo_工作台', 'dashboard', 2, '/dashboard', 'dashboard', '', 'dashboard', 0, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001302, 9001300, 'demo_用户管理', 'user', 2, '/sys/user', 'user', '', 'user', 20, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001303, 9001300, 'demo_角色管理', 'role', 2, '/sys/role', 'role', '', 'role', 21, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001304, 9001300, 'demo_菜单管理', 'menu', 2, '/sys/menu', 'menu', '', 'menu', 22, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001305, 9001300, 'demo_部门管理', 'dept', 2, '/sys/dept', 'dept', '', 'dept', 23, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001306, 9001300, 'demo_字典管理', 'dict', 2, '/sys/dict', 'dict', '', 'dict', 24, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001307, 9001300, 'demo_API 资源', 'api', 2, '/sys/api', 'api', '', 'api', 25, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001308, 9001300, 'demo_文件管理', 'file', 2, '/sys/file', 'file', '', 'file', 26, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001309, 9001300, 'demo_审计日志', 'audit', 2, '/sys/audit', 'audit', '', 'audit', 27, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001310, 9001300, 'demo_个人中心', 'profile', 2, '/sys/profile', 'profile', '', 'profile', 28, 1, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001320, 9001302, 'demo_新增用户', 'demo_user_add', 3, '', '', 'sys:user:add', '', 0, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001321, 9001302, 'demo_编辑用户', 'demo_user_edit', 3, '', '', 'sys:user:edit', '', 1, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001322, 9001302, 'demo_删除用户', 'demo_user_delete', 3, '', '', 'sys:user:delete', '', 2, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001323, 9001302, 'demo_重置密码', 'demo_user_reset_password', 3, '', '', 'sys:user:reset-password', '', 3, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001324, 9001303, 'demo_新增角色', 'demo_role_add', 3, '', '', 'sys:role:add', '', 4, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001325, 9001303, 'demo_编辑角色', 'demo_role_edit', 3, '', '', 'sys:role:edit', '', 5, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001326, 9001303, 'demo_删除角色', 'demo_role_delete', 3, '', '', 'sys:role:delete', '', 6, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001327, 9001303, 'demo_分配菜单', 'demo_role_menus', 3, '', '', 'sys:role:menus', '', 7, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001328, 9001304, 'demo_新增菜单', 'demo_menu_add', 3, '', '', 'sys:menu:add', '', 8, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001329, 9001304, 'demo_编辑菜单', 'demo_menu_edit', 3, '', '', 'sys:menu:edit', '', 9, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001330, 9001304, 'demo_删除菜单', 'demo_menu_delete', 3, '', '', 'sys:menu:delete', '', 10, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001331, 9001305, 'demo_新增部门', 'demo_dept_add', 3, '', '', 'sys:dept:add', '', 11, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001332, 9001305, 'demo_编辑部门', 'demo_dept_edit', 3, '', '', 'sys:dept:edit', '', 12, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001333, 9001305, 'demo_删除部门', 'demo_dept_delete', 3, '', '', 'sys:dept:delete', '', 13, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001334, 9001306, 'demo_新增字典', 'demo_dict_add', 3, '', '', 'sys:dict:add', '', 14, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001335, 9001306, 'demo_编辑字典', 'demo_dict_edit', 3, '', '', 'sys:dict:edit', '', 15, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001336, 9001306, 'demo_删除字典', 'demo_dict_delete', 3, '', '', 'sys:dict:delete', '', 16, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001337, 9001307, 'demo_删除 API', 'demo_api_delete', 3, '', '', 'sys:api:delete', '', 17, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001338, 9001308, 'demo_删除文件', 'demo_file_delete', 3, '', '', 'sys:file:delete', '', 18, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001339, 9001310, 'demo_编辑资料', 'demo_profile_edit', 3, '', '', 'sys:profile:edit', '', 19, 0, 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00');

-- 仅给既有超级管理员角色添加测试菜单关联；demo_ 角色仅分配页面、不分配写操作按钮。
INSERT INTO demo_fixture_sys_role_menu (id,role_id,menu_id,created_at) VALUES
  (9001400, 1, 9001300, '2026-10-30 09:00:00'),
  (9001401, 1, 9001301, '2026-10-30 09:00:00'),
  (9001402, 1, 9001302, '2026-10-30 09:00:00'),
  (9001403, 1, 9001303, '2026-10-30 09:00:00'),
  (9001404, 1, 9001304, '2026-10-30 09:00:00'),
  (9001405, 1, 9001305, '2026-10-30 09:00:00'),
  (9001406, 1, 9001306, '2026-10-30 09:00:00'),
  (9001407, 1, 9001307, '2026-10-30 09:00:00'),
  (9001408, 1, 9001308, '2026-10-30 09:00:00'),
  (9001409, 1, 9001309, '2026-10-30 09:00:00'),
  (9001410, 1, 9001310, '2026-10-30 09:00:00'),
  (9001411, 1, 9001320, '2026-10-30 09:00:00'),
  (9001412, 1, 9001321, '2026-10-30 09:00:00'),
  (9001413, 1, 9001322, '2026-10-30 09:00:00'),
  (9001414, 1, 9001323, '2026-10-30 09:00:00'),
  (9001415, 1, 9001324, '2026-10-30 09:00:00'),
  (9001416, 1, 9001325, '2026-10-30 09:00:00'),
  (9001417, 1, 9001326, '2026-10-30 09:00:00'),
  (9001418, 1, 9001327, '2026-10-30 09:00:00'),
  (9001419, 1, 9001328, '2026-10-30 09:00:00'),
  (9001420, 1, 9001329, '2026-10-30 09:00:00'),
  (9001421, 1, 9001330, '2026-10-30 09:00:00'),
  (9001422, 1, 9001331, '2026-10-30 09:00:00'),
  (9001423, 1, 9001332, '2026-10-30 09:00:00'),
  (9001424, 1, 9001333, '2026-10-30 09:00:00'),
  (9001425, 1, 9001334, '2026-10-30 09:00:00'),
  (9001426, 1, 9001335, '2026-10-30 09:00:00'),
  (9001427, 1, 9001336, '2026-10-30 09:00:00'),
  (9001428, 1, 9001337, '2026-10-30 09:00:00'),
  (9001429, 1, 9001338, '2026-10-30 09:00:00'),
  (9001430, 1, 9001339, '2026-10-30 09:00:00'),
  (9001440, 9001200, 9001300, '2026-10-30 09:00:00'),
  (9001441, 9001200, 9001301, '2026-10-30 09:00:00'),
  (9001442, 9001200, 9001302, '2026-10-30 09:00:00'),
  (9001443, 9001200, 9001303, '2026-10-30 09:00:00'),
  (9001444, 9001200, 9001304, '2026-10-30 09:00:00'),
  (9001445, 9001200, 9001305, '2026-10-30 09:00:00'),
  (9001446, 9001200, 9001306, '2026-10-30 09:00:00'),
  (9001447, 9001200, 9001307, '2026-10-30 09:00:00'),
  (9001448, 9001200, 9001308, '2026-10-30 09:00:00'),
  (9001449, 9001200, 9001309, '2026-10-30 09:00:00'),
  (9001450, 9001200, 9001310, '2026-10-30 09:00:00'),
  (9001451, 9001201, 9001300, '2026-10-30 09:00:00'),
  (9001452, 9001201, 9001301, '2026-10-30 09:00:00'),
  (9001453, 9001201, 9001302, '2026-10-30 09:00:00'),
  (9001454, 9001201, 9001303, '2026-10-30 09:00:00'),
  (9001455, 9001201, 9001304, '2026-10-30 09:00:00'),
  (9001456, 9001201, 9001305, '2026-10-30 09:00:00'),
  (9001457, 9001201, 9001306, '2026-10-30 09:00:00'),
  (9001458, 9001201, 9001307, '2026-10-30 09:00:00'),
  (9001459, 9001201, 9001308, '2026-10-30 09:00:00'),
  (9001460, 9001201, 9001309, '2026-10-30 09:00:00'),
  (9001461, 9001201, 9001310, '2026-10-30 09:00:00'),
  (9001462, 9001202, 9001300, '2026-10-30 09:00:00'),
  (9001463, 9001202, 9001301, '2026-10-30 09:00:00'),
  (9001464, 9001202, 9001302, '2026-10-30 09:00:00'),
  (9001465, 9001202, 9001303, '2026-10-30 09:00:00'),
  (9001466, 9001202, 9001304, '2026-10-30 09:00:00'),
  (9001467, 9001202, 9001305, '2026-10-30 09:00:00'),
  (9001468, 9001202, 9001306, '2026-10-30 09:00:00'),
  (9001469, 9001202, 9001307, '2026-10-30 09:00:00'),
  (9001470, 9001202, 9001308, '2026-10-30 09:00:00'),
  (9001471, 9001202, 9001309, '2026-10-30 09:00:00'),
  (9001472, 9001202, 9001310, '2026-10-30 09:00:00'),
  (9001473, 9001203, 9001300, '2026-10-30 09:00:00'),
  (9001474, 9001203, 9001301, '2026-10-30 09:00:00'),
  (9001475, 9001203, 9001302, '2026-10-30 09:00:00'),
  (9001476, 9001203, 9001303, '2026-10-30 09:00:00'),
  (9001477, 9001203, 9001304, '2026-10-30 09:00:00'),
  (9001478, 9001203, 9001305, '2026-10-30 09:00:00'),
  (9001479, 9001203, 9001306, '2026-10-30 09:00:00'),
  (9001480, 9001203, 9001307, '2026-10-30 09:00:00'),
  (9001481, 9001203, 9001308, '2026-10-30 09:00:00'),
  (9001482, 9001203, 9001309, '2026-10-30 09:00:00'),
  (9001483, 9001203, 9001310, '2026-10-30 09:00:00');

-- 路由逐条核对自当前生成 routes.go；固定 ID 与 method/path 必须精确对应，冲突时中止。
INSERT INTO demo_fixture_sys_api (id,group_name,method,path,title,created_at,updated_at) VALUES
  (9001500, 'sys/auth', 'GET', '/v1/sys/auth/captcha', 'demo_获取验证码', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001501, 'sys/auth', 'POST', '/v1/sys/auth/login', 'demo_登录', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001502, 'sys/user', 'GET', '/v1/sys/user/list', 'demo_查询用户', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001503, 'sys/user', 'GET', '/v1/sys/user/:id', 'demo_用户详情', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001504, 'sys/user', 'POST', '/v1/sys/user/create', 'demo_新增用户', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001505, 'sys/user', 'PUT', '/v1/sys/user/:id', 'demo_编辑用户', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001506, 'sys/user', 'DELETE', '/v1/sys/user/:id', 'demo_删除用户', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001507, 'sys/user', 'POST', '/v1/sys/user/:id/password', 'demo_重置密码', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001508, 'sys/user', 'POST', '/v1/sys/user/password/change', 'demo_修改本人密码', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001509, 'sys/dept', 'GET', '/v1/sys/dept/tree', 'demo_部门树', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001510, 'sys/dept', 'POST', '/v1/sys/dept/create', 'demo_新增部门', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001511, 'sys/dept', 'PUT', '/v1/sys/dept/:id', 'demo_编辑部门', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001512, 'sys/dept', 'DELETE', '/v1/sys/dept/:id', 'demo_删除部门', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001513, 'sys/role', 'GET', '/v1/sys/role/list', 'demo_查询角色', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001514, 'sys/role', 'GET', '/v1/sys/role/:id', 'demo_角色详情', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001515, 'sys/role', 'POST', '/v1/sys/role/create', 'demo_新增角色', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001516, 'sys/role', 'PUT', '/v1/sys/role/:id', 'demo_编辑角色', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001517, 'sys/role', 'DELETE', '/v1/sys/role/:id', 'demo_删除角色', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001518, 'sys/role', 'POST', '/v1/sys/role/:id/menus', 'demo_角色菜单授权', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001519, 'sys/menu', 'GET', '/v1/sys/menu/list', 'demo_查询菜单', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001520, 'sys/menu', 'GET', '/v1/sys/menu/:id', 'demo_菜单详情', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001521, 'sys/menu', 'GET', '/v1/sys/menu/tree', 'demo_菜单树', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001522, 'sys/menu', 'GET', '/v1/sys/menu/my', 'demo_本人菜单', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001523, 'sys/menu', 'GET', '/v1/sys/menu/my/perm-codes', 'demo_本人权限码', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001524, 'sys/menu', 'POST', '/v1/sys/menu/create', 'demo_新增菜单', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001525, 'sys/menu', 'PUT', '/v1/sys/menu/:id', 'demo_编辑菜单', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001526, 'sys/menu', 'DELETE', '/v1/sys/menu/:id', 'demo_删除菜单', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001527, 'sys/dict', 'GET', '/v1/sys/dict/list', 'demo_查询字典', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001528, 'sys/dict', 'GET', '/v1/sys/dict/code/:code', 'demo_字典详情', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001529, 'sys/dict', 'POST', '/v1/sys/dict/create', 'demo_新增字典', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001530, 'sys/dict', 'PUT', '/v1/sys/dict/:id', 'demo_编辑字典', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001531, 'sys/dict', 'DELETE', '/v1/sys/dict/:id', 'demo_删除字典', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001532, 'sys/api', 'GET', '/v1/sys/api/list', 'demo_查询 API', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001533, 'sys/api', 'DELETE', '/v1/sys/api/:id', 'demo_删除 API', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001534, 'sys/file', 'GET', '/v1/sys/file/list', 'demo_查询文件', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001535, 'sys/file', 'GET', '/v1/sys/file/:id/url', 'demo_获取文件链接', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001536, 'sys/file', 'DELETE', '/v1/sys/file/:id', 'demo_删除文件', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001537, 'sys/audit', 'GET', '/v1/sys/audit/list', 'demo_查询审计日志', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001538, 'sys/profile', 'GET', '/v1/sys/profile/', 'demo_本人资料', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001539, 'sys/profile', 'PUT', '/v1/sys/profile/', 'demo_更新本人资料', '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001540, 'system', 'GET', '/ping', 'demo_健康检查', '2026-10-30 09:00:00', '2026-10-30 09:00:00');

INSERT INTO demo_fixture_sys_dict (id,name,code,items,description,status,created_at,updated_at) VALUES
  (9001600, 'demo_启停状态', 'demo_status', '[{"label":"启用","value":"1","sort":0},{"label":"停用","value":"2","sort":1}]', 'demo_前端联调字典', 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001601, 'demo_用户性别', 'demo_gender', '[{"label":"未知","value":"0","sort":0},{"label":"男","value":"1","sort":1},{"label":"女","value":"2","sort":2}]', 'demo_前端联调字典', 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001602, 'demo_数据范围', 'demo_data_scope', '[{"label":"全部","value":"ALL","sort":0},{"label":"本部门及下级","value":"DEPT_AND_CHILDREN","sort":1},{"label":"本部门","value":"DEPT","sort":2},{"label":"本人","value":"SELF","sort":3}]', 'demo_前端联调字典', 1, '2026-10-30 09:00:00', '2026-10-30 09:00:00'),
  (9001603, 'demo_工单优先级', 'demo_priority', '[{"label":"低","value":"low","sort":0},{"label":"中","value":"normal","sort":1},{"label":"高","value":"high","sort":2}]', 'demo_前端联调字典', 2, '2026-10-30 09:00:00', '2026-10-30 09:00:00');

-- 只为新 demo_ 角色和新 demo_ 用户增加只读规则，不改变任何既有主体。
INSERT INTO demo_fixture_casbin_rule (id,ptype,v0,v1,v2,v3,v4,v5,created_at) VALUES
  (9001700, 'p', 'demo_all', '*', '/v1/sys/*', 'GET', '', '', '2026-10-30 09:00:00'),
  (9001701, 'p', 'demo_dept_children', '*', '/v1/sys/*', 'GET', '', '', '2026-10-30 09:00:00'),
  (9001702, 'p', 'demo_dept', '*', '/v1/sys/*', 'GET', '', '', '2026-10-30 09:00:00'),
  (9001703, 'p', 'demo_self', '*', '/v1/sys/*', 'GET', '', '', '2026-10-30 09:00:00'),
  (9001710, 'g', 'user:9001100', 'demo_all', 'dept:9001000', '', '', '', '2026-10-30 09:00:00'),
  (9001711, 'g', 'user:9001101', 'demo_dept_children', 'dept:9001001', '', '', '', '2026-10-30 09:00:00'),
  (9001712, 'g', 'user:9001102', 'demo_dept', 'dept:9001002', '', '', '', '2026-10-30 09:00:00'),
  (9001713, 'g', 'user:9001103', 'demo_self', 'dept:9001003', '', '', '', '2026-10-30 09:00:00'),
  (9001714, 'g', 'user:9001104', 'demo_all', 'dept:9001004', '', '', '', '2026-10-30 09:00:00'),
  (9001715, 'g', 'user:9001105', 'demo_dept_children', 'dept:9001000', '', '', '', '2026-10-30 09:00:00'),
  (9001716, 'g', 'user:9001106', 'demo_dept', 'dept:9001001', '', '', '', '2026-10-30 09:00:00'),
  (9001717, 'g', 'user:9001107', 'demo_self', 'dept:9001002', '', '', '', '2026-10-30 09:00:00'),
  (9001718, 'g', 'user:9001108', 'demo_all', 'dept:9001003', '', '', '', '2026-10-30 09:00:00'),
  (9001719, 'g', 'user:9001109', 'demo_dept_children', 'dept:9001004', '', '', '', '2026-10-30 09:00:00'),
  (9001720, 'g', 'user:9001110', 'demo_dept', 'dept:9001000', '', '', '', '2026-10-30 09:00:00'),
  (9001721, 'g', 'user:9001111', 'demo_self', 'dept:9001001', '', '', '', '2026-10-30 09:00:00'),
  (9001722, 'g', 'user:9001112', 'demo_all', 'dept:9001002', '', '', '', '2026-10-30 09:00:00'),
  (9001723, 'g', 'user:9001113', 'demo_dept_children', 'dept:9001003', '', '', '', '2026-10-30 09:00:00'),
  (9001724, 'g', 'user:9001114', 'demo_dept', 'dept:9001004', '', '', '', '2026-10-30 09:00:00'),
  (9001725, 'g', 'user:9001115', 'demo_self', 'dept:9001000', '', '', '', '2026-10-30 09:00:00'),
  (9001726, 'g', 'user:9001116', 'demo_all', 'dept:9001001', '', '', '', '2026-10-30 09:00:00'),
  (9001727, 'g', 'user:9001117', 'demo_dept_children', 'dept:9001002', '', '', '', '2026-10-30 09:00:00'),
  (9001728, 'g', 'user:9001118', 'demo_dept', 'dept:9001003', '', '', '', '2026-10-30 09:00:00'),
  (9001729, 'g', 'user:9001119', 'demo_self', 'dept:9001004', '', '', '', '2026-10-30 09:00:00'),
  (9001730, 'g', 'user:9001120', 'demo_all', 'dept:9001000', '', '', '', '2026-10-30 09:00:00'),
  (9001731, 'g', 'user:9001121', 'demo_dept_children', 'dept:9001001', '', '', '', '2026-10-30 09:00:00'),
  (9001732, 'g', 'user:9001122', 'demo_dept', 'dept:9001002', '', '', '', '2026-10-30 09:00:00'),
  (9001733, 'g', 'user:9001123', 'demo_self', 'dept:9001003', '', '', '', '2026-10-30 09:00:00'),
  (9001734, 'g', 'user:9001124', 'demo_all', 'dept:9001004', '', '', '', '2026-10-30 09:00:00'),
  (9001735, 'g', 'user:9001125', 'demo_dept_children', 'dept:9001000', '', '', '', '2026-10-30 09:00:00'),
  (9001736, 'g', 'user:9001126', 'demo_dept', 'dept:9001001', '', '', '', '2026-10-30 09:00:00'),
  (9001737, 'g', 'user:9001127', 'demo_self', 'dept:9001002', '', '', '', '2026-10-30 09:00:00');

-- 以下为明确标记的模拟审计记录，不执行其中描述的业务操作。
INSERT INTO demo_fixture_sys_audit_log (id,user_id,username,module,action,method,path,params,result,error_msg,ip,user_agent,duration_ms,created_at) VALUES
  (9001800, 9001100, 'demo_user_01', 'sys/user', 'create', 'POST', '/v1/sys/user/create', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 1, '', '127.0.0.1', 'demo_frontend_fixture', 15, '2026-10-30 09:01:00'),
  (9001801, 9001101, 'demo_user_02', 'sys/user', 'update', 'PUT', '/v1/sys/user/9001100', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 1, '', '127.0.0.1', 'demo_frontend_fixture', 22, '2026-10-30 09:02:00'),
  (9001802, 9001102, 'demo_user_03', 'sys/user', 'delete', 'DELETE', '/v1/sys/user/9001127', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 1, '', '127.0.0.1', 'demo_frontend_fixture', 29, '2026-10-30 09:03:00'),
  (9001803, 9001103, 'demo_user_04', 'sys/role', 'create', 'POST', '/v1/sys/role/create', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 2, 'demo_模拟失败：用于验证错误信息展示', '127.0.0.1', 'demo_frontend_fixture', 36, '2026-10-30 09:04:00'),
  (9001804, 9001104, 'demo_user_05', 'sys/role', 'update', 'PUT', '/v1/sys/role/9001200', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 1, '', '127.0.0.1', 'demo_frontend_fixture', 43, '2026-10-30 09:05:00'),
  (9001805, 9001105, 'demo_user_06', 'sys/dept', 'create', 'POST', '/v1/sys/dept/create', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 1, '', '127.0.0.1', 'demo_frontend_fixture', 50, '2026-10-30 09:06:00'),
  (9001806, 9001106, 'demo_user_07', 'sys/dept', 'update', 'PUT', '/v1/sys/dept/9001000', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 1, '', '127.0.0.1', 'demo_frontend_fixture', 57, '2026-10-30 09:07:00'),
  (9001807, 9001107, 'demo_user_08', 'sys/dict', 'create', 'POST', '/v1/sys/dict/create', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 2, 'demo_模拟失败：用于验证错误信息展示', '127.0.0.1', 'demo_frontend_fixture', 64, '2026-10-30 09:08:00'),
  (9001808, 9001108, 'demo_user_09', 'sys/dict', 'update', 'PUT', '/v1/sys/dict/9001600', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 1, '', '127.0.0.1', 'demo_frontend_fixture', 71, '2026-10-30 09:09:00'),
  (9001809, 9001109, 'demo_user_10', 'sys/api', 'list', 'GET', '/v1/sys/api/list', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 1, '', '127.0.0.1', 'demo_frontend_fixture', 78, '2026-10-30 09:10:00'),
  (9001810, 9001110, 'demo_user_11', 'sys/file', 'list', 'GET', '/v1/sys/file/list', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 1, '', '127.0.0.1', 'demo_frontend_fixture', 85, '2026-10-30 09:11:00'),
  (9001811, 9001111, 'demo_user_12', 'sys/auth', 'login', 'POST', '/v1/sys/auth/login', '{"fixture":"demo_frontend","note":"仅用于前端联调的模拟记录，不代表真实操作"}', 2, 'demo_模拟失败：用于验证错误信息展示', '127.0.0.1', 'demo_frontend_fixture', 92, '2026-10-30 09:12:00');

-- 同一固定 ID 必须属于预期身份；同一自然身份也不能出现在另一个 ID。
-- 按字节核对字符串，防止大小写或排序规则等价掩盖不同身份；不比较昵称等可编辑内容。
INSERT INTO demo_fixture_checks VALUES
  ('identity_sys_dept'), ('identity_sys_user'), ('identity_sys_role'), ('identity_sys_menu'),
  ('identity_sys_role_menu'), ('identity_sys_api'), ('identity_sys_dict'),
  ('identity_casbin_rule'), ('identity_sys_audit_log');

INSERT INTO demo_fixture_checks
SELECT 'identity_sys_dept' FROM demo_fixture_sys_dept expected
JOIN sys_dept actual ON actual.id=expected.id OR (actual.parent_id=expected.parent_id AND actual.name=expected.name)
WHERE actual.id<>expected.id OR actual.parent_id<>expected.parent_id OR BINARY actual.name<>BINARY expected.name
LIMIT 1;

INSERT INTO demo_fixture_checks
SELECT 'identity_sys_user' FROM demo_fixture_sys_user expected
JOIN sys_user actual ON actual.id=expected.id OR actual.username=expected.username
WHERE actual.id<>expected.id OR BINARY actual.username<>BINARY expected.username
LIMIT 1;

INSERT INTO demo_fixture_checks
SELECT 'identity_sys_role' FROM demo_fixture_sys_role expected
JOIN sys_role actual ON actual.id=expected.id OR actual.code=expected.code
WHERE actual.id<>expected.id OR BINARY actual.code<>BINARY expected.code
LIMIT 1;

INSERT INTO demo_fixture_checks
SELECT 'identity_sys_menu' FROM demo_fixture_sys_menu expected
JOIN sys_menu actual ON actual.id=expected.id OR actual.name=expected.name
WHERE actual.id<>expected.id OR actual.parent_id<>expected.parent_id
  OR BINARY actual.name<>BINARY expected.name OR actual.type<>expected.type
  OR BINARY actual.path<>BINARY expected.path OR BINARY actual.component<>BINARY expected.component
  OR BINARY actual.perm_code<>BINARY expected.perm_code
LIMIT 1;

INSERT INTO demo_fixture_checks
SELECT 'identity_sys_role_menu' FROM demo_fixture_sys_role_menu expected
JOIN sys_role_menu actual ON actual.id=expected.id OR (actual.role_id=expected.role_id AND actual.menu_id=expected.menu_id)
WHERE actual.id<>expected.id OR actual.role_id<>expected.role_id OR actual.menu_id<>expected.menu_id
LIMIT 1;

INSERT INTO demo_fixture_checks
SELECT 'identity_sys_api' FROM demo_fixture_sys_api expected
JOIN sys_api actual ON actual.id=expected.id OR (actual.method=expected.method AND actual.path=expected.path)
WHERE actual.id<>expected.id OR BINARY actual.method<>BINARY expected.method OR BINARY actual.path<>BINARY expected.path
LIMIT 1;

INSERT INTO demo_fixture_checks
SELECT 'identity_sys_dict' FROM demo_fixture_sys_dict expected
JOIN sys_dict actual ON actual.id=expected.id OR actual.code=expected.code
WHERE actual.id<>expected.id OR BINARY actual.code<>BINARY expected.code
LIMIT 1;

-- 唯一索引只覆盖 ptype、v0–v3；固定 ID 复用还必须核对 v4、v5，不能静默保留另一条规则。
INSERT INTO demo_fixture_checks
SELECT 'identity_casbin_rule' FROM demo_fixture_casbin_rule expected
JOIN casbin_rule actual ON actual.id=expected.id OR (
  actual.ptype=expected.ptype AND actual.v0=expected.v0 AND actual.v1=expected.v1
  AND actual.v2=expected.v2 AND actual.v3=expected.v3)
WHERE actual.id<>expected.id OR BINARY actual.ptype<>BINARY expected.ptype
  OR BINARY actual.v0<>BINARY expected.v0 OR BINARY actual.v1<>BINARY expected.v1
  OR BINARY actual.v2<>BINARY expected.v2 OR BINARY actual.v3<>BINARY expected.v3
  OR BINARY actual.v4<>BINARY expected.v4 OR BINARY actual.v5<>BINARY expected.v5
LIMIT 1;

-- 审计记录没有独立自然键，固定 ID 必须仍是同一条明确标记的模拟事件。
INSERT INTO demo_fixture_checks
SELECT 'identity_sys_audit_log' FROM demo_fixture_sys_audit_log expected
JOIN sys_audit_log actual ON actual.id=expected.id
WHERE NOT (actual.user_id<=>expected.user_id) OR BINARY actual.username<>BINARY expected.username
  OR BINARY actual.module<>BINARY expected.module OR BINARY actual.action<>BINARY expected.action
  OR BINARY actual.method<>BINARY expected.method OR BINARY actual.path<>BINARY expected.path
  OR NOT (BINARY actual.params<=>BINARY expected.params) OR actual.result<>expected.result
  OR BINARY actual.error_msg<>BINARY expected.error_msg OR BINARY actual.ip<>BINARY expected.ip
  OR BINARY actual.user_agent<>BINARY expected.user_agent OR actual.duration_ms<>expected.duration_ms
  OR actual.created_at<>expected.created_at
LIMIT 1;

-- 身份全部匹配后，只补齐缺失 ID；不使用 INSERT IGNORE 掩盖固定 ID/自然键碰撞。
INSERT INTO sys_dept
SELECT expected.* FROM demo_fixture_sys_dept expected
LEFT JOIN sys_dept actual ON actual.id=expected.id
WHERE actual.id IS NULL;

INSERT INTO sys_user
SELECT expected.* FROM demo_fixture_sys_user expected
LEFT JOIN sys_user actual ON actual.id=expected.id
WHERE actual.id IS NULL;

INSERT INTO sys_role
SELECT expected.* FROM demo_fixture_sys_role expected
LEFT JOIN sys_role actual ON actual.id=expected.id
WHERE actual.id IS NULL;

INSERT INTO sys_menu
SELECT expected.* FROM demo_fixture_sys_menu expected
LEFT JOIN sys_menu actual ON actual.id=expected.id
WHERE actual.id IS NULL;

INSERT INTO sys_role_menu
SELECT expected.* FROM demo_fixture_sys_role_menu expected
LEFT JOIN sys_role_menu actual ON actual.id=expected.id
WHERE actual.id IS NULL;

INSERT INTO sys_api
SELECT expected.* FROM demo_fixture_sys_api expected
LEFT JOIN sys_api actual ON actual.id=expected.id
WHERE actual.id IS NULL;

INSERT INTO sys_dict
SELECT expected.* FROM demo_fixture_sys_dict expected
LEFT JOIN sys_dict actual ON actual.id=expected.id
WHERE actual.id IS NULL;

INSERT INTO casbin_rule
SELECT expected.* FROM demo_fixture_casbin_rule expected
LEFT JOIN casbin_rule actual ON actual.id=expected.id
WHERE actual.id IS NULL;

INSERT INTO sys_audit_log
SELECT expected.* FROM demo_fixture_sys_audit_log expected
LEFT JOIN sys_audit_log actual ON actual.id=expected.id
WHERE actual.id IS NULL;

DROP TEMPORARY TABLE demo_fixture_sys_dept, demo_fixture_sys_user, demo_fixture_sys_role, demo_fixture_sys_menu, demo_fixture_sys_role_menu, demo_fixture_sys_api, demo_fixture_sys_dict, demo_fixture_casbin_rule, demo_fixture_sys_audit_log, demo_fixture_checks;
COMMIT;

-- 只输出统计和公开展示字段，绝不输出密码哈希；range_rows 只是区间计数，不代表 fixture 归属。
SELECT 'sys_dept' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM sys_dept
UNION ALL
SELECT 'sys_user' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM sys_user
UNION ALL
SELECT 'sys_role' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM sys_role
UNION ALL
SELECT 'sys_menu' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM sys_menu
UNION ALL
SELECT 'sys_role_menu' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM sys_role_menu
UNION ALL
SELECT 'sys_api' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM sys_api
UNION ALL
SELECT 'sys_dict' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM sys_dict
UNION ALL
SELECT 'casbin_rule' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM casbin_rule
UNION ALL
SELECT 'sys_audit_log' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM sys_audit_log
UNION ALL
SELECT 'sys_file' AS table_name,COUNT(*) AS total_rows,COALESCE(SUM(id BETWEEN 9001000 AND 9001999),0) AS range_rows FROM sys_file;
SELECT id,name,parent_id,status FROM sys_dept WHERE id BETWEEN 9001000 AND 9001004 ORDER BY id;
SELECT username,nickname,status FROM sys_user WHERE id BETWEEN 9001100 AND 9001127 ORDER BY id;
