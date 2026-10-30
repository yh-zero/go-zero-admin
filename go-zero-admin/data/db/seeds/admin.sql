-- go-zero-admin 种子数据：初始管理员 + 基础角色/部门/casbin 策略
-- 由栈内 migrate job 在 goose up 之后执行；仅空库初始化时插入

-- ============================================
-- 1. 根部门
-- ============================================
INSERT INTO sys_dept (id, parent_id, name, leader, sort, status, path) VALUES
(1, 0, '总部', '', 0, 1, '/1/');

-- ============================================
-- 2. 初始超管用户（密码: Admin@123456）
-- ============================================
-- bcrypt hash of Admin@123456 (cost=10)
INSERT INTO sys_user (id, username, password_hash, nickname, real_name, dept_id, status) VALUES
(1, 'admin', '$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6', '超级管理员', 'Admin', 1, 1);

-- ============================================
-- 3. 基础角色
-- ============================================
INSERT INTO sys_role (id, name, code, description, data_scope, sort, status) VALUES
(1, '超级管理员', 'super_admin', '拥有所有权限，全局域', 'ALL', 0, 1),
(2, '部门管理员', 'dept_admin', '管理本部门及下级', 'DEPT_AND_CHILDREN', 1, 1),
(3, '普通员工', 'employee', '仅本人数据', 'SELF', 2, 1);

-- ============================================
-- 4. Casbin 策略
-- ============================================
-- p: super_admin 全局放行（dom=*，obj=*，act=*）
-- p: dept_admin 本部门用户/部门查询权限
-- p: employee 仅个人资料
-- g: admin 用户 → super_admin 角色（全局）
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES ('p', 'super_admin', '*', '*', '*');
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES ('p', 'dept_admin', 'dept:1', '/v1/sys/user/*', 'GET');
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES ('p', 'dept_admin', 'dept:1', '/v1/sys/dept/*', 'GET');
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES ('p', 'employee', 'dept:1', '/v1/sys/profile/*', 'GET');
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES ('g', 'user:1', 'super_admin', '*', '');
