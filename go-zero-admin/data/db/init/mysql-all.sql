-- ============================================================================
-- go-zero-admin 开源初始化数据库（完整可开发数据集）
--
-- 使用方式（二选一）：
--   1. 一键启动（推荐）：docker compose up -d —— MySQL 空卷时自动导入本文件
--      （compose 已将本目录挂载到 /docker-entrypoint-initdb.d，仅空卷生效）
--   2. 手动导入：mysql -h 127.0.0.1 -P 3307 -u root -p < mysql-all.sql
--
-- 内容：
--   gozero_admin    全量：17 张表结构 + 完整演示数据（菜单/角色/部门/用户/
--                         字典/API/Casbin 规则/文件登记/AI 与 App 演示行）
--   gozero_platform 全量：tenant_registry 含 3 条演示租户行（stopped/error/idle，
--                         其密钥列均为本地演示值，仅用于联调租户列表展示）
--
-- 开发账号（仅本地开发，生产必须修改）：
--   admin / Admin@123456    超级管理员（含全部菜单与 Casbin 权限）
--   user01..user28 / Admin@123456    演示用户（scope_all 等数据范围角色，GET 只读策略）
-- MySQL dump 10.13  Distrib 9.7.4, for Linux (x86_64)
--
-- Host: 127.0.0.1    Database: gozero_admin
-- ------------------------------------------------------
-- Server version	9.7.4

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Current Database: `gozero_admin`
--

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `gozero_admin` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci */ /*!80016 DEFAULT ENCRYPTION='N' */;

USE `gozero_admin`;

--
-- Table structure for table `ai_conversation`
--

DROP TABLE IF EXISTS `ai_conversation`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ai_conversation` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `user_id` bigint unsigned NOT NULL COMMENT '所属用户',
  `title` varchar(128) NOT NULL DEFAULT '' COMMENT '会话标题',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=9010305 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 会话';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_conversation`
--

