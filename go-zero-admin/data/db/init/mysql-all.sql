-- go-zero-admin 开源初始化：单库完整结构 + 虚构演示数据
-- 仅用于新建或重置开发数据库：本文件会 DROP 并重建列出的 20 张表。
-- 首次 docker compose up -d 时，MySQL 空数据卷自动导入；已有卷不会重复执行。
-- 手动导入：mysql -h 127.0.0.1 -P 3307 -u root -p < mysql-all.sql
-- 后台账号：admin、demo_all、demo_tree、demo_dept、demo_self（密码均为 Admin@123456）。
-- APP 示例账号：alice@example.test / Admin@123456；example.test 均为保留演示域，不接收邮件。
-- 默认开启图片验证码；请在登录页按提示填写。演示密码仅限本地开发，部署前必须修改。
-- 必备数据：部门、角色、菜单/操作权限、API 资源、字典及授权关联。
-- 演示数据：后台用户、APP 用户、会员和反馈；文件/审计/AI/租户登记初始为空。
-- 不包含模型密钥、邮箱授权码、真实个人资料、对象存储文件或租户运行凭据。
-- 租户导航独立于登记数据，顶层与 APP 管理同级；真实租户通过平台创建。

SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT;
SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS;
SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION;
SET NAMES utf8mb4;
SET @OLD_TIME_ZONE=@@TIME_ZONE;
SET TIME_ZONE='+00:00';
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS;
SET FOREIGN_KEY_CHECKS=0;
SET @OLD_SQL_MODE=@@SQL_MODE;
SET SQL_MODE='NO_AUTO_VALUE_ON_ZERO,STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION';

CREATE DATABASE IF NOT EXISTS `gozero_admin` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
USE `gozero_admin`;

-- ai_conversation
DROP TABLE IF EXISTS `ai_conversation`;
CREATE TABLE `ai_conversation` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `user_id` bigint unsigned NOT NULL COMMENT '所属用户',
  `title` varchar(128) NOT NULL DEFAULT '' COMMENT '会话标题',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 会话';

