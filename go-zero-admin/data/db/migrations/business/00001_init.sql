-- +goose Up
-- go-zero-admin 初始化：核心 RBAC + 组织 + 字典 + 文件 + 审计 + App 模块（10 张表）
-- 每租户独立库；本文件由栈内 migrate job 在首次启动时执行

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================
-- 1. 部门（物化路径树）
-- ============================================
CREATE TABLE sys_dept (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  parent_id   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父部门ID，0=根',
  name        VARCHAR(64)  NOT NULL COMMENT '部门名称',
  leader      VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '负责人',
  phone       VARCHAR(20)  NOT NULL DEFAULT '' COMMENT '联系电话',
  email       VARCHAR(128) NOT NULL DEFAULT '' COMMENT '邮箱',
  sort        INT NOT NULL DEFAULT 0 COMMENT '排序（小在前）',
  status      TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 2停用',
  path        VARCHAR(512) NOT NULL COMMENT '物化路径，如 /1/7/12/，根为 /1/',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at  DATETIME NULL DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (id),
  KEY idx_parent (parent_id),
  KEY idx_path (path(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='部门';

-- ============================================
-- 2. 后台用户（单部门）
-- ============================================
CREATE TABLE sys_user (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  username      VARCHAR(64)  NOT NULL COMMENT '登录名',
  password_hash VARCHAR(128) NOT NULL COMMENT '密码哈希（bcrypt）',
  nickname      VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '昵称',
  real_name     VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '真实姓名',
  email         VARCHAR(128) NOT NULL DEFAULT '' COMMENT '邮箱',
  phone         VARCHAR(20)  NOT NULL DEFAULT '' COMMENT '手机号',
  dept_id       BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '所属部门',
  avatar_file_id BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '头像文件ID → sys_file',
  gender        TINYINT NOT NULL DEFAULT 0 COMMENT '性别：0未知 1男 2女',
  status        TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 2停用',
  remark        VARCHAR(512) NOT NULL DEFAULT '' COMMENT '备注',
  last_login_at DATETIME NULL DEFAULT NULL COMMENT '最后登录时间',
  last_login_ip VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '最后登录IP',
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at    DATETIME NULL DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (id),
  UNIQUE KEY uk_username (username),
  KEY idx_dept (dept_id),
  KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='后台用户';

-- ============================================
-- 3. 角色（含数据范围）
-- ============================================
CREATE TABLE sys_role (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  name        VARCHAR(64)  NOT NULL COMMENT '角色名称',
  code        VARCHAR(64)  NOT NULL COMMENT '角色编码（英文唯一，casbin 主体用）',
  description VARCHAR(512) NOT NULL DEFAULT '' COMMENT '描述',
  data_scope  VARCHAR(32) NOT NULL DEFAULT 'SELF' COMMENT '数据范围：ALL/DEPT_AND_CHILDREN/DEPT/SELF',
  sort        INT NOT NULL DEFAULT 0 COMMENT '排序',
  status      TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 2停用',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at  DATETIME NULL DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (id),
  UNIQUE KEY uk_code (code),
  KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色';

-- ============================================
-- 4. 菜单（目录/菜单/按钮）
-- ============================================
CREATE TABLE sys_menu (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  parent_id  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父菜单ID，0=根',
  title      VARCHAR(64)  NOT NULL COMMENT '显示名称',
  name       VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '路由名（英文，前端用）',
  type       TINYINT NOT NULL COMMENT '类型：1目录 2菜单 3按钮',
  path       VARCHAR(255) NOT NULL DEFAULT '' COMMENT '路由路径',
  component  VARCHAR(255) NOT NULL DEFAULT '' COMMENT '前端组件路径',
  perm_code  VARCHAR(128) NOT NULL DEFAULT '' COMMENT '按钮权限码（如 sys:user:add）',
  icon       VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '图标',
  sort       INT NOT NULL DEFAULT 0 COMMENT '排序',
  visible    TINYINT NOT NULL DEFAULT 1 COMMENT '是否显示：1显示 0隐藏',
  status     TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 2停用',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME NULL DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (id),
  KEY idx_parent (parent_id),
  KEY idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单';

-- ============================================
-- 5. 角色-菜单（按钮权限通过菜单表的 perm_code 实现）
-- ============================================
CREATE TABLE sys_role_menu (
  id      BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  role_id BIGINT UNSIGNED NOT NULL,
  menu_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_role_menu (role_id, menu_id),
  KEY idx_menu (menu_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色-菜单';

-- ============================================
-- 6. API 资源（接口目录，casbin 的 obj 来源）
-- ============================================
CREATE TABLE sys_api (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  group_name VARCHAR(64)  NOT NULL COMMENT '分组（如 sys/user）',
  method     VARCHAR(8)   NOT NULL COMMENT 'HTTP 方法：GET/POST/PUT/DELETE',
  path       VARCHAR(255) NOT NULL COMMENT '路径，如 /v1/sys/user/:id',
  title      VARCHAR(128) NOT NULL DEFAULT '' COMMENT '接口说明',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_method_path (method, path)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='API 资源';

-- ============================================
-- 7. Casbin 策略（p + g 同表；v1=dom 装部门，*=全局）
-- ============================================
CREATE TABLE casbin_rule (
  id    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  ptype VARCHAR(8)   NOT NULL COMMENT 'p=策略 g=角色分组',
  v0    VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'p: sub / g: user',
  v1    VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'p: dom / g: role',
  v2    VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'p: obj / g: dom',
  v3    VARCHAR(64)  NOT NULL DEFAULT '' COMMENT 'p: act',
  v4    VARCHAR(64)  NOT NULL DEFAULT '',
  v5    VARCHAR(64)  NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_ptype_v0_v1_v2_v3 (ptype, v0, v1, v2, v3),
  KEY idx_v0 (v0),
  KEY idx_v1 (v1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Casbin 策略';

-- ============================================
-- 8. 字典（字典项合并为 JSON 数组）
-- ============================================
CREATE TABLE sys_dict (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name        VARCHAR(64)  NOT NULL COMMENT '字典名称',
  code        VARCHAR(64)  NOT NULL COMMENT '字典编码（英文唯一）',
  items       JSON NOT NULL COMMENT '[{"label":"启用","value":1},{"label":"停用","value":2}]',
  description VARCHAR(512) NOT NULL DEFAULT '' COMMENT '描述',
  status      TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 2停用',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at  DATETIME NULL DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (id),
  UNIQUE KEY uk_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='字典';

-- ============================================
-- 9. 文件资源（RustFS）
-- ============================================
CREATE TABLE sys_file (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  bucket      VARCHAR(64)  NOT NULL COMMENT 'bucket 名',
  object_key  VARCHAR(512) NOT NULL COMMENT '对象键（存储路径）',
  original_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '原始文件名',
  ext         VARCHAR(16)  NOT NULL DEFAULT '' COMMENT '扩展名',
  mime        VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'MIME 类型',
  size        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '字节数',
  visibility  VARCHAR(16) NOT NULL DEFAULT 'public' COMMENT '可见性：public/private',
  ref_count   INT NOT NULL DEFAULT 0 COMMENT '引用计数（0 可删）',
  uploader_id BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '上传者用户ID',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at  DATETIME NULL DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (id),
  KEY idx_bucket_key (bucket, object_key(191)),
  KEY idx_uploader (uploader_id),
  KEY idx_ref_count (ref_count)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件资源';

-- ============================================
-- 10. 审计日志
-- ============================================
CREATE TABLE sys_audit_log (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  user_id     BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '操作人ID',
  username    VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '操作人用户名',
  module      VARCHAR(64)  NOT NULL COMMENT '模块（如 sys/user）',
  action      VARCHAR(64)  NOT NULL COMMENT '操作（如 create/update/delete/login）',
  method      VARCHAR(8)   NOT NULL DEFAULT '' COMMENT 'HTTP 方法',
  path        VARCHAR(255) NOT NULL DEFAULT '' COMMENT '请求路径',
  params      TEXT NULL COMMENT '请求参数（JSON，敏感字段脱敏）',
  result      TINYINT NOT NULL DEFAULT 1 COMMENT '结果：1成功 2失败',
  error_msg   VARCHAR(512) NOT NULL DEFAULT '' COMMENT '错误信息',
  ip          VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '客户端IP',
  user_agent  VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  duration_ms INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '耗时毫秒',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_user (user_id),
  KEY idx_module_action (module, action),
  KEY idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='审计日志';

-- ============================================
-- 11. App 用户
-- ============================================
CREATE TABLE app_user (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  account       VARCHAR(128) NOT NULL COMMENT '账号（邮箱或手机号，小写）',
  identity_kind VARCHAR(16)  NOT NULL COMMENT '账号类型：email/phone',
  password_hash VARCHAR(128) NOT NULL COMMENT '密码哈希（bcrypt）',
  nickname      VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '昵称',
  avatar_file_id BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '头像文件ID',
  status        VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/active/disabled/rejected',
  review_reason VARCHAR(512) NOT NULL DEFAULT '' COMMENT '审核备注',
  email_verified_at DATETIME NULL DEFAULT NULL COMMENT '邮箱验证时间',
  phone_verified_at DATETIME NULL DEFAULT NULL COMMENT '手机验证时间',
  last_login_at DATETIME NULL DEFAULT NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at    DATETIME NULL DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (id),
  UNIQUE KEY uk_account (account),
  KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 用户';

-- ============================================
-- 12. App 会员授予（plan 用常量枚举：vip 等）
-- ============================================
CREATE TABLE app_membership_grant (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  user_id     BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  plan_code   VARCHAR(32)  NOT NULL COMMENT '方案编码（常量：vip）',
  benefits    VARCHAR(512) NOT NULL DEFAULT '' COMMENT '权益快照（逗号分隔，如 profile.export）',
  request_id  VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '幂等请求ID（续期重试复用）',
  granted_by  BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '操作管理员ID（自助开通为 NULL）',
  starts_at   DATETIME NOT NULL COMMENT '生效时间',
  expires_at  DATETIME NOT NULL COMMENT '过期时间',
  revoked     TINYINT NOT NULL DEFAULT 0 COMMENT '是否已撤销',
  revoked_at  DATETIME NULL DEFAULT NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_request (request_id),
  KEY idx_user (user_id),
  KEY idx_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 会员授予';

SET FOREIGN_KEY_CHECKS = 1;

-- +goose Down
DROP TABLE IF EXISTS app_membership_grant;
DROP TABLE IF EXISTS app_user;
DROP TABLE IF EXISTS sys_audit_log;
DROP TABLE IF EXISTS sys_file;
DROP TABLE IF EXISTS sys_dict;
DROP TABLE IF EXISTS casbin_rule;
DROP TABLE IF EXISTS sys_api;
DROP TABLE IF EXISTS sys_role_menu;
DROP TABLE IF EXISTS sys_menu;
DROP TABLE IF EXISTS sys_role;
DROP TABLE IF EXISTS sys_user;
DROP TABLE IF EXISTS sys_dept;