LOCK TABLES `ai_conversation` WRITE;
/*!40000 ALTER TABLE `ai_conversation` DISABLE KEYS */;
INSERT INTO `ai_conversation` VALUES (9010301,1,'四种消息角色','2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010302,1,'多轮与工具消息','2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010303,9001100,'其他用户会话','2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010304,1,'空会话','2026-10-31 15:28:18','2026-10-31 18:23:19',NULL);
/*!40000 ALTER TABLE `ai_conversation` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ai_message`
--

DROP TABLE IF EXISTS `ai_message`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ai_message` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `conversation_id` bigint unsigned NOT NULL COMMENT '会话ID',
  `role` varchar(16) NOT NULL COMMENT '角色：user/assistant/tool/system',
  `content` text NOT NULL COMMENT '消息内容',
  `tool_calls` json DEFAULT NULL COMMENT '工具调用记录（JSON 数组）',
  `tokens` int unsigned NOT NULL DEFAULT '0' COMMENT '消耗 token 数',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_conversation` (`conversation_id`)
) ENGINE=InnoDB AUTO_INCREMENT=9010413 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 消息';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_message`
--

LOCK TABLES `ai_message` WRITE;
/*!40000 ALTER TABLE `ai_message` DISABLE KEYS */;
INSERT INTO `ai_message` VALUES (9010401,9010301,'system','模拟系统说明，不连接模型。',NULL,0,'2026-10-31 09:00:01'),(9010402,9010301,'user','请统计示例文本。',NULL,6,'2026-10-31 09:00:02'),(9010403,9010301,'assistant','模拟工具调用。','[{\"id\": \"call_01\", \"type\": \"function\", \"function\": {\"name\": \"text_stats\", \"arguments\": \"{\\\"text\\\":\\\"sample text\\\"}\"}}]',12,'2026-10-31 09:00:03'),(9010404,9010301,'tool','模拟结果：字符数 11，单词数 2，行数 1。',NULL,0,'2026-10-31 09:00:04'),(9010405,9010302,'user','第一轮：你好。',NULL,4,'2026-10-31 09:01:01'),(9010406,9010302,'assistant','第一轮：这是保存的示例回复。',NULL,8,'2026-10-31 09:01:02'),(9010407,9010302,'user','第二轮：中文、English 与多行\n第二行。',NULL,10,'2026-10-31 09:01:03'),(9010408,9010302,'assistant','第二轮：模拟工具调用。','[{\"id\": \"call_02\", \"type\": \"function\", \"function\": {\"name\": \"text_stats\", \"arguments\": \"{\\\"text\\\":\\\"example\\\"}\"}}]',9,'2026-10-31 09:01:04'),(9010409,9010302,'tool','模拟结果：字符数 7，单词数 1，行数 1。',NULL,0,'2026-10-31 09:01:05'),(9010410,9010302,'assistant','最终回复：统计完成。',NULL,6,'2026-10-31 09:01:06'),(9010411,9010303,'user','其他用户的示例提问。',NULL,5,'2026-10-31 09:02:01'),(9010412,9010303,'assistant','其他用户的模拟回复。',NULL,7,'2026-10-31 09:02:02');
/*!40000 ALTER TABLE `ai_message` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ai_task`
--

DROP TABLE IF EXISTS `ai_task`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
  PRIMARY KEY (`id`),
  KEY `idx_workflow` (`workflow_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=9010608 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 任务';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_task`
--

LOCK TABLES `ai_task` WRITE;
/*!40000 ALTER TABLE `ai_task` DISABLE KEYS */;
INSERT INTO `ai_task` VALUES (9010601,9010502,NULL,'pending','{\"fixture\":\"sample\",\"case\":\"pending\"}','','',NULL,'2026-10-31 09:10:01',NULL),(9010602,9010502,NULL,'running','{\"fixture\":\"sample\",\"case\":\"running\"}','','',NULL,'2026-10-31 09:10:02',NULL),(9010603,9010503,NULL,'pending_approval','{\"fixture\":\"sample\",\"case\":\"approve\"}','模拟审批通过','',NULL,'2026-10-31 09:10:03',NULL),(9010604,9010502,9010301,'succeeded','{\"fixture\":\"sample\",\"case\":\"succeeded\"}','模拟执行成功','',1,'2026-10-31 09:10:04','2026-10-31 09:11:04'),(9010605,9010502,9010302,'failed','{\"fixture\":\"sample\",\"case\":\"failed\"}','','模拟工具失败，不代表真实调用',NULL,'2026-10-31 09:10:05','2026-10-31 09:11:05'),(9010606,9010503,NULL,'rejected','{\"fixture\":\"sample\",\"case\":\"rejected\"}','模拟审批拒绝','',1,'2026-10-31 09:10:06','2026-10-31 09:11:06'),(9010607,9010503,9010303,'pending_approval','{\"fixture\":\"sample\",\"case\":\"reject\"}','模拟第二条审批拒绝','',NULL,'2026-10-31 09:10:07',NULL);
/*!40000 ALTER TABLE `ai_task` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ai_workflow`
--

DROP TABLE IF EXISTS `ai_workflow`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9010504 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 工作流';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_workflow`
--

LOCK TABLES `ai_workflow` WRITE;
/*!40000 ALTER TABLE `ai_workflow` DISABLE KEYS */;
INSERT INTO `ai_workflow` VALUES (9010501,'空步骤边界','{\"steps\": []}',0,0,NULL,1,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010502,'本地文本统计','{\"steps\": [{\"name\": \"示例统计\", \"tool\": \"text_stats\", \"input\": {\"text\": \"sample fixture\"}, \"approval\": false}]}',0,0,NULL,1,'2026-10-31 15:28:18','2026-10-31 18:31:03',NULL),(9010503,'审批展示','{\"steps\": [{\"name\": \"审批前统计\", \"tool\": \"text_stats\", \"input\": {\"text\": \"sample approval\"}, \"approval\": true}, {\"name\": \"后续统计\", \"tool\": \"text_stats\", \"input\": {\"text\": \"sample after approval\"}, \"approval\": false}]}',0,0,NULL,1,'2026-10-31 15:28:18','2026-10-31 18:32:47',NULL);
/*!40000 ALTER TABLE `ai_workflow` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_membership_grant`
--

DROP TABLE IF EXISTS `app_membership_grant`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9010207 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 会员授予';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_membership_grant`
--

LOCK TABLES `app_membership_grant` WRITE;
/*!40000 ALTER TABLE `app_membership_grant` DISABLE KEYS */;
INSERT INTO `app_membership_grant` VALUES (9010201,9010101,'vip','profile.export','grant_active',1,'2026-10-30 15:28:18','2026-11-30 15:28:18',0,NULL,'2026-10-31 15:28:18','2026-10-31 18:31:03'),(9010202,9010105,'vip','profile.export','grant_expired',1,'2026-09-01 15:28:18','2026-10-01 15:28:18',0,NULL,'2026-10-31 15:28:18','2026-10-31 18:31:03'),(9010203,9010105,'vip','profile.export','grant_future',1,'2026-11-30 15:28:18','2026-12-30 15:28:18',0,NULL,'2026-10-31 15:28:18','2026-10-31 18:31:03'),(9010204,9010101,'vip','profile.export','grant_revoked',1,'2026-10-24 15:28:18','2026-11-07 15:28:18',1,'2026-10-31 15:28:18','2026-10-31 15:28:18','2026-10-31 18:31:03'),(9010205,9010101,'vip','profile.export','grant_overlap',1,'2026-10-30 15:28:18','2026-11-07 15:28:18',0,NULL,'2026-10-31 15:28:18','2026-10-31 18:31:03'),(9010206,9010105,'vip','profile.export','grant_self',NULL,'2026-10-30 15:28:18','2026-11-07 15:28:18',0,NULL,'2026-10-31 15:28:18','2026-10-31 18:31:03');
/*!40000 ALTER TABLE `app_membership_grant` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_user`
--

DROP TABLE IF EXISTS `app_user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `account` varchar(128) NOT NULL COMMENT '账号（邮箱或手机号，小写）',
  `identity_kind` varchar(16) NOT NULL COMMENT '账号类型：email/phone',
  `password_hash` varchar(128) NOT NULL COMMENT '密码哈希（bcrypt）',
  `nickname` varchar(64) NOT NULL DEFAULT '' COMMENT '昵称',
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
) ENGINE=InnoDB AUTO_INCREMENT=9010109 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 用户';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_user`
--

LOCK TABLES `app_user` WRITE;
/*!40000 ALTER TABLE `app_user` DISABLE KEYS */;
INSERT INTO `app_user` VALUES (9010101,'active@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','邮箱已激活',NULL,'active','测试样例：已审核','2026-10-31 15:28:18',NULL,'2026-10-31 15:28:18','2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010102,'pending@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','邮箱待审核',NULL,'pending','',NULL,NULL,NULL,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010103,'disabled@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','邮箱已停用',NULL,'disabled','测试样例：账号停用','2026-10-31 15:28:18',NULL,NULL,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010104,'rejected@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','邮箱已驳回',NULL,'rejected','测试样例：审核资料不完整',NULL,NULL,NULL,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010105,'0009010105','phone','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','手机已激活',NULL,'active','测试样例：已审核',NULL,'2026-10-31 15:28:18','2026-10-31 15:28:18','2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010106,'0009010106','phone','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','手机待审核',NULL,'pending','',NULL,NULL,NULL,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010107,'0009010107','phone','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','手机已停用',NULL,'disabled','测试样例：账号停用',NULL,'2026-10-31 15:28:18',NULL,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010108,'0009010108','phone','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','手机已驳回',NULL,'rejected','测试样例：审核资料不完整',NULL,NULL,NULL,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL);
/*!40000 ALTER TABLE `app_user` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `casbin_rule`
--

DROP TABLE IF EXISTS `casbin_rule`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9001748 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Casbin 策略';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `casbin_rule`
--

LOCK TABLES `casbin_rule` WRITE;
/*!40000 ALTER TABLE `casbin_rule` DISABLE KEYS */;
INSERT INTO `casbin_rule` VALUES (1,'p','super_admin','*','*','*','','','2026-10-30 14:07:15'),(2,'p','dept_admin','dept:1','/v1/sys/user/*','GET','','','2026-10-30 14:07:15'),(3,'p','dept_admin','dept:1','/v1/sys/dept/*','GET','','','2026-10-30 14:07:15'),(4,'p','employee','dept:1','/v1/sys/profile/*','GET','','','2026-10-30 14:07:15'),(6,'g','user:1','super_admin','*','','','','2026-10-30 14:12:32'),(9001700,'p','scope_all','*','/v1/sys/*','GET','','','2026-10-30 09:00:00'),(9001701,'p','scope_dept_children','*','/v1/sys/*','GET','','','2026-10-30 09:00:00'),(9001702,'p','scope_dept','*','/v1/sys/*','GET','','','2026-10-30 09:00:00'),(9001703,'p','scope_self','*','/v1/sys/*','GET','','','2026-10-30 09:00:00'),(9001710,'g','user:9001100','scope_all','dept:9001000','','','','2026-10-30 09:00:00'),(9001711,'g','user:9001101','scope_dept_children','dept:9001001','','','','2026-10-30 09:00:00'),(9001712,'g','user:9001102','scope_dept','dept:9001002','','','','2026-10-30 09:00:00'),(9001713,'g','user:9001103','scope_self','dept:9001003','','','','2026-10-30 09:00:00'),(9001714,'g','user:9001104','scope_all','dept:9001004','','','','2026-10-30 09:00:00'),(9001715,'g','user:9001105','scope_dept_children','dept:9001000','','','','2026-10-30 09:00:00'),(9001716,'g','user:9001106','scope_dept','dept:9001001','','','','2026-10-30 09:00:00'),(9001717,'g','user:9001107','scope_self','dept:9001002','','','','2026-10-30 09:00:00'),(9001718,'g','user:9001108','scope_all','dept:9001003','','','','2026-10-30 09:00:00'),(9001719,'g','user:9001109','scope_dept_children','dept:9001004','','','','2026-10-30 09:00:00'),(9001720,'g','user:9001110','scope_dept','dept:9001000','','','','2026-10-30 09:00:00'),(9001721,'g','user:9001111','scope_self','dept:9001001','','','','2026-10-30 09:00:00'),(9001722,'g','user:9001112','scope_all','dept:9001002','','','','2026-10-30 09:00:00'),(9001723,'g','user:9001113','scope_dept_children','dept:9001003','','','','2026-10-30 09:00:00'),(9001724,'g','user:9001114','scope_dept','dept:9001004','','','','2026-10-30 09:00:00'),(9001725,'g','user:9001115','scope_self','dept:9001000','','','','2026-10-30 09:00:00'),(9001726,'g','user:9001116','scope_all','dept:9001001','','','','2026-10-30 09:00:00'),(9001727,'g','user:9001117','scope_dept_children','dept:9001002','','','','2026-10-30 09:00:00'),(9001728,'g','user:9001118','scope_dept','dept:9001003','','','','2026-10-30 09:00:00'),(9001729,'g','user:9001119','scope_self','dept:9001004','','','','2026-10-30 09:00:00'),(9001730,'g','user:9001120','scope_all','dept:9001000','','','','2026-10-30 09:00:00'),(9001731,'g','user:9001121','scope_dept_children','dept:9001001','','','','2026-10-30 09:00:00'),(9001732,'g','user:9001122','scope_dept','dept:9001002','','','','2026-10-30 09:00:00'),(9001733,'g','user:9001123','scope_self','dept:9001003','','','','2026-10-30 09:00:00'),(9001734,'g','user:9001124','scope_all','dept:9001004','','','','2026-10-30 09:00:00'),(9001735,'g','user:9001125','scope_dept_children','dept:9001000','','','','2026-10-30 09:00:00'),(9001736,'g','user:9001126','scope_dept','dept:9001001','','','','2026-10-30 09:00:00'),(9001737,'g','user:9001127','scope_self','dept:9001002','','','','2026-10-30 09:00:00');
/*!40000 ALTER TABLE `casbin_rule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `goose_db_version`
--

DROP TABLE IF EXISTS `goose_db_version`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `goose_db_version` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint NOT NULL,
  `is_applied` tinyint(1) NOT NULL,
  `tstamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `goose_db_version`
--

LOCK TABLES `goose_db_version` WRITE;
/*!40000 ALTER TABLE `goose_db_version` DISABLE KEYS */;
INSERT INTO `goose_db_version` VALUES (1,0,1,'2026-10-30 14:07:10'),(2,1,1,'2026-10-30 14:07:13'),(3,2,1,'2026-10-30 14:07:14');
/*!40000 ALTER TABLE `goose_db_version` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_api`
--

DROP TABLE IF EXISTS `sys_api`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9001543 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='API 资源';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_api`
--

LOCK TABLES `sys_api` WRITE;
/*!40000 ALTER TABLE `sys_api` DISABLE KEYS */;
INSERT INTO `sys_api` VALUES (9001500,'sys/auth','GET','/v1/sys/auth/captcha','获取验证码','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001501,'sys/auth','POST','/v1/sys/auth/login','登录','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001502,'sys/user','GET','/v1/sys/user/list','查询用户','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001503,'sys/user','GET','/v1/sys/user/:id','用户详情','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001504,'sys/user','POST','/v1/sys/user/create','新增用户','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001505,'sys/user','PUT','/v1/sys/user/:id','编辑用户','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001506,'sys/user','DELETE','/v1/sys/user/:id','删除用户','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001507,'sys/user','POST','/v1/sys/user/:id/password','重置密码','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001508,'sys/user','POST','/v1/sys/user/password/change','修改本人密码','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001509,'sys/dept','GET','/v1/sys/dept/tree','部门树','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001510,'sys/dept','POST','/v1/sys/dept/create','新增部门','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001511,'sys/dept','PUT','/v1/sys/dept/:id','编辑部门','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001512,'sys/dept','DELETE','/v1/sys/dept/:id','删除部门','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001513,'sys/role','GET','/v1/sys/role/list','查询角色','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001514,'sys/role','GET','/v1/sys/role/:id','角色详情','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001515,'sys/role','POST','/v1/sys/role/create','新增角色','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001516,'sys/role','PUT','/v1/sys/role/:id','编辑角色','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001517,'sys/role','DELETE','/v1/sys/role/:id','删除角色','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001518,'sys/role','POST','/v1/sys/role/:id/menus','角色菜单授权','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001519,'sys/menu','GET','/v1/sys/menu/list','查询菜单','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001520,'sys/menu','GET','/v1/sys/menu/:id','菜单详情','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001521,'sys/menu','GET','/v1/sys/menu/tree','菜单树','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001522,'sys/menu','GET','/v1/sys/menu/my','本人菜单','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001523,'sys/menu','GET','/v1/sys/menu/my/perm-codes','本人权限码','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001524,'sys/menu','POST','/v1/sys/menu/create','新增菜单','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001525,'sys/menu','PUT','/v1/sys/menu/:id','编辑菜单','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001526,'sys/menu','DELETE','/v1/sys/menu/:id','删除菜单','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001527,'sys/dict','GET','/v1/sys/dict/list','查询字典','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001528,'sys/dict','GET','/v1/sys/dict/code/:code','字典详情','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001529,'sys/dict','POST','/v1/sys/dict/create','新增字典','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001530,'sys/dict','PUT','/v1/sys/dict/:id','编辑字典','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001531,'sys/dict','DELETE','/v1/sys/dict/:id','删除字典','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001532,'sys/api','GET','/v1/sys/api/list','查询 API','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001533,'sys/api','DELETE','/v1/sys/api/:id','删除 API','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001534,'sys/file','GET','/v1/sys/file/list','查询文件','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001535,'sys/file','GET','/v1/sys/file/:id/url','获取文件链接','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001536,'sys/file','DELETE','/v1/sys/file/:id','删除文件','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001537,'sys/audit','GET','/v1/sys/audit/list','查询审计日志','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001538,'sys/profile','GET','/v1/sys/profile/','本人资料','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001539,'sys/profile','PUT','/v1/sys/profile/','更新本人资料','2026-10-30 09:00:00','2026-10-31 18:31:03'),(9001540,'system','GET','/ping','健康检查','2026-10-30 09:00:00','2026-10-31 18:31:03');
/*!40000 ALTER TABLE `sys_api` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_audit_log`
--

DROP TABLE IF EXISTS `sys_audit_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9001816 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='审计日志';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_audit_log`
--

LOCK TABLES `sys_audit_log` WRITE;
/*!40000 ALTER TABLE `sys_audit_log` DISABLE KEYS */;
INSERT INTO `sys_audit_log` VALUES (9001800,9001100,'user01','sys/user','create','POST','/v1/sys/user/create','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',1,'','127.0.0.1','frontend_fixture',15,'2026-10-30 09:01:00'),(9001801,9001101,'user02','sys/user','update','PUT','/v1/sys/user/9001100','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',1,'','127.0.0.1','frontend_fixture',22,'2026-10-30 09:02:00'),(9001802,9001102,'user03','sys/user','delete','DELETE','/v1/sys/user/9001127','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',1,'','127.0.0.1','frontend_fixture',29,'2026-10-30 09:03:00'),(9001803,9001103,'user04','sys/role','create','POST','/v1/sys/role/create','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',2,'模拟失败：用于验证错误信息展示','127.0.0.1','frontend_fixture',36,'2026-10-30 09:04:00'),(9001804,9001104,'user05','sys/role','update','PUT','/v1/sys/role/9001200','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',1,'','127.0.0.1','frontend_fixture',43,'2026-10-30 09:05:00'),(9001805,9001105,'user06','sys/dept','create','POST','/v1/sys/dept/create','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',1,'','127.0.0.1','frontend_fixture',50,'2026-10-30 09:06:00'),(9001806,9001106,'user07','sys/dept','update','PUT','/v1/sys/dept/9001000','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',1,'','127.0.0.1','frontend_fixture',57,'2026-10-30 09:07:00'),(9001807,9001107,'user08','sys/dict','create','POST','/v1/sys/dict/create','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',2,'模拟失败：用于验证错误信息展示','127.0.0.1','frontend_fixture',64,'2026-10-30 09:08:00'),(9001808,9001108,'user09','sys/dict','update','PUT','/v1/sys/dict/9001600','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',1,'','127.0.0.1','frontend_fixture',71,'2026-10-30 09:09:00'),(9001809,9001109,'user10','sys/api','list','GET','/v1/sys/api/list','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',1,'','127.0.0.1','frontend_fixture',78,'2026-10-30 09:10:00'),(9001810,9001110,'user11','sys/file','list','GET','/v1/sys/file/list','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',1,'','127.0.0.1','frontend_fixture',85,'2026-10-30 09:11:00'),(9001811,9001111,'user12','sys/auth','login','POST','/v1/sys/auth/login','{\"fixture\":\"frontend\",\"note\":\"仅用于前端联调的模拟记录，不代表真实操作\"}',2,'模拟失败：用于验证错误信息展示','127.0.0.1','frontend_fixture',92,'2026-10-30 09:12:00');
/*!40000 ALTER TABLE `sys_audit_log` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_dept`
--

DROP TABLE IF EXISTS `sys_dept`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_dept` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '父部门ID，0=根',
  `name` varchar(64) NOT NULL COMMENT '部门名称',
  `leader` varchar(64) NOT NULL DEFAULT '' COMMENT '负责人',
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
) ENGINE=InnoDB AUTO_INCREMENT=9001009 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='部门';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dept`
--

LOCK TABLES `sys_dept` WRITE;
/*!40000 ALTER TABLE `sys_dept` DISABLE KEYS */;
INSERT INTO `sys_dept` VALUES (1,0,'总部','','','',0,1,'/1/','2026-10-30 14:07:15','2026-10-30 14:07:15',NULL),(9001000,1,'研发中心','示例负责人甲','000-0000-0001','rd@example.test',10,1,'/1/9001000/','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001001,9001000,'平台研发组','示例负责人乙','000-0000-0002','platform@example.test',20,1,'/1/9001000/9001001/','2026-10-30 09:00:00','2026-10-31 18:32:47',NULL),(9001002,9001000,'测试组','','','',0,0,'/1/9001000/9001002/','2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001003,1,'客户服务中心','示例负责人丁','000-0000-0004','service@example.test',40,1,'/1/9001003/','2026-10-30 09:00:00','2026-10-31 18:32:47',NULL),(9001004,9001003,'停用示例组','示例负责人戊','000-0000-0005','disabled@example.test',50,2,'/1/9001003/9001004/','2026-10-30 09:00:00','2026-10-31 18:32:47',NULL);
/*!40000 ALTER TABLE `sys_dept` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_dict`
--

DROP TABLE IF EXISTS `sys_dict`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9010012 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='字典';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dict`
--

LOCK TABLES `sys_dict` WRITE;
/*!40000 ALTER TABLE `sys_dict` DISABLE KEYS */;
INSERT INTO `sys_dict` VALUES (9001600,'启停状态','user_status','[{\"sort\": 0, \"label\": \"启用\", \"value\": \"1\"}, {\"sort\": 1, \"label\": \"停用\", \"value\": \"2\"}]','前端联调字典',1,'2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001601,'用户性别','user_gender','[{\"sort\": 0, \"label\": \"未知\", \"value\": \"0\"}, {\"sort\": 1, \"label\": \"男\", \"value\": \"1\"}, {\"sort\": 2, \"label\": \"女\", \"value\": \"2\"}]','前端联调字典',1,'2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001602,'数据范围','data_scope','[{\"sort\": 0, \"label\": \"全部\", \"value\": \"ALL\"}, {\"sort\": 1, \"label\": \"本部门及下级\", \"value\": \"DEPT_AND_CHILDREN\"}, {\"sort\": 2, \"label\": \"本部门\", \"value\": \"DEPT\"}, {\"sort\": 3, \"label\": \"本人\", \"value\": \"SELF\"}]','前端联调字典',1,'2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001603,'工单优先级','priority','[{\"sort\": 0, \"label\": \"低\", \"value\": \"low\"}, {\"sort\": 1, \"label\": \"中\", \"value\": \"normal\"}, {\"sort\": 2, \"label\": \"高\", \"value\": \"high\"}]','前端联调字典',2,'2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9010010,'空字典','empty_dict','[]','全模块联调：空字典项边界',1,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL);
/*!40000 ALTER TABLE `sys_dict` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_file`
--

DROP TABLE IF EXISTS `sys_file`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9001907 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件资源';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_file`
--

LOCK TABLES `sys_file` WRITE;
/*!40000 ALTER TABLE `sys_file` DISABLE KEYS */;
INSERT INTO `sys_file` VALUES (9001901,'gozero-public','demo/frontend/demo-readme.txt','readme.txt','txt','text/plain',46,'public',0,1,'2026-10-30 15:14:21','2026-10-31 18:23:19',NULL),(9001902,'gozero-private','demo/frontend/demo-report.csv','report.csv','csv','text/csv',46,'private',0,1,'2026-10-30 15:14:21','2026-10-31 18:23:19',NULL),(9001903,'gozero-public','demo/frontend/demo-referenced.txt','referenced.txt','txt','text/plain',50,'public',1,1,'2026-10-30 15:14:21','2026-10-31 18:23:19',NULL);
/*!40000 ALTER TABLE `sys_file` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_menu`
--

DROP TABLE IF EXISTS `sys_menu`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9001356 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_menu`
--

LOCK TABLES `sys_menu` WRITE;
/*!40000 ALTER TABLE `sys_menu` DISABLE KEYS */;
INSERT INTO `sys_menu` VALUES (9001300,0,'系统管理','system',1,'/sys','','','system',10,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001301,0,'工作台','dashboard',2,'/dashboard','dashboard','','dashboard',0,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001302,9001300,'用户管理','user',2,'/sys/user','user','','user',20,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001303,9001300,'角色管理','role',2,'/sys/role','role','','role',21,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001304,9001300,'菜单管理','menu',2,'/sys/menu','menu','','menu',22,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001305,9001300,'部门管理','dept',2,'/sys/dept','dept','','dept',23,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001306,9001300,'字典管理','dict',2,'/sys/dict','dict','','dict',24,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001307,9001300,'API 资源','api',2,'/sys/api','api','','api',25,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001308,9001300,'文件管理','file',2,'/sys/file','file','','file',26,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001309,9001300,'审计日志','audit',2,'/sys/audit','audit','','audit',27,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001310,9001300,'个人中心','profile',2,'/sys/profile','profile','','profile',28,1,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001320,9001302,'新增用户','user_add',3,'','','sys:user:add','',0,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001321,9001302,'编辑用户','user_edit',3,'','','sys:user:edit','',1,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001322,9001302,'删除用户','user_delete',3,'','','sys:user:delete','',2,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001323,9001302,'重置密码','user_reset_password',3,'','','sys:user:reset-password','',3,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001324,9001303,'新增角色','role_add',3,'','','sys:role:add','',4,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001325,9001303,'编辑角色','role_edit',3,'','','sys:role:edit','',5,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001326,9001303,'删除角色','role_delete',3,'','','sys:role:delete','',6,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001327,9001303,'分配菜单','role_menus',3,'','','sys:role:menus','',7,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001328,9001304,'新增菜单','menu_add',3,'','','sys:menu:add','',8,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001329,9001304,'编辑菜单','menu_edit',3,'','','sys:menu:edit','',9,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001330,9001304,'删除菜单','menu_delete',3,'','','sys:menu:delete','',10,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001331,9001305,'新增部门','dept_add',3,'','','sys:dept:add','',11,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001332,9001305,'编辑部门','dept_edit',3,'','','sys:dept:edit','',12,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001333,9001305,'删除部门','dept_delete',3,'','','sys:dept:delete','',13,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001334,9001306,'新增字典','dict_add',3,'','','sys:dict:add','',14,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001335,9001306,'编辑字典','dict_edit',3,'','','sys:dict:edit','',15,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001336,9001306,'删除字典','dict_delete',3,'','','sys:dict:delete','',16,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001337,9001307,'删除 API','api_delete',3,'','','sys:api:delete','',17,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001338,9001308,'删除文件','file_delete',3,'','','sys:file:delete','',18,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001339,9001310,'编辑资料','profile_edit',3,'','','sys:profile:edit','',19,0,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL);
/*!40000 ALTER TABLE `sys_menu` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_role`
--

DROP TABLE IF EXISTS `sys_role`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9010006 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_role`
--

LOCK TABLES `sys_role` WRITE;
/*!40000 ALTER TABLE `sys_role` DISABLE KEYS */;
INSERT INTO `sys_role` VALUES (1,'超级管理员','super_admin','拥有所有权限，全局域','ALL',0,1,'2026-10-30 14:07:15','2026-10-30 14:07:15',NULL),(2,'部门管理员','dept_admin','管理本部门及下级','DEPT_AND_CHILDREN',1,1,'2026-10-30 14:07:15','2026-10-30 14:07:15',NULL),(3,'普通员工','employee','仅本人数据','SELF',2,1,'2026-10-30 14:07:15','2026-10-30 14:07:15',NULL),(9001200,'全部查看','scope_all','演示角色，仅配置 GET 权限；数据范围以后端实现为准','ALL',10,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001201,'部门及下级','scope_dept_children','演示角色，仅配置 GET 权限；数据范围以实际后端实现为准','DEPT_AND_CHILDREN',11,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001202,'部门查看','scope_dept','演示角色，仅配置 GET 权限；数据范围以实际后端实现为准','DEPT',12,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9001203,'本人查看','scope_self','演示角色，仅配置 GET 权限；数据范围以实际后端实现为准','SELF',13,1,'2026-10-30 09:00:00','2026-10-31 18:23:19',NULL),(9010001,'停用角色','disabled','全模块演示：停用角色，无授权关联','SELF',99,2,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL);
/*!40000 ALTER TABLE `sys_role` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_role_menu`
--

DROP TABLE IF EXISTS `sys_role_menu`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_role_menu` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `role_id` bigint unsigned NOT NULL,
  `menu_id` bigint unsigned NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_menu` (`role_id`,`menu_id`),
  KEY `idx_menu` (`menu_id`)
) ENGINE=InnoDB AUTO_INCREMENT=9001501 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色-菜单';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_role_menu`
--

LOCK TABLES `sys_role_menu` WRITE;
/*!40000 ALTER TABLE `sys_role_menu` DISABLE KEYS */;
INSERT INTO `sys_role_menu` VALUES (9001400,1,9001300,'2026-10-30 09:00:00'),(9001401,1,9001301,'2026-10-30 09:00:00'),(9001402,1,9001302,'2026-10-30 09:00:00'),(9001403,1,9001303,'2026-10-30 09:00:00'),(9001404,1,9001304,'2026-10-30 09:00:00'),(9001405,1,9001305,'2026-10-30 09:00:00'),(9001406,1,9001306,'2026-10-30 09:00:00'),(9001407,1,9001307,'2026-10-30 09:00:00'),(9001408,1,9001308,'2026-10-30 09:00:00'),(9001409,1,9001309,'2026-10-30 09:00:00'),(9001410,1,9001310,'2026-10-30 09:00:00'),(9001411,1,9001320,'2026-10-30 09:00:00'),(9001412,1,9001321,'2026-10-30 09:00:00'),(9001413,1,9001322,'2026-10-30 09:00:00'),(9001414,1,9001323,'2026-10-30 09:00:00'),(9001415,1,9001324,'2026-10-30 09:00:00'),(9001416,1,9001325,'2026-10-30 09:00:00'),(9001417,1,9001326,'2026-10-30 09:00:00'),(9001418,1,9001327,'2026-10-30 09:00:00'),(9001419,1,9001328,'2026-10-30 09:00:00'),(9001420,1,9001329,'2026-10-30 09:00:00'),(9001421,1,9001330,'2026-10-30 09:00:00'),(9001422,1,9001331,'2026-10-30 09:00:00'),(9001423,1,9001332,'2026-10-30 09:00:00'),(9001424,1,9001333,'2026-10-30 09:00:00'),(9001425,1,9001334,'2026-10-30 09:00:00'),(9001426,1,9001335,'2026-10-30 09:00:00'),(9001427,1,9001336,'2026-10-30 09:00:00'),(9001428,1,9001337,'2026-10-30 09:00:00'),(9001429,1,9001338,'2026-10-30 09:00:00'),(9001430,1,9001339,'2026-10-30 09:00:00'),(9001451,9001201,9001300,'2026-10-30 09:00:00'),(9001452,9001201,9001301,'2026-10-30 09:00:00'),(9001453,9001201,9001302,'2026-10-30 09:00:00'),(9001454,9001201,9001303,'2026-10-30 09:00:00'),(9001455,9001201,9001304,'2026-10-30 09:00:00'),(9001456,9001201,9001305,'2026-10-30 09:00:00'),(9001457,9001201,9001306,'2026-10-30 09:00:00'),(9001458,9001201,9001307,'2026-10-30 09:00:00'),(9001459,9001201,9001308,'2026-10-30 09:00:00'),(9001460,9001201,9001309,'2026-10-30 09:00:00'),(9001461,9001201,9001310,'2026-10-30 09:00:00'),(9001462,9001202,9001300,'2026-10-30 09:00:00'),(9001463,9001202,9001301,'2026-10-30 09:00:00'),(9001464,9001202,9001302,'2026-10-30 09:00:00'),(9001465,9001202,9001303,'2026-10-30 09:00:00'),(9001466,9001202,9001304,'2026-10-30 09:00:00'),(9001467,9001202,9001305,'2026-10-30 09:00:00'),(9001468,9001202,9001306,'2026-10-30 09:00:00'),(9001469,9001202,9001307,'2026-10-30 09:00:00'),(9001470,9001202,9001308,'2026-10-30 09:00:00'),(9001471,9001202,9001309,'2026-10-30 09:00:00'),(9001472,9001202,9001310,'2026-10-30 09:00:00'),(9001473,9001203,9001300,'2026-10-30 09:00:00'),(9001474,9001203,9001301,'2026-10-30 09:00:00'),(9001475,9001203,9001302,'2026-10-30 09:00:00'),(9001476,9001203,9001303,'2026-10-30 09:00:00'),(9001477,9001203,9001304,'2026-10-30 09:00:00'),(9001478,9001203,9001305,'2026-10-30 09:00:00'),(9001479,9001203,9001306,'2026-10-30 09:00:00'),(9001480,9001203,9001307,'2026-10-30 09:00:00'),(9001481,9001203,9001308,'2026-10-30 09:00:00'),(9001482,9001203,9001309,'2026-10-30 09:00:00'),(9001483,9001203,9001310,'2026-10-30 09:00:00'),(9001490,9001200,9001300,'2026-10-31 17:09:17'),(9001491,9001200,9001301,'2026-10-31 17:09:17'),(9001492,9001200,9001302,'2026-10-31 17:09:17'),(9001493,9001200,9001303,'2026-10-31 17:09:17'),(9001494,9001200,9001304,'2026-10-31 17:09:17'),(9001495,9001200,9001305,'2026-10-31 17:09:17'),(9001496,9001200,9001306,'2026-10-31 17:09:17'),(9001497,9001200,9001307,'2026-10-31 17:09:17'),(9001498,9001200,9001308,'2026-10-31 17:09:17'),(9001499,9001200,9001309,'2026-10-31 17:09:17'),(9001500,9001200,9001310,'2026-10-31 17:09:17');
/*!40000 ALTER TABLE `sys_role_menu` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_user`
--

DROP TABLE IF EXISTS `sys_user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
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
) ENGINE=InnoDB AUTO_INCREMENT=9001153 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='后台用户';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_user`
--

LOCK TABLES `sys_user` WRITE;
/*!40000 ALTER TABLE `sys_user` DISABLE KEYS */;
INSERT INTO `sys_user` VALUES (1,'admin','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','超级管理员','Admin','','',1,NULL,0,1,'','2026-10-31 18:26:53','172.20.0.1','2026-10-30 14:07:15','2026-10-31 18:26:53',NULL),(9001100,'user01','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户01','示例用户1','user01@example.test','000-0000-0001',9001000,NULL,0,1,'前端联调账号，请勿用于真实业务','2026-10-31 18:24:33','172.20.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001101,'user02','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户02','示例用户2','user02@example.test','000-0000-0002',9001001,NULL,1,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:02:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001102,'user03','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户03','示例用户3','user03@example.test','000-0000-0003',9001002,NULL,2,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:03:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001103,'user04','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户04','示例用户4','user04@example.test','000-0000-0004',9001003,NULL,0,1,'前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001104,'user05','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户05','示例用户5','user05@example.test','000-0000-0005',9001004,NULL,1,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:05:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001105,'user06','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户06','示例用户6','user06@example.test','000-0000-0006',9001000,NULL,2,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:06:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001106,'user07','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户07','示例用户7','user07@example.test','000-0000-0007',9001001,NULL,0,2,'前端联调账号，请勿用于真实业务','2026-10-30 08:07:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001107,'user08','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户08','示例用户8','user08@example.test','000-0000-0008',9001002,NULL,1,1,'前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001108,'user09','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户09','示例用户9','user09@example.test','000-0000-0009',9001003,NULL,2,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:09:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001109,'user10','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户10','示例用户10','user10@example.test','000-0000-0010',9001004,NULL,0,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:10:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001110,'user11','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户11','示例用户11','user11@example.test','000-0000-0011',9001000,NULL,1,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:11:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001111,'user12','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户12','示例用户12','user12@example.test','000-0000-0012',9001001,NULL,2,1,'前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001112,'user13','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户13','示例用户13','user13@example.test','000-0000-0013',9001002,NULL,0,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:13:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001113,'user14','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户14','示例用户14','user14@example.test','000-0000-0014',9001003,NULL,1,2,'前端联调账号，请勿用于真实业务','2026-10-30 08:14:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001114,'user15','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户15','示例用户15','user15@example.test','000-0000-0015',9001004,NULL,2,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:15:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001115,'user16','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户16','示例用户16','user16@example.test','000-0000-0016',9001000,NULL,0,1,'前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001116,'user17','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户17','示例用户17','user17@example.test','000-0000-0017',9001001,NULL,1,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:17:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001117,'user18','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户18','示例用户18','user18@example.test','000-0000-0018',9001002,NULL,2,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:18:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001118,'user19','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户19','示例用户19','user19@example.test','000-0000-0019',9001003,NULL,0,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:19:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001119,'user20','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户20','示例用户20','user20@example.test','000-0000-0020',9001004,NULL,1,1,'前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001120,'user21','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户21','示例用户21','user21@example.test','000-0000-0021',9001000,NULL,2,2,'前端联调账号，请勿用于真实业务','2026-10-30 08:21:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001121,'user22','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户22','示例用户22','user22@example.test','000-0000-0022',9001001,NULL,0,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:22:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001122,'user23','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户23','示例用户23','user23@example.test','000-0000-0023',9001002,NULL,1,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:23:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001123,'user24','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户24','示例用户24','user24@example.test','000-0000-0024',9001003,NULL,2,1,'前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001124,'user25','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户25','示例用户25','user25@example.test','000-0000-0025',9001004,NULL,0,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:25:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001125,'user26','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户26','示例用户26','user26@example.test','000-0000-0026',9001000,NULL,1,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:26:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001126,'user27','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户27','示例用户27','user27@example.test','000-0000-0027',9001001,NULL,2,1,'前端联调账号，请勿用于真实业务','2026-10-30 08:27:00','127.0.0.1','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL),(9001127,'user28','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','联调用户28','示例用户28','user28@example.test','000-0000-0028',9001002,NULL,0,2,'前端联调账号，请勿用于真实业务',NULL,'','2026-10-30 09:00:00','2026-10-31 18:31:03',NULL);
/*!40000 ALTER TABLE `sys_user` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-10-31 18:32:50

-- MySQL dump 10.13  Distrib 9.7.4, for Linux (x86_64)
--
-- Host: 127.0.0.1    Database: gozero_platform
-- ------------------------------------------------------
-- Server version	9.7.4

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Current Database: `gozero_platform`
--

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `gozero_platform` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci */ /*!80016 DEFAULT ENCRYPTION='N' */;

USE `gozero_platform`;

--
-- Table structure for table `goose_db_version`
--

DROP TABLE IF EXISTS `goose_db_version`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `goose_db_version` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint NOT NULL,
  `is_applied` tinyint(1) NOT NULL,
  `tstamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `goose_db_version`
--

LOCK TABLES `goose_db_version` WRITE;
/*!40000 ALTER TABLE `goose_db_version` DISABLE KEYS */;
INSERT INTO `goose_db_version` VALUES (1,0,1,'2026-10-30 14:07:14'),(2,1,1,'2026-10-30 14:07:14');
/*!40000 ALTER TABLE `goose_db_version` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `tenant_registry`
--

DROP TABLE IF EXISTS `tenant_registry`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tenant_registry` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `slug` varchar(32) NOT NULL COMMENT '租户标识（英文小写，子域名前缀）',
  `domain` varchar(128) NOT NULL COMMENT '完整域名，如 t1.localhost',
  `compose_project` varchar(64) NOT NULL COMMENT 'compose 项目名，如 gozero-t-t1',
  `status` varchar(16) NOT NULL DEFAULT 'active' COMMENT '状态：active/stopped/error',
  `mysql_root_password` varchar(128) NOT NULL COMMENT '该租户 MySQL root 密码（随机生成）',
  `jwt_access_secret` varchar(128) NOT NULL COMMENT '该租户后台 JWT 密钥（随机生成）',
  `app_jwt_access_secret` varchar(128) NOT NULL COMMENT '该租户 App JWT 密钥（随机生成）',
  `error_msg` varchar(512) NOT NULL DEFAULT '' COMMENT '最近一次错误信息',
  `created_by` bigint unsigned DEFAULT NULL COMMENT '创建操作员ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除（注销租户）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_slug` (`slug`),
  UNIQUE KEY `uk_domain` (`domain`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=9010704 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户注册表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `tenant_registry`
--

LOCK TABLES `tenant_registry` WRITE;
/*!40000 ALTER TABLE `tenant_registry` DISABLE KEYS */;
INSERT INTO `tenant_registry` VALUES (9010701,'stopped','stopped.example.test','gozero-tenant-stopped','stopped','EE42CBCACB3690C26A1994CFBE2883AE710234C86F551A00D8989DFEFF03EE56','E1C90B3C2802DC1881D7C99E117F85E707DBDBF97E32B0CD5FE06BB84BE2AE42','2A00452BA1DF67E102CBB4F5DAFF8CA8798865C74BB40D89E357AAFBDB82D62D','纯测试展示，未部署容器；禁止启停、删除栈或重置管理员密码。',1,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010702,'error','error.example.test','gozero-tenant-error','error','697CF020CC3CE2A6DAB85CB93C601BAFA88DF3CFFEB5E1425566AF841D061D15','DE11ACF182BF5B8C215B282A9266EC5EF598C487882704C0C44243322CFB1928','403E53A57CE3F9A2EDE6201A3B05A090E39858DD11AF6D3328E48C153A12DFD4','模拟部署失败；纯测试展示，未部署容器，禁止生命周期操作。',1,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL),(9010703,'idle','idle.example.test','gozero-tenant-idle','stopped','D1292122EEA2A8BDF2F13C769FC65D1CAE0292137175274D5EF854C7A0AEFFA7','20B1965040D1E088D1127E52C125741E1055D4CA5B6B2CA0133F5D4FE5246173','34904873CBF1773084BEC2FD00F6773B0EE07A457FC4B487445887A50756FAA1','空闲展示样例，未部署容器；禁止生命周期操作。',NULL,'2026-10-31 15:28:18','2026-10-31 18:23:19',NULL);
/*!40000 ALTER TABLE `tenant_registry` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-10-31 18:32:50