-- ai_message
DROP TABLE IF EXISTS `ai_message`;
CREATE TABLE `ai_message` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `conversation_id` bigint unsigned NOT NULL COMMENT '会话ID',
  `role` varchar(16) NOT NULL COMMENT '角色：user/assistant/tool/system',
  `content` text NOT NULL COMMENT '消息内容',
  `tool_calls` json DEFAULT NULL COMMENT '调用元数据，兼容历史工具调用数组',
  `tokens` int unsigned NOT NULL DEFAULT '0' COMMENT '消耗 token 数',
  `status` varchar(16) NOT NULL DEFAULT 'succeeded' COMMENT '调用状态：running/succeeded/failed',
  `call_id` varchar(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL COMMENT '用户命名空间内的幂等调用标识，历史消息为空',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_call_role` (`call_id`,`role`),
  KEY `idx_conversation` (`conversation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 消息';

-- sys_menu_lock：菜单结构写入的专用事务锁，不与用户编辑共享锁。
DROP TABLE IF EXISTS `sys_menu_lock`;
CREATE TABLE `sys_menu_lock` (
  `id` bigint unsigned NOT NULL COMMENT '固定锁记录，必须为1',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单结构事务锁';
INSERT INTO `sys_menu_lock` (`id`) VALUES (1);

-- ai_task
DROP TABLE IF EXISTS `ai_task`;
CREATE TABLE `ai_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `workflow_id` bigint unsigned DEFAULT NULL COMMENT '工作流ID（对话触发的任务为空）',
  `conversation_id` bigint unsigned DEFAULT NULL COMMENT '会话ID',
  `status` varchar(24) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/running/pending_approval/succeeded/failed/rejected',
  `input` text COMMENT '输入（JSON）',
  `result` text COMMENT '执行结果',
  `error` varchar(512) NOT NULL DEFAULT '' COMMENT '错误信息',
  `approved_by` bigint unsigned DEFAULT NULL COMMENT '审批人ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_at` datetime DEFAULT NULL COMMENT '完成时间',
  `created_by` bigint unsigned NOT NULL DEFAULT '0' COMMENT '任务所属用户，0表示历史归属待确认',
  `next_step` int unsigned NOT NULL DEFAULT '0' COMMENT '下一待执行步骤索引',
  `workflow_snapshot` json DEFAULT NULL COMMENT '执行时工作流定义快照',
  PRIMARY KEY (`id`),
  KEY `idx_workflow` (`workflow_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 任务';

-- ai_workflow
DROP TABLE IF EXISTS `ai_workflow`;
CREATE TABLE `ai_workflow` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `name` varchar(128) NOT NULL COMMENT '工作流名称',
  `definition` json NOT NULL COMMENT 'JSON DSL：{"steps":[{"name","tool","input","approval"}]}',
  `interval_seconds` int unsigned NOT NULL DEFAULT '0' COMMENT '调度间隔秒数，0=仅手动触发',
  `enabled` tinyint NOT NULL DEFAULT '1' COMMENT '是否启用',
  `next_run_at` datetime DEFAULT NULL COMMENT '下次执行时间',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者用户ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_next_run` (`enabled`,`next_run_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 工作流';

-- app_membership_grant
DROP TABLE IF EXISTS `app_membership_grant`;
CREATE TABLE `app_membership_grant` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `plan_code` varchar(32) NOT NULL COMMENT '方案编码（常量：vip）',
  `benefits` varchar(512) NOT NULL DEFAULT '' COMMENT '权益快照（逗号分隔，如 profile.export）',
  `request_id` varchar(64) NOT NULL DEFAULT '' COMMENT '幂等请求ID（续期重试复用）',
  `granted_by` bigint unsigned DEFAULT NULL COMMENT '操作管理员ID（自助开通为 NULL）',
  `starts_at` datetime NOT NULL COMMENT '生效时间',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `revoked` tinyint NOT NULL DEFAULT '0' COMMENT '是否已撤销',
  `revoked_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_request` (`request_id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_expires` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 会员授予';

-- app_feedback
DROP TABLE IF EXISTS `app_feedback`;
CREATE TABLE `app_feedback` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `user_id` bigint unsigned NOT NULL COMMENT '提交用户',
  `content` text NOT NULL COMMENT '纯文字反馈内容',
  `reply` text NOT NULL COMMENT '后台回复，空字符串为未回复',
  `status` varchar(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/resolved',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`,`id`),
  KEY `idx_status_id` (`status`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 意见反馈';

-- app_user
DROP TABLE IF EXISTS `app_user`;
CREATE TABLE `app_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `account` varchar(128) NOT NULL COMMENT '账号（邮箱或手机号，小写）',
  `identity_kind` varchar(16) NOT NULL COMMENT '账号类型：email/phone',
  `password_hash` varchar(128) NOT NULL COMMENT '密码哈希（bcrypt）',
  `nickname` varchar(64) NOT NULL DEFAULT '' COMMENT '昵称',
  `remark` varchar(500) NOT NULL DEFAULT '' COMMENT '仅后台可见的内部备注',
  `avatar_file_id` bigint unsigned DEFAULT NULL COMMENT '头像文件ID',
  `status` varchar(16) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/active/disabled/rejected',
  `review_reason` varchar(512) NOT NULL DEFAULT '' COMMENT '审核备注',
  `email_verified_at` datetime DEFAULT NULL COMMENT '邮箱验证时间',
  `phone_verified_at` datetime DEFAULT NULL COMMENT '手机验证时间',
  `last_login_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account` (`account`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 用户';

-- casbin_rule
DROP TABLE IF EXISTS `casbin_rule`;
CREATE TABLE `casbin_rule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ptype` varchar(8) NOT NULL COMMENT 'p=策略 g=角色分组',
  `v0` varchar(128) NOT NULL DEFAULT '' COMMENT 'p: sub / g: user',
  `v1` varchar(128) NOT NULL DEFAULT '' COMMENT 'p: dom / g: role',
  `v2` varchar(255) NOT NULL DEFAULT '' COMMENT 'p: obj / g: dom',
  `v3` varchar(64) NOT NULL DEFAULT '' COMMENT 'p: act',
  `v4` varchar(64) NOT NULL DEFAULT '',
  `v5` varchar(64) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ptype_v0_v1_v2_v3` (`ptype`,`v0`,`v1`,`v2`,`v3`),
  KEY `idx_v0` (`v0`),
  KEY `idx_v1` (`v1`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Casbin 策略';

-- sys_api
DROP TABLE IF EXISTS `sys_api`;
CREATE TABLE `sys_api` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `group_name` varchar(64) NOT NULL COMMENT '分组（如 sys/user）',
  `method` varchar(8) NOT NULL COMMENT 'HTTP 方法：GET/POST/PUT/DELETE',
  `path` varchar(255) NOT NULL COMMENT '路径，如 /v1/sys/user/:id',
  `title` varchar(128) NOT NULL DEFAULT '' COMMENT '接口说明',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_method_path` (`method`,`path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='API 资源';

-- sys_audit_log
DROP TABLE IF EXISTS `sys_audit_log`;
CREATE TABLE `sys_audit_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `user_id` bigint unsigned DEFAULT NULL COMMENT '操作人ID',
  `username` varchar(64) NOT NULL DEFAULT '' COMMENT '操作人用户名',
  `module` varchar(64) NOT NULL COMMENT '模块（如 sys/user）',
  `action` varchar(64) NOT NULL COMMENT '操作（如 create/update/delete/login）',
  `method` varchar(8) NOT NULL DEFAULT '' COMMENT 'HTTP 方法',
  `path` varchar(255) NOT NULL DEFAULT '' COMMENT '请求路径',
  `params` text COMMENT '请求参数（JSON，敏感字段脱敏）',
  `result` tinyint NOT NULL DEFAULT '1' COMMENT '结果：1成功 2失败',
  `error_msg` varchar(512) NOT NULL DEFAULT '' COMMENT '错误信息',
  `ip` varchar(64) NOT NULL DEFAULT '' COMMENT '客户端IP',
  `user_agent` varchar(512) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  `duration_ms` int unsigned NOT NULL DEFAULT '0' COMMENT '耗时毫秒',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_module_action` (`module`,`action`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='审计日志';

-- sys_dept
DROP TABLE IF EXISTS `sys_dept`;
CREATE TABLE `sys_dept` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '父部门ID，0=根',
  `name` varchar(64) NOT NULL COMMENT '部门名称',
  `leader` varchar(64) NOT NULL DEFAULT '' COMMENT '负责人',
  `leader_user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '负责人账号ID，0=未关联',
  `phone` varchar(20) NOT NULL DEFAULT '' COMMENT '联系电话',
  `email` varchar(128) NOT NULL DEFAULT '' COMMENT '邮箱',
  `sort` int NOT NULL DEFAULT '0' COMMENT '排序（小在前）',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `path` varchar(512) NOT NULL COMMENT '物化路径，如 /1/7/12/，根为 /1/',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_parent` (`parent_id`),
  KEY `idx_path` (`path`(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='部门';

-- sys_dict
DROP TABLE IF EXISTS `sys_dict`;
CREATE TABLE `sys_dict` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL COMMENT '字典名称',
  `code` varchar(64) NOT NULL COMMENT '字典编码（英文唯一）',
  `items` json NOT NULL COMMENT '[{"label":"启用","value":1},{"label":"停用","value":2}]',
  `description` varchar(512) NOT NULL DEFAULT '' COMMENT '描述',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='字典';

-- sys_file
DROP TABLE IF EXISTS `sys_file`;
CREATE TABLE `sys_file` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `bucket` varchar(64) NOT NULL COMMENT 'bucket 名',
  `object_key` varchar(512) NOT NULL COMMENT '对象键（存储路径）',
  `original_name` varchar(255) NOT NULL DEFAULT '' COMMENT '原始文件名',
  `ext` varchar(16) NOT NULL DEFAULT '' COMMENT '扩展名',
  `mime` varchar(128) NOT NULL DEFAULT '' COMMENT 'MIME 类型',
  `size` bigint unsigned NOT NULL DEFAULT '0' COMMENT '字节数',
  `visibility` varchar(16) NOT NULL DEFAULT 'public' COMMENT '可见性：public/private',
  `ref_count` int NOT NULL DEFAULT '0' COMMENT '引用计数（0 可删）',
  `uploader_id` bigint unsigned DEFAULT NULL COMMENT '上传者用户ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_bucket_key` (`bucket`,`object_key`(191)),
  KEY `idx_uploader` (`uploader_id`),
  KEY `idx_ref_count` (`ref_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件资源';

-- sys_file_reference
DROP TABLE IF EXISTS `sys_file_reference`;
CREATE TABLE `sys_file_reference` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `file_id` bigint unsigned NOT NULL COMMENT '文件ID',
  `ref_type` varchar(32) NOT NULL COMMENT '业务引用类型：admin_avatar/app_avatar',
  `ref_id` bigint unsigned NOT NULL COMMENT '业务对象ID',
  `created_by` bigint unsigned NOT NULL COMMENT '建立引用的操作者ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_file_ref` (`file_id`,`ref_type`,`ref_id`),
  KEY `idx_ref_subject` (`ref_type`,`ref_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件业务引用明细';

-- sys_menu
DROP TABLE IF EXISTS `sys_menu`;
CREATE TABLE `sys_menu` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '父菜单ID，0=根',
  `title` varchar(64) NOT NULL COMMENT '显示名称',
  `name` varchar(64) NOT NULL DEFAULT '' COMMENT '路由名（英文，前端用）',
  `type` tinyint NOT NULL COMMENT '类型：1目录 2菜单 3按钮',
  `path` varchar(255) NOT NULL DEFAULT '' COMMENT '路由路径',
  `component` varchar(255) NOT NULL DEFAULT '' COMMENT '前端组件路径',
  `perm_code` varchar(128) NOT NULL DEFAULT '' COMMENT '按钮权限码（如 sys:user:add）',
  `icon` varchar(64) NOT NULL DEFAULT '' COMMENT '图标',
  `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
  `visible` tinyint NOT NULL DEFAULT '1' COMMENT '是否显示：1显示 0隐藏',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_parent` (`parent_id`),
  KEY `idx_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单';

-- sys_role
DROP TABLE IF EXISTS `sys_role`;
CREATE TABLE `sys_role` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `name` varchar(64) NOT NULL COMMENT '角色名称',
  `code` varchar(64) NOT NULL COMMENT '角色编码（英文唯一，casbin 主体用）',
  `description` varchar(512) NOT NULL DEFAULT '' COMMENT '描述',
  `data_scope` varchar(32) NOT NULL DEFAULT 'SELF' COMMENT '数据范围：ALL/DEPT_AND_CHILDREN/DEPT/SELF',
  `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色';

-- sys_role_menu
DROP TABLE IF EXISTS `sys_role_menu`;
CREATE TABLE `sys_role_menu` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `role_id` bigint unsigned NOT NULL,
  `menu_id` bigint unsigned NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_menu` (`role_id`,`menu_id`),
  KEY `idx_menu` (`menu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色-菜单';

-- sys_user
DROP TABLE IF EXISTS `sys_user`;
CREATE TABLE `sys_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `username` varchar(64) NOT NULL COMMENT '登录名',
  `password_hash` varchar(128) NOT NULL COMMENT '密码哈希（bcrypt）',
  `nickname` varchar(64) NOT NULL DEFAULT '' COMMENT '昵称',
  `real_name` varchar(64) NOT NULL DEFAULT '' COMMENT '真实姓名',
  `email` varchar(128) NOT NULL DEFAULT '' COMMENT '邮箱',
  `phone` varchar(20) NOT NULL DEFAULT '' COMMENT '手机号',
  `dept_id` bigint unsigned DEFAULT NULL COMMENT '所属部门',
  `avatar_file_id` bigint unsigned DEFAULT NULL COMMENT '头像文件ID → sys_file',
  `gender` tinyint NOT NULL DEFAULT '0' COMMENT '性别：0未知 1男 2女',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `remark` varchar(512) NOT NULL DEFAULT '' COMMENT '备注',
  `last_login_at` datetime DEFAULT NULL COMMENT '最后登录时间',
  `last_login_ip` varchar(64) NOT NULL DEFAULT '' COMMENT '最后登录IP',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_dept` (`dept_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='后台用户';

-- tenant_registry
DROP TABLE IF EXISTS `tenant_registry`;
CREATE TABLE `tenant_registry` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `slug` varchar(32) NOT NULL COMMENT '租户标识（英文小写，子域名前缀）',
  `domain` varchar(128) NOT NULL COMMENT '完整域名，如 t1.localhost',
  `compose_project` varchar(64) NOT NULL COMMENT 'compose 项目名，如 gozero-t-t1',
  `status` varchar(16) NOT NULL DEFAULT 'active' COMMENT '状态：active/stopped/error',
  `mysql_root_password` varchar(128) NOT NULL COMMENT '该租户 MySQL root 密码（随机生成）',
  `jwt_access_secret` varchar(128) NOT NULL COMMENT '该租户后台 JWT 密钥（随机生成）',
  `app_jwt_access_secret` varchar(128) NOT NULL COMMENT '该租户 App JWT 密钥（随机生成）',
  `admin_account_ready` tinyint NOT NULL DEFAULT '0' COMMENT '初始管理员是否就绪',
  `error_msg` varchar(512) NOT NULL DEFAULT '' COMMENT '最近一次错误信息',
  `current_step` varchar(32) DEFAULT '' COMMENT 'provisioning 当前步骤',
  `step_updated_at` datetime DEFAULT NULL COMMENT '步骤更新时间',
  `created_by` bigint unsigned DEFAULT NULL COMMENT '创建操作员ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除（注销租户）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_slug` (`slug`),
  UNIQUE KEY `uk_domain` (`domain`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户注册表';

-- 部门层级与负责人使用同批虚构账号，路径和启停状态保持一致。
INSERT INTO `sys_dept` (`id`,`parent_id`,`name`,`leader`,`leader_user_id`,`email`,`sort`,`status`,`path`,`created_at`,`updated_at`) VALUES
(1,0,'总部','',0,'',0,1,'/1/','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(2,1,'研发中心','研发负责人',3,'rd@example.test',10,1,'/1/2/','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(3,2,'平台研发组','平台负责人',4,'platform@example.test',10,1,'/1/2/3/','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(4,2,'质量保障组','',0,'qa@example.test',20,1,'/1/2/4/','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(5,1,'运营中心','运营负责人',7,'operations@example.test',20,1,'/1/5/','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(6,1,'归档示例部门','',0,'',99,2,'/1/6/','2026-01-01 00:00:00','2026-01-01 00:00:00');

INSERT INTO `sys_role` (`id`,`name`,`code`,`description`,`data_scope`,`sort`,`status`,`created_at`,`updated_at`) VALUES
(1,'超级管理员','super_admin','平台全权限；仅供本地开发初始化','ALL',0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(2,'全部用户查看','scope_all','只读用户和部门，数据范围为全部用户','ALL',10,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(3,'部门及下级查看','scope_dept_children','只读用户和部门，覆盖当前部门及下级','DEPT_AND_CHILDREN',20,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(4,'本部门查看','scope_dept','只读用户和部门，仅当前部门用户','DEPT',30,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(5,'本人查看','scope_self','只读用户和部门，仅本人用户数据','SELF',40,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(6,'停用示例角色','disabled_example','停用角色无用户或菜单关联','SELF',99,2,'2026-01-01 00:00:00','2026-01-01 00:00:00');

INSERT INTO `sys_user` (`id`,`username`,`password_hash`,`nickname`,`real_name`,`email`,`dept_id`,`gender`,`status`,`remark`,`created_at`,`updated_at`) VALUES
(1,'admin','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','超级管理员','','',1,0,1,'本地开发默认账号，部署前请修改密码','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(2,'demo_all','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','全局查看员','演示成员甲','all@example.test',2,0,1,'ALL 数据范围示例，只读业务列表','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(3,'demo_tree','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','研发负责人','演示成员乙','tree@example.test',2,0,1,'研发中心为本部门及下级；切换运营中心时仅本人','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(4,'demo_dept','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','平台负责人','演示成员丙','dept@example.test',3,0,1,'DEPT 数据范围示例','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(5,'demo_self','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','平台成员','演示成员丁','self@example.test',3,0,1,'SELF 数据范围示例','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(6,'demo_quality','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','质量成员','演示成员戊','quality@example.test',4,0,1,'研发下级部门成员，用于比较查询范围','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(7,'demo_operations','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','运营负责人','演示成员己','operations@example.test',5,0,1,'其他部门成员，用于比较查询范围','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(8,'demo_disabled','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','停用示例用户','演示成员庚','disabled@example.test',5,0,2,'停用账号不可登录','2026-01-01 00:00:00','2026-01-01 00:00:00');

-- 系统管理、APP 管理、租户管理和 AI 功能；按钮权限挂在所属页面下。
INSERT INTO `sys_menu` (`id`,`parent_id`,`title`,`name`,`type`,`path`,`component`,`perm_code`,`icon`,`sort`,`visible`,`status`,`created_at`,`updated_at`) VALUES
(9001300,0,'系统管理','system',1,'/sys','','','system',10,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001301,0,'工作台','dashboard',2,'/dashboard','dashboard','','dashboard',0,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001302,9001300,'用户管理','user',2,'/sys/user','user','','user',20,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001303,9001300,'角色管理','role',2,'/sys/role','role','','role',21,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001304,9001300,'菜单管理','menu',2,'/sys/menu','menu','','menu',22,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001305,9001300,'部门管理','dept',2,'/sys/dept','dept','','dept',23,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001306,9001300,'字典管理','dict',2,'/sys/dict','dict','','dict',24,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001307,9001300,'API 资源','api',2,'/sys/api','api','','api',25,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001308,9001300,'文件管理','file',2,'/sys/file','file','','file',26,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001309,9001300,'审计日志','audit',2,'/sys/audit','audit','','audit',27,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001310,9001300,'个人中心','profile',2,'/sys/profile','profile','','profile',28,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001320,9001302,'新增用户','user_add',3,'','','sys:user:add','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001321,9001302,'编辑用户','user_edit',3,'','','sys:user:edit','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001322,9001302,'删除用户','user_delete',3,'','','sys:user:delete','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001323,9001302,'重置密码','user_reset_password',3,'','','sys:user:reset-password','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001324,9001303,'新增角色','role_add',3,'','','sys:role:add','',4,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001325,9001303,'编辑角色','role_edit',3,'','','sys:role:edit','',5,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001326,9001303,'删除角色','role_delete',3,'','','sys:role:delete','',6,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001327,9001303,'分配菜单','role_menus',3,'','','sys:role:menus','',7,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001328,9001304,'新增菜单','menu_add',3,'','','sys:menu:add','',8,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001329,9001304,'编辑菜单','menu_edit',3,'','','sys:menu:edit','',9,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001330,9001304,'删除菜单','menu_delete',3,'','','sys:menu:delete','',10,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001331,9001305,'新增部门','dept_add',3,'','','sys:dept:add','',11,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001332,9001305,'编辑部门','dept_edit',3,'','','sys:dept:edit','',12,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001333,9001305,'删除部门','dept_delete',3,'','','sys:dept:delete','',13,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001334,9001306,'新增字典','dict_add',3,'','','sys:dict:add','',14,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001335,9001306,'编辑字典','dict_edit',3,'','','sys:dict:edit','',15,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001336,9001306,'删除字典','dict_delete',3,'','','sys:dict:delete','',16,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001337,9001307,'删除 API','api_delete',3,'','','sys:api:delete','',17,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001338,9001308,'删除文件','file_delete',3,'','','sys:file:delete','',18,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001339,9001310,'编辑资料','profile_edit',3,'','','sys:profile:edit','',19,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001360,0,'APP管理','app',1,'/app','','','app',20,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001361,9001360,'APP用户','app_user',2,'/app/users','app_user','','app_user',0,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001362,9001360,'会员管理','app_membership',2,'/app/memberships','app_membership','','membership',1,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001363,9001361,'审核用户','app_user_review',3,'','','sys:app:user:review','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001364,9001361,'设置头像','app_user_avatar',3,'','','sys:app:user:avatar','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001365,9001361,'查看登录设备','app_session_list',3,'','','sys:app:session:list','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001366,9001361,'撤销登录设备','app_session_revoke',3,'','','sys:app:session:revoke','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001367,9001362,'授予会员','app_membership_grant',3,'','','sys:app:membership:grant','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001368,9001362,'撤销会员','app_membership_revoke',3,'','','sys:app:membership:revoke','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001369,9001308,'上传文件','file_upload',3,'','','sys:file:upload','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001370,9001361,'编辑资料','app_user_edit',3,'','','sys:app:user:edit','',4,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001371,9001360,'意见反馈','app_feedback',2,'/app/feedback','app_feedback','','message',2,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001372,9001371,'处理反馈','app_feedback_reply',3,'','','sys:app:feedback:reply','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001373,0,'租户管理','tenant',1,'/tenant','','','cloud-server',25,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001374,9001373,'租户列表','tenant_list',2,'/tenant/list','tenant_list','','cloud-server',0,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001375,9001374,'创建租户','tenant_create',3,'','','sys:tenant:create','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001376,9001374,'启停租户','tenant_lifecycle',3,'','','sys:tenant:lifecycle','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001377,9001374,'重置管理员密码','tenant_reset',3,'','','sys:tenant:reset','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001378,9001374,'删除租户','tenant_delete',3,'','','sys:tenant:delete','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001400,0,'AI 功能','ai',1,'/ai','','','robot',30,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001401,9001400,'AI 对话','ai_chat',2,'/ai/chat','ai_chat','','message',0,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001402,9001401,'新建会话','ai_conversation_create',3,'','','sys:ai:conversation:create','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001403,9001401,'发送消息','ai_message_send',3,'','','sys:ai:message:send','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001404,9001401,'删除会话','ai_conversation_delete',3,'','','sys:ai:conversation:delete','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9001405,9001401,'查看会话','ai_conversation_list',3,'','','sys:ai:conversation:list','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00');

-- 超级管理员获得全部菜单与操作权限；服务端仍独立鉴权。
INSERT INTO `sys_role_menu` (`role_id`,`menu_id`,`created_at`) VALUES
(1,9001300,'2026-01-01 00:00:00'),
(1,9001301,'2026-01-01 00:00:00'),
(1,9001302,'2026-01-01 00:00:00'),
(1,9001303,'2026-01-01 00:00:00'),
(1,9001304,'2026-01-01 00:00:00'),
(1,9001305,'2026-01-01 00:00:00'),
(1,9001306,'2026-01-01 00:00:00'),
(1,9001307,'2026-01-01 00:00:00'),
(1,9001308,'2026-01-01 00:00:00'),
(1,9001309,'2026-01-01 00:00:00'),
(1,9001310,'2026-01-01 00:00:00'),
(1,9001320,'2026-01-01 00:00:00'),
(1,9001321,'2026-01-01 00:00:00'),
(1,9001322,'2026-01-01 00:00:00'),
(1,9001323,'2026-01-01 00:00:00'),
(1,9001324,'2026-01-01 00:00:00'),
(1,9001325,'2026-01-01 00:00:00'),
(1,9001326,'2026-01-01 00:00:00'),
(1,9001327,'2026-01-01 00:00:00'),
(1,9001328,'2026-01-01 00:00:00'),
(1,9001329,'2026-01-01 00:00:00'),
(1,9001330,'2026-01-01 00:00:00'),
(1,9001331,'2026-01-01 00:00:00'),
(1,9001332,'2026-01-01 00:00:00'),
(1,9001333,'2026-01-01 00:00:00'),
(1,9001334,'2026-01-01 00:00:00'),
(1,9001335,'2026-01-01 00:00:00'),
(1,9001336,'2026-01-01 00:00:00'),
(1,9001337,'2026-01-01 00:00:00'),
(1,9001338,'2026-01-01 00:00:00'),
(1,9001339,'2026-01-01 00:00:00'),
(1,9001360,'2026-01-01 00:00:00'),
(1,9001361,'2026-01-01 00:00:00'),
(1,9001362,'2026-01-01 00:00:00'),
(1,9001363,'2026-01-01 00:00:00'),
(1,9001364,'2026-01-01 00:00:00'),
(1,9001365,'2026-01-01 00:00:00'),
(1,9001366,'2026-01-01 00:00:00'),
(1,9001367,'2026-01-01 00:00:00'),
(1,9001368,'2026-01-01 00:00:00'),
(1,9001369,'2026-01-01 00:00:00'),
(1,9001370,'2026-01-01 00:00:00'),
(1,9001371,'2026-01-01 00:00:00'),
(1,9001372,'2026-01-01 00:00:00'),
(1,9001373,'2026-01-01 00:00:00'),
(1,9001374,'2026-01-01 00:00:00'),
(1,9001375,'2026-01-01 00:00:00'),
(1,9001376,'2026-01-01 00:00:00'),
(1,9001377,'2026-01-01 00:00:00'),
(1,9001378,'2026-01-01 00:00:00'),
(1,9001400,'2026-01-01 00:00:00'),
(1,9001401,'2026-01-01 00:00:00'),
(1,9001402,'2026-01-01 00:00:00'),
(1,9001403,'2026-01-01 00:00:00'),
(1,9001404,'2026-01-01 00:00:00'),
(1,9001405,'2026-01-01 00:00:00');

-- 普通演示角色仅显示工作台、用户、部门和个人中心。
INSERT INTO `sys_role_menu` (`role_id`,`menu_id`,`created_at`) VALUES
(2,9001301,'2026-01-01 00:00:00'),
(2,9001300,'2026-01-01 00:00:00'),
(2,9001302,'2026-01-01 00:00:00'),
(2,9001305,'2026-01-01 00:00:00'),
(2,9001310,'2026-01-01 00:00:00'),
(2,9001339,'2026-01-01 00:00:00'),
(3,9001301,'2026-01-01 00:00:00'),
(3,9001300,'2026-01-01 00:00:00'),
(3,9001302,'2026-01-01 00:00:00'),
(3,9001305,'2026-01-01 00:00:00'),
(3,9001310,'2026-01-01 00:00:00'),
(3,9001339,'2026-01-01 00:00:00'),
(4,9001301,'2026-01-01 00:00:00'),
(4,9001300,'2026-01-01 00:00:00'),
(4,9001302,'2026-01-01 00:00:00'),
(4,9001305,'2026-01-01 00:00:00'),
(4,9001310,'2026-01-01 00:00:00'),
(4,9001339,'2026-01-01 00:00:00'),
(5,9001301,'2026-01-01 00:00:00'),
(5,9001300,'2026-01-01 00:00:00'),
(5,9001302,'2026-01-01 00:00:00'),
(5,9001305,'2026-01-01 00:00:00'),
(5,9001310,'2026-01-01 00:00:00'),
(5,9001339,'2026-01-01 00:00:00');

-- 普通角色只读用户/部门，可维护本人资料；demo_tree 同时示范部门身份切换。
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`,`created_at`) VALUES
('p','super_admin','*','/*','*','','','2026-01-01 00:00:00'),
('g','user:1','super_admin','*','','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/user/list','GET','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/user/:id','GET','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/dept/tree','GET','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/menu/my','GET','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/menu/my/perm-codes','GET','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/profile','GET','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/profile/','GET','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/profile','PUT','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/profile/','PUT','','','2026-01-01 00:00:00'),
('p','scope_all','*','/v1/sys/user/password/change','POST','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/user/list','GET','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/user/:id','GET','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/dept/tree','GET','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/menu/my','GET','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/menu/my/perm-codes','GET','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/profile','GET','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/profile/','GET','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/profile','PUT','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/profile/','PUT','','','2026-01-01 00:00:00'),
('p','scope_dept_children','*','/v1/sys/user/password/change','POST','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/user/list','GET','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/user/:id','GET','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/dept/tree','GET','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/menu/my','GET','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/menu/my/perm-codes','GET','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/profile','GET','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/profile/','GET','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/profile','PUT','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/profile/','PUT','','','2026-01-01 00:00:00'),
('p','scope_dept','*','/v1/sys/user/password/change','POST','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/user/list','GET','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/user/:id','GET','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/dept/tree','GET','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/menu/my','GET','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/menu/my/perm-codes','GET','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/profile','GET','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/profile/','GET','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/profile','PUT','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/profile/','PUT','','','2026-01-01 00:00:00'),
('p','scope_self','*','/v1/sys/user/password/change','POST','','','2026-01-01 00:00:00'),
('g','user:2','scope_all','dept:2','','','','2026-01-01 00:00:00'),
('g','user:3','scope_dept_children','dept:2','','','','2026-01-01 00:00:00'),
('g','user:3','scope_self','dept:5','','','','2026-01-01 00:00:00'),
('g','user:4','scope_dept','dept:3','','','','2026-01-01 00:00:00'),
('g','user:5','scope_self','dept:3','','','','2026-01-01 00:00:00'),
('g','user:6','scope_self','dept:4','','','','2026-01-01 00:00:00'),
('g','user:7','scope_self','dept:5','','','','2026-01-01 00:00:00'),
('g','user:8','scope_self','dept:5','','','','2026-01-01 00:00:00');

INSERT INTO `sys_dict` (`id`,`name`,`code`,`items`,`description`,`status`,`created_at`,`updated_at`) VALUES
(1,'启停状态','user_status','[{"label":"启用","value":1,"sort":0},{"label":"停用","value":2,"sort":1}]','后台用户、部门和角色状态',1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(2,'用户性别','user_gender','[{"label":"未知","value":0,"sort":0},{"label":"男","value":1,"sort":1},{"label":"女","value":2,"sort":2}]','后台用户性别',1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(3,'数据范围','data_scope','[{"label":"全部","value":"ALL","sort":0},{"label":"本部门及下级","value":"DEPT_AND_CHILDREN","sort":1},{"label":"本部门","value":"DEPT","sort":2},{"label":"本人","value":"SELF","sort":3}]','用户数据查询范围，多角色取并集',1,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(4,'APP会员套餐','app_membership_plan','[{"label":"基础会员","value":"basic","sort":0},{"label":"专业会员","value":"pro","sort":1}]','APP 后台开通会员时的套餐选择',1,'2026-01-01 00:00:00','2026-01-01 00:00:00');

-- 按当前 .api 注册的 v1 路由预置，分组/标题与后端同步入口一致；健康检查不进入业务资源。
INSERT INTO `sys_api` (`id`,`group_name`,`method`,`path`,`title`,`created_at`,`updated_at`) VALUES
(1,'app/auth','POST','/v1/app/auth/email-code','POST /v1/app/auth/email-code','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(2,'app/auth','POST','/v1/app/auth/login','POST /v1/app/auth/login','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(3,'app/auth','POST','/v1/app/auth/logout','POST /v1/app/auth/logout','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(4,'app/auth','GET','/v1/app/auth/options','GET /v1/app/auth/options','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(5,'app/auth','POST','/v1/app/auth/refresh','POST /v1/app/auth/refresh','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(6,'app/auth','POST','/v1/app/auth/register','POST /v1/app/auth/register','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(7,'app/feedback','GET','/v1/app/feedback','GET /v1/app/feedback','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(8,'app/feedback','POST','/v1/app/feedback','POST /v1/app/feedback','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(9,'app/membership','GET','/v1/app/membership','GET /v1/app/membership','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(10,'app/profile','GET','/v1/app/profile','GET /v1/app/profile','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(11,'app/profile','PUT','/v1/app/profile','PUT /v1/app/profile','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(12,'sys/ai','GET','/v1/sys/ai/conversations','GET /v1/sys/ai/conversations','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(13,'sys/ai','POST','/v1/sys/ai/conversations','POST /v1/sys/ai/conversations','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(14,'sys/ai','DELETE','/v1/sys/ai/conversations/:id','DELETE /v1/sys/ai/conversations/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(15,'sys/ai','GET','/v1/sys/ai/conversations/:id','GET /v1/sys/ai/conversations/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(16,'sys/ai','POST','/v1/sys/ai/messages','POST /v1/sys/ai/messages','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(17,'sys/ai','GET','/v1/sys/ai/tasks','GET /v1/sys/ai/tasks','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(18,'sys/ai','GET','/v1/sys/ai/tasks/:id','GET /v1/sys/ai/tasks/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(19,'sys/ai','POST','/v1/sys/ai/tasks/:id/approve','POST /v1/sys/ai/tasks/:id/approve','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(20,'sys/ai','POST','/v1/sys/ai/tasks/:id/reject','POST /v1/sys/ai/tasks/:id/reject','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(21,'sys/ai','GET','/v1/sys/ai/tools','GET /v1/sys/ai/tools','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(22,'sys/ai','GET','/v1/sys/ai/workflows','GET /v1/sys/ai/workflows','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(23,'sys/ai','POST','/v1/sys/ai/workflows','POST /v1/sys/ai/workflows','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(24,'sys/ai','DELETE','/v1/sys/ai/workflows/:id','DELETE /v1/sys/ai/workflows/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(25,'sys/ai','GET','/v1/sys/ai/workflows/:id','GET /v1/sys/ai/workflows/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(26,'sys/ai','PUT','/v1/sys/ai/workflows/:id','PUT /v1/sys/ai/workflows/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(27,'sys/ai','POST','/v1/sys/ai/workflows/:id/run','POST /v1/sys/ai/workflows/:id/run','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(28,'sys/api','DELETE','/v1/sys/api/:id','DELETE /v1/sys/api/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(29,'sys/api','GET','/v1/sys/api/list','GET /v1/sys/api/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(30,'sys/api','POST','/v1/sys/api/sync','POST /v1/sys/api/sync','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(31,'sys/app','GET','/v1/sys/app/feedback','GET /v1/sys/app/feedback','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(32,'sys/app','PUT','/v1/sys/app/feedback/:id','PUT /v1/sys/app/feedback/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(33,'sys/app','GET','/v1/sys/app/membership-plans','GET /v1/sys/app/membership-plans','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(34,'sys/app','GET','/v1/sys/app/memberships','GET /v1/sys/app/memberships','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(35,'sys/app','POST','/v1/sys/app/memberships','POST /v1/sys/app/memberships','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(36,'sys/app','DELETE','/v1/sys/app/memberships/:id','DELETE /v1/sys/app/memberships/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(37,'sys/app','GET','/v1/sys/app/users','GET /v1/sys/app/users','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(38,'sys/app','GET','/v1/sys/app/users/:id','GET /v1/sys/app/users/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(39,'sys/app','PUT','/v1/sys/app/users/:id','PUT /v1/sys/app/users/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(40,'sys/app','PUT','/v1/sys/app/users/:id/avatar','PUT /v1/sys/app/users/:id/avatar','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(41,'sys/app','POST','/v1/sys/app/users/:id/review','POST /v1/sys/app/users/:id/review','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(42,'sys/app','GET','/v1/sys/app/users/:id/sessions','GET /v1/sys/app/users/:id/sessions','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(43,'sys/app','DELETE','/v1/sys/app/users/:id/sessions/:session_id','DELETE /v1/sys/app/users/:id/sessions/:session_id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(44,'sys/audit','GET','/v1/sys/audit/list','GET /v1/sys/audit/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(45,'sys/auth','GET','/v1/sys/auth/captcha','GET /v1/sys/auth/captcha','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(46,'sys/auth','POST','/v1/sys/auth/login','POST /v1/sys/auth/login','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(47,'sys/auth','POST','/v1/sys/auth/logout','POST /v1/sys/auth/logout','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(48,'sys/auth','POST','/v1/sys/auth/refresh','POST /v1/sys/auth/refresh','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(49,'sys/auth','GET','/v1/sys/auth/sessions','GET /v1/sys/auth/sessions','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(50,'sys/auth','DELETE','/v1/sys/auth/sessions/:id','DELETE /v1/sys/auth/sessions/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(51,'sys/dept','DELETE','/v1/sys/dept/:id','DELETE /v1/sys/dept/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(52,'sys/dept','PUT','/v1/sys/dept/:id','PUT /v1/sys/dept/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(53,'sys/dept','POST','/v1/sys/dept/create','POST /v1/sys/dept/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(54,'sys/dept','GET','/v1/sys/dept/my','GET /v1/sys/dept/my','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(55,'sys/dept','GET','/v1/sys/dept/tree','GET /v1/sys/dept/tree','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(56,'sys/dict','DELETE','/v1/sys/dict/:id','DELETE /v1/sys/dict/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(57,'sys/dict','PUT','/v1/sys/dict/:id','PUT /v1/sys/dict/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(58,'sys/dict','GET','/v1/sys/dict/code/:code','GET /v1/sys/dict/code/:code','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(59,'sys/dict','POST','/v1/sys/dict/create','POST /v1/sys/dict/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(60,'sys/dict','GET','/v1/sys/dict/list','GET /v1/sys/dict/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(61,'sys/file','DELETE','/v1/sys/file/:id','DELETE /v1/sys/file/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(62,'sys/file','GET','/v1/sys/file/:id/url','GET /v1/sys/file/:id/url','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(63,'sys/file','GET','/v1/sys/file/list','GET /v1/sys/file/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(64,'sys/file','POST','/v1/sys/file/upload','POST /v1/sys/file/upload','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(65,'sys/menu','DELETE','/v1/sys/menu/:id','DELETE /v1/sys/menu/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(66,'sys/menu','GET','/v1/sys/menu/:id','GET /v1/sys/menu/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(67,'sys/menu','PUT','/v1/sys/menu/:id','PUT /v1/sys/menu/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(68,'sys/menu','POST','/v1/sys/menu/create','POST /v1/sys/menu/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(69,'sys/menu','GET','/v1/sys/menu/list','GET /v1/sys/menu/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(70,'sys/menu','GET','/v1/sys/menu/my','GET /v1/sys/menu/my','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(71,'sys/menu','GET','/v1/sys/menu/my/perm-codes','GET /v1/sys/menu/my/perm-codes','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(72,'sys/menu','GET','/v1/sys/menu/tree','GET /v1/sys/menu/tree','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(73,'sys/profile','GET','/v1/sys/profile','GET /v1/sys/profile','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(74,'sys/profile','PUT','/v1/sys/profile','PUT /v1/sys/profile','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(75,'sys/role','DELETE','/v1/sys/role/:id','DELETE /v1/sys/role/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(76,'sys/role','GET','/v1/sys/role/:id','GET /v1/sys/role/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(77,'sys/role','PUT','/v1/sys/role/:id','PUT /v1/sys/role/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(78,'sys/role','POST','/v1/sys/role/:id/menus','POST /v1/sys/role/:id/menus','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(79,'sys/role','POST','/v1/sys/role/create','POST /v1/sys/role/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(80,'sys/role','GET','/v1/sys/role/list','GET /v1/sys/role/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(81,'sys/tenant','DELETE','/v1/sys/tenant/:id','DELETE /v1/sys/tenant/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(82,'sys/tenant','GET','/v1/sys/tenant/:id','GET /v1/sys/tenant/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(83,'sys/tenant','POST','/v1/sys/tenant/:id/reset-admin','POST /v1/sys/tenant/:id/reset-admin','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(84,'sys/tenant','POST','/v1/sys/tenant/:id/start','POST /v1/sys/tenant/:id/start','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(85,'sys/tenant','POST','/v1/sys/tenant/:id/stop','POST /v1/sys/tenant/:id/stop','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(86,'sys/tenant','POST','/v1/sys/tenant/create','POST /v1/sys/tenant/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(87,'sys/tenant','GET','/v1/sys/tenant/list','GET /v1/sys/tenant/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(88,'sys/user','DELETE','/v1/sys/user/:id','DELETE /v1/sys/user/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(89,'sys/user','GET','/v1/sys/user/:id','GET /v1/sys/user/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(90,'sys/user','PUT','/v1/sys/user/:id','PUT /v1/sys/user/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(91,'sys/user','POST','/v1/sys/user/:id/password','POST /v1/sys/user/:id/password','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(92,'sys/user','POST','/v1/sys/user/create','POST /v1/sys/user/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(93,'sys/user','GET','/v1/sys/user/list','GET /v1/sys/user/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(94,'sys/user','POST','/v1/sys/user/password/change','POST /v1/sys/user/password/change','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(95,'sys/ai','POST','/v1/sys/ai/messages/stream','POST /v1/sys/ai/messages/stream','2026-01-01 00:00:00','2026-01-01 00:00:00');

-- APP 账号属于独立认证域，不用于后台登录；邮箱为保留演示域。
INSERT INTO `app_user` (`id`,`account`,`identity_kind`,`password_hash`,`nickname`,`remark`,`status`,`review_reason`,`email_verified_at`,`created_at`,`updated_at`) VALUES
(1,'alice@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','演示用户 Alice','虚构已启用账号，可用于本地 APP 登录','active','','2026-01-01 00:00:00','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(2,'bob@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','演示用户 Bob','虚构已启用账号，用于会员和反馈展示','active','','2026-01-01 00:00:00','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(3,'pending@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','待审核用户','虚构待审核账号','pending','','2026-01-01 00:00:00','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(4,'disabled@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','停用用户','虚构停用账号','disabled','演示：后台停用','2026-01-01 00:00:00','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(5,'rejected@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','驳回用户','虚构审核驳回账号','rejected','演示：注册资料需要补充','2026-01-01 00:00:00','2026-01-01 00:00:00','2026-01-01 00:00:00');

-- 以导入时间生成生效、过期、撤销、未生效四种示例，不包含支付订单。
INSERT INTO `app_membership_grant` (`id`,`user_id`,`plan_code`,`benefits`,`request_id`,`granted_by`,`starts_at`,`expires_at`,`revoked`,`revoked_at`,`created_at`,`updated_at`) VALUES
(1,1,'pro','profile.export','demo-membership-active',1,DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY),DATE_ADD(UTC_TIMESTAMP(), INTERVAL 30 DAY),0,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(2,2,'basic','','demo-membership-expired',1,DATE_SUB(UTC_TIMESTAMP(), INTERVAL 90 DAY),DATE_SUB(UTC_TIMESTAMP(), INTERVAL 30 DAY),0,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(3,2,'pro','profile.export','demo-membership-revoked',1,DATE_SUB(UTC_TIMESTAMP(), INTERVAL 10 DAY),DATE_ADD(UTC_TIMESTAMP(), INTERVAL 20 DAY),1,UTC_TIMESTAMP(),'2026-01-01 00:00:00','2026-01-01 00:00:00'),
(4,2,'basic','','demo-membership-upcoming',1,DATE_ADD(UTC_TIMESTAMP(), INTERVAL 7 DAY),DATE_ADD(UTC_TIMESTAMP(), INTERVAL 37 DAY),0,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00');

INSERT INTO `app_feedback` (`id`,`user_id`,`content`,`reply`,`status`,`created_at`,`updated_at`) VALUES
(1,1,'建议在账户设置中增加操作指引，方便新用户快速开始。','','pending','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(2,2,'会员状态说明很清晰，希望补充有效期的展示示例。','感谢建议，会员页面已展示生效时间和到期时间。','resolved','2026-01-01 00:00:00','2026-01-01 00:00:00'),
(3,1,'这是一条用于演示处理流程的虚构反馈。','','pending','2026-01-01 00:00:00','2026-01-01 00:00:00');

-- 以下表保留完整结构但无预置数据：
-- sys_file / sys_file_reference：不创建没有真实对象的文件登记。
-- sys_audit_log：只记录实际操作，不伪造审计历史。
-- ai_conversation / ai_message / ai_task / ai_workflow：配置模型后由真实使用产生。
-- tenant_registry：通过平台创建真实独立栈，不提供可误启停的假租户或共享凭据。

SET SQL_MODE=@OLD_SQL_MODE;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET TIME_ZONE=@OLD_TIME_ZONE;
SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT;
SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS;
SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION;

-- 初始化完成。
