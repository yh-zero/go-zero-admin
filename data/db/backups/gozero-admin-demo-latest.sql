-- MySQL dump 10.13  Distrib 8.4.11, for Linux (x86_64)
--
-- Host: localhost    Database: goZero-admin
-- ------------------------------------------------------
-- Server version	8.4.11

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
-- Current Database: `goZero-admin`
--

/*!40000 DROP DATABASE IF EXISTS `goZero-admin`*/;

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `goZero-admin` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci */ /*!80016 DEFAULT ENCRYPTION='N' */;

USE `goZero-admin`;

--
-- Table structure for table `app_membership_grants`
--

DROP TABLE IF EXISTS `app_membership_grants`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_membership_grants` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `plan_id` bigint NOT NULL,
  `plan_code` varchar(32) NOT NULL,
  `plan_version` bigint NOT NULL,
  `entitlements` json NOT NULL,
  `request_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `days` bigint NOT NULL,
  `starts_at` datetime(3) NOT NULL,
  `ends_at` datetime(3) NOT NULL,
  `revoked_at` datetime(3) DEFAULT NULL,
  `granted_by` bigint NOT NULL,
  `revoked_by` bigint NOT NULL DEFAULT '0',
  `reason` varchar(500) NOT NULL,
  `revoke_reason` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_grant_request` (`request_id`),
  KEY `idx_app_grant_user_end` (`user_id`,`ends_at`),
  KEY `idx_app_grant_revoked` (`revoked_at`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_membership_grants`
--

LOCK TABLES `app_membership_grants` WRITE;
/*!40000 ALTER TABLE `app_membership_grants` DISABLE KEYS */;
INSERT INTO `app_membership_grants` VALUES (1,2,1,'vip',1,'[\"profile.export\"]','demo-vip-initial-grant',30,'2026-10-20 14:56:14.543','2026-11-19 14:56:14.543',NULL,1,0,'初始化演示会员，导入时起有效30天','','2026-10-20 14:56:14.543'),(2,6,1,'vip',1,'[\"profile.export\"]','demo-vip-expired-grant',30,'2026-08-21 14:56:14.543','2026-09-20 14:56:14.543',NULL,1,0,'内置演示：会员已过期','','2026-08-21 14:56:14.543'),(3,1,1,'vip',1,'[\"profile.export\"]','demo-vip-revoked-grant',30,'2026-10-10 14:56:14.543','2026-11-09 14:56:14.543','2026-10-19 14:56:14.543',1,1,'内置演示：曾获会员授予','内置演示：会员已撤销','2026-10-10 14:56:14.543');
/*!40000 ALTER TABLE `app_membership_grants` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_membership_plans`
--

DROP TABLE IF EXISTS `app_membership_plans`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_membership_plans` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `code` varchar(32) NOT NULL,
  `name` varchar(64) NOT NULL,
  `version` bigint NOT NULL,
  `entitlements` json NOT NULL,
  `enabled` tinyint(1) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_plan_version` (`code`,`version`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_membership_plans`
--

LOCK TABLES `app_membership_plans` WRITE;
/*!40000 ALTER TABLE `app_membership_plans` DISABLE KEYS */;
INSERT INTO `app_membership_plans` VALUES (1,'vip','VIP 会员',1,'[\"profile.export\"]',1,'2026-10-20 14:56:12.754','2026-10-20 14:56:12.754');
/*!40000 ALTER TABLE `app_membership_plans` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_sessions`
--

DROP TABLE IF EXISTS `app_sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_sessions` (
  `id` varchar(36) NOT NULL,
  `user_id` bigint NOT NULL,
  `refresh_hash` varchar(64) NOT NULL,
  `previous_refresh_hash` varchar(64) NOT NULL DEFAULT '',
  `refresh_generation` bigint NOT NULL DEFAULT '1',
  `session_version` bigint NOT NULL,
  `ip` varchar(64) NOT NULL,
  `user_agent` varchar(512) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `revoked_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_app_session_user` (`user_id`),
  KEY `idx_app_session_expiry` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_sessions`
--

LOCK TABLES `app_sessions` WRITE;
/*!40000 ALTER TABLE `app_sessions` DISABLE KEYS */;
/*!40000 ALTER TABLE `app_sessions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_user_identities`
--

DROP TABLE IF EXISTS `app_user_identities`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_user_identities` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `kind` varchar(16) NOT NULL,
  `identifier` varchar(254) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `verified_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_identity_user` (`user_id`),
  UNIQUE KEY `uk_app_identity_identifier` (`identifier`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_user_identities`
--

LOCK TABLES `app_user_identities` WRITE;
/*!40000 ALTER TABLE `app_user_identities` DISABLE KEYS */;
INSERT INTO `app_user_identities` VALUES (1,1,'email','member@example.test','$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy',NULL,'2026-10-20 14:56:14.543'),(2,2,'email','vip@example.test','$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy',NULL,'2026-10-20 14:56:14.543'),(3,3,'email','pending@example.test','$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy',NULL,'2026-10-20 14:56:14.543'),(4,4,'email','disabled@example.test','$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy',NULL,'2026-10-20 14:56:14.543'),(5,5,'email','rejected@example.test','$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy',NULL,'2026-10-20 14:56:14.543'),(6,6,'email','expired@example.test','$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy',NULL,'2026-10-20 14:56:14.543');
/*!40000 ALTER TABLE `app_user_identities` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_users`
--

DROP TABLE IF EXISTS `app_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `nickname` varchar(64) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `session_version` bigint NOT NULL DEFAULT '1',
  `review_reason` varchar(500) NOT NULL DEFAULT '',
  `reviewed_by` bigint NOT NULL DEFAULT '0',
  `reviewed_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  `registration_ip` varchar(64) NOT NULL DEFAULT '',
  `registration_region` varchar(255) NOT NULL DEFAULT '',
  `avatar_url` varchar(1024) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_app_user_status` (`status`,`id`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_users`
--

LOCK TABLES `app_users` WRITE;
/*!40000 ALTER TABLE `app_users` DISABLE KEYS */;
INSERT INTO `app_users` VALUES (1,'普通用户示例','active',1,'初始化演示账号',1,'2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','','',''),(2,'会员用户示例','active',1,'初始化演示账号',1,'2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','','',''),(3,'待审核用户示例','pending',1,'',0,NULL,'2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','','',''),(4,'禁用用户示例','disabled',1,'内置演示：账号已禁用',1,'2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','','',''),(5,'审核拒绝示例','rejected',1,'内置演示：注册资料待完善',1,'2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','','',''),(6,'会员过期示例','active',1,'初始化演示账号',1,'2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','2026-10-20 14:56:14.543','','','');
/*!40000 ALTER TABLE `app_users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `casbin_rule`
--

DROP TABLE IF EXISTS `casbin_rule`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `casbin_rule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ptype` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `v0` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `v1` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `v2` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `v3` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `v4` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `v5` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `idx_casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=294 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `casbin_rule`
--

LOCK TABLES `casbin_rule` WRITE;
/*!40000 ALTER TABLE `casbin_rule` DISABLE KEYS */;
INSERT INTO `casbin_rule` VALUES (251,'p','1','/v1/ai/conversations','GET',NULL,NULL,NULL),(252,'p','1','/v1/ai/conversations/:id/messages','GET',NULL,NULL,NULL),(247,'p','1','/v1/ai/info','GET',NULL,NULL,NULL),(248,'p','1','/v1/ai/runs','POST',NULL,NULL,NULL),(249,'p','1','/v1/ai/runs/:id','GET',NULL,NULL,NULL),(250,'p','1','/v1/ai/runs/:id/cancel','POST',NULL,NULL,NULL),(223,'p','1','/v1/sys/api/applySync','POST',NULL,NULL,NULL),(64,'p','1','/v1/sys/api/createApi','POST','','',''),(65,'p','1','/v1/sys/api/deleteApi','DELETE','','',''),(67,'p','1','/v1/sys/api/deleteApisByIds','DELETE','','',''),(66,'p','1','/v1/sys/api/getAllApiList','GET','','',''),(63,'p','1','/v1/sys/api/getApiList','GET','','',''),(222,'p','1','/v1/sys/api/previewSync','GET',NULL,NULL,NULL),(68,'p','1','/v1/sys/api/updateApi','PUT','','',''),(271,'p','1','/v1/sys/app-members/avatar','PUT','','',''),(256,'p','1','/v1/sys/app-members/grant','DELETE','','',''),(257,'p','1','/v1/sys/app-members/grants','GET','','',''),(258,'p','1','/v1/sys/app-members/grants','POST','','',''),(259,'p','1','/v1/sys/app-members/plans','GET','','',''),(260,'p','1','/v1/sys/app-members/session','DELETE','','',''),(261,'p','1','/v1/sys/app-members/sessions','GET','','',''),(262,'p','1','/v1/sys/app-members/status','PUT','','',''),(263,'p','1','/v1/sys/app-members/users','GET','','',''),(225,'p','1','/v1/sys/audit/getAuditLogList','GET',NULL,NULL,NULL),(59,'p','1','/v1/sys/authority/addAuthorityMenu','POST','','',''),(61,'p','1','/v1/sys/authority/createAuthority','POST','','',''),(62,'p','1','/v1/sys/authority/deleteAuthority','DELETE','','',''),(58,'p','1','/v1/sys/authority/getAuthorityList','GET','','',''),(60,'p','1','/v1/sys/authority/updateAuthority','PUT','','',''),(84,'p','1','/v1/sys/base/sendEmailCode','POST','','',''),(82,'p','1','/v1/sys/base/uploadFileImg','POST','','',''),(79,'p','1','/v1/sys/casbin/getPathByAuthorityId','GET','','',''),(80,'p','1','/v1/sys/casbin/updateCasbinData','PUT','','',''),(81,'p','1','/v1/sys/casbin/updateCasbinDataByApiIds','PUT','','',''),(49,'p','1','/v1/sys/deleteUser','DELETE','','',''),(70,'p','1','/v1/sys/dictionary/createSysDictionary','POST','','',''),(77,'p','1','/v1/sys/dictionary/createSysDictionaryInfo','POST','','',''),(73,'p','1','/v1/sys/dictionary/deleteSysDictionary','DELETE','','',''),(78,'p','1','/v1/sys/dictionary/deleteSysDictionaryInfo','DELETE','','',''),(71,'p','1','/v1/sys/dictionary/getSysDictionaryDetails','GET','','',''),(74,'p','1','/v1/sys/dictionary/getSysDictionaryInfoList','GET','','',''),(75,'p','1','/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','GET','','',''),(83,'p','1','/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','GET','','',''),(69,'p','1','/v1/sys/dictionary/getSysDictionaryList','GET','','',''),(72,'p','1','/v1/sys/dictionary/updateSysDictionary','PUT','','',''),(76,'p','1','/v1/sys/dictionary/updateSysDictionaryInfo','PUT','','',''),(226,'p','1','/v1/sys/files/list','GET',NULL,NULL,NULL),(227,'p','1','/v1/sys/files/reference','DELETE',NULL,NULL,NULL),(228,'p','1','/v1/sys/files/reference','POST',NULL,NULL,NULL),(229,'p','1','/v1/sys/files/resource','DELETE',NULL,NULL,NULL),(230,'p','1','/v1/sys/files/url','GET',NULL,NULL,NULL),(45,'p','1','/v1/sys/getUserList','GET','','',''),(43,'p','1','/v1/sys/login','POST','','',''),(52,'p','1','/v1/sys/menu/addBaseMenu','POST','','',''),(57,'p','1','/v1/sys/menu/deleteBaseMenu','DELETE','','',''),(219,'p','1','/v1/sys/menu/getAuthorityButtons','GET',NULL,NULL,NULL),(55,'p','1','/v1/sys/menu/getBaseMenuById','GET','','',''),(53,'p','1','/v1/sys/menu/getBaseMenuTree','GET','','',''),(50,'p','1','/v1/sys/menu/getMenu','GET','','',''),(54,'p','1','/v1/sys/menu/getMenuAuthority','GET','','',''),(51,'p','1','/v1/sys/menu/getMenuList','GET','','',''),(254,'p','1','/v1/sys/menu/previewMove','POST','','',''),(220,'p','1','/v1/sys/menu/updateAuthorityButtons','PUT',NULL,NULL,NULL),(56,'p','1','/v1/sys/menu/updateBaseMenu','PUT','','',''),(243,'p','1','/v1/sys/organization/dataScope','GET',NULL,NULL,NULL),(244,'p','1','/v1/sys/organization/dataScope','PUT',NULL,NULL,NULL),(236,'p','1','/v1/sys/organization/departments','DELETE',NULL,NULL,NULL),(233,'p','1','/v1/sys/organization/departments','GET',NULL,NULL,NULL),(234,'p','1','/v1/sys/organization/departments','POST',NULL,NULL,NULL),(235,'p','1','/v1/sys/organization/departments','PUT',NULL,NULL,NULL),(241,'p','1','/v1/sys/organization/membership','GET',NULL,NULL,NULL),(242,'p','1','/v1/sys/organization/membership','PUT',NULL,NULL,NULL),(240,'p','1','/v1/sys/organization/positions','DELETE',NULL,NULL,NULL),(237,'p','1','/v1/sys/organization/positions','GET',NULL,NULL,NULL),(238,'p','1','/v1/sys/organization/positions','POST',NULL,NULL,NULL),(239,'p','1','/v1/sys/organization/positions','PUT',NULL,NULL,NULL),(253,'p','1','/v1/sys/permissions/edit','GET','','',''),(44,'p','1','/v1/sys/randomImage','GET','','',''),(46,'p','1','/v1/sys/register','POST','','',''),(48,'p','1','/v1/sys/resetUserPassword','PUT','','',''),(246,'p','1','/v1/sys/session/admin/device','DELETE',NULL,NULL,NULL),(245,'p','1','/v1/sys/session/admin/devices','GET',NULL,NULL,NULL),(47,'p','1','/v1/sys/updateUserInfo','PUT','','',''),(272,'p','2','/v1/sys/app-members/avatar','PUT','','',''),(273,'p','2','/v1/sys/app-members/grants','GET','','',''),(274,'p','2','/v1/sys/app-members/plans','GET','','',''),(275,'p','2','/v1/sys/app-members/sessions','GET','','',''),(276,'p','2','/v1/sys/app-members/status','PUT','','',''),(277,'p','2','/v1/sys/app-members/users','GET','','',''),(278,'p','2','/v1/sys/base/uploadFileImg','POST','','',''),(279,'p','2','/v1/sys/files/list','GET','','',''),(280,'p','2','/v1/sys/files/url','GET','','',''),(281,'p','2','/v1/sys/menu/getMenu','GET','','',''),(287,'p','3','/v1/sys/app-members/grants','GET','','',''),(288,'p','3','/v1/sys/app-members/plans','GET','','',''),(289,'p','3','/v1/sys/app-members/sessions','GET','','',''),(290,'p','3','/v1/sys/app-members/users','GET','','',''),(291,'p','3','/v1/sys/menu/getMenu','GET','','','');
/*!40000 ALTER TABLE `casbin_rule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `schema_migrations`
--

DROP TABLE IF EXISTS `schema_migrations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `schema_migrations` (
  `filename` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `checksum` char(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `applied_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`filename`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `schema_migrations`
--

LOCK TABLES `schema_migrations` WRITE;
/*!40000 ALTER TABLE `schema_migrations` DISABLE KEYS */;
INSERT INTO `schema_migrations` VALUES ('20260925_business_access.sql','af8cb82d64db93891146c3c69dfcaf0ca0eff96f861a2b8ce74f050204639659','2026-10-07 15:43:08'),('20260925_user_active_username.sql','cf2440bfd60941cbec000dd9eef18e517d0a305ee1ea126cd349cef8ee1d4b7c','2026-10-07 15:43:09'),('20260926_00_method_dictionary.sql','399b0b3901b6f2e7edb71d042f4738379164a8686ec2242a1154a7faab671dba','2026-10-07 15:43:10'),('20260926_api_sync_access.sql','287f534292e3643e23f3c450c47aa5420e7416b4c0b5d1564ca669144481bb9b','2026-10-07 15:43:11'),('20260926_business_unique.sql','5baeac3baed30fae0a4c3ed9b81e8ca4a6ca5ecaf76caeb8017f720684a16aa6','2026-10-07 15:43:13'),('20260926_session_version.sql','9898ed43d66cf2e842e7cd70ca6ccaedcbb0e2964c07cacf5a6eb013f2d3c9ba','2026-10-07 15:43:14'),('20261002_00_policy_sync.sql','bf40d0fa1ef28e3c4e175dddbbe00ae875df7ed0e471f0dfea4fc757e4b4ae73','2026-10-07 15:43:15'),('20261002_01_audit.sql','06d7fde9ac5519750f070fa592e0d49d23d522f939096f6989cb5fb2352fcdea','2026-10-07 15:43:16'),('20261002_02_organization.sql','2a39b8e7682539b92c24b32d18eed8e98eb97bd63ca0a8029ce35832566c6f40','2026-10-07 15:43:18'),('20261002_03_files.sql','6600329d92643c909eff06848d7944b8d65b0b66a91eb77c5a648f806a957f8e','2026-10-07 15:43:19'),('20261002_04_device_sessions.sql','06dcc6a7e7f263f34b5d0373d8eb1bcde40cf91bdb237ff448ee6a752e1a39b4','2026-10-07 15:43:20'),('20261002_05_module_access.sql','cfadb622d1002666708d0cff4a35bb666f051fab81d3440de6eb5f1da11b3e0d','2026-10-07 15:43:21'),('20261003_frontend_modules.sql','e62dc29eb9d409d949c24c7387d2e10deab064ccee2f17abc3da5b48dca7c9fa','2026-10-07 15:43:23'),('20261003_frontend_routes_compat.sql','aaffb0a8d4f204b4a0f94e5c54b10c33fc2a4772d7d989eeef1cf159c4007b2e','2026-10-07 15:43:24'),('20261003_zz_ai_agent.sql','9865c0e14f334a3ae7ee0b861363cea6b0eba97ae70db72a32e33649a378ee6e','2026-10-07 15:43:25'),('20261008_permission_revision_history.sql','bc8c6e02847cc29f0dcabff6098326e3854a37303c28ca9cc03e7cbeb58ebfe9','2026-10-20 22:56:11'),('20261019_app_members.sql','d6b4f5ba9cdad5192f4234486d3bbe28e81cefde16c001601fb16a101f14bf50','2026-10-20 22:56:12'),('20261019_app_registration_ip.sql','96b60c1c9ee4582b55eb8f7a46d9413af8c5b1700e7f7fc88f7118dfd31ebf12','2026-10-20 22:56:13'),('20261020_rustfs_avatar.sql','b372a186d2f65de8f75a98fa223689e4c0c0fb414acffa94d4fdf786722960e2','2026-10-20 22:56:14'),('20261020_user_avatar_references.sql','feb049cb6932a6bf16d7066381361c944d425fb6049907eb27c4ccb39301eeb9','2026-10-20 22:56:14');
/*!40000 ALTER TABLE `schema_migrations` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_ai_conversations`
--

DROP TABLE IF EXISTS `sys_ai_conversations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_ai_conversations` (
  `id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `owner_id` bigint NOT NULL,
  `title` varchar(160) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `next_sequence` bigint NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_agent_conversation_owner` (`owner_id`,`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_ai_conversations`
--

LOCK TABLES `sys_ai_conversations` WRITE;
/*!40000 ALTER TABLE `sys_ai_conversations` DISABLE KEYS */;
INSERT INTO `sys_ai_conversations` VALUES ('d0000000-0000-4000-8000-000000000001',1,'【内置演示】后台功能导览',1,'2026-10-20 14:55:14.587','2026-10-20 14:56:14.587');
/*!40000 ALTER TABLE `sys_ai_conversations` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_ai_messages`
--

DROP TABLE IF EXISTS `sys_ai_messages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_ai_messages` (
  `id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `owner_id` bigint NOT NULL,
  `conversation_id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `run_id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `role` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `content` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_message_run_role` (`run_id`,`role`),
  KEY `idx_agent_message_history` (`conversation_id`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_ai_messages`
--

LOCK TABLES `sys_ai_messages` WRITE;
/*!40000 ALTER TABLE `sys_ai_messages` DISABLE KEYS */;
INSERT INTO `sys_ai_messages` VALUES ('d0000000-0000-4000-8000-000000000004',1,'d0000000-0000-4000-8000-000000000001','d0000000-0000-4000-8000-000000000002','user','【内置演示】第一次使用这个后台，可以从哪些页面开始体验？','2026-10-20 14:55:14.587'),('d0000000-0000-4000-8000-000000000005',1,'d0000000-0000-4000-8000-000000000001','d0000000-0000-4000-8000-000000000002','assistant','## 内置演示：开始体验\n\n这段对话是随初始化数据库提供的静态示例，未调用云模型、未执行工具，也未产生模型费用。\n\n| 页面 | 可以体验的内容 |\n| --- | --- |\n| 用户管理、角色管理 | 比较管理员、运营人员和只读观察员的菜单、按钮及接口权限。 |\n| App 用户、App 会员 | 查看普通、会员和待审核示例账号，以及会员权益和到期时间。 |\n| 文件资源 | 上传公开或私有图片，预览、下载，并体验头像引用与删除保护。 |\n| 操作审计 | 查看标注为“内置演示”的成功和拒绝记录，尝试筛选与详情。 |\n\n配置并启用模型后，只有你主动发送新问题才会创建新的 AI 任务；当前历史示例不会自动运行。','2026-10-20 14:56:14.587');
/*!40000 ALTER TABLE `sys_ai_messages` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_ai_runs`
--

DROP TABLE IF EXISTS `sys_ai_runs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_ai_runs` (
  `id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `owner_id` bigint NOT NULL,
  `authority_id` bigint NOT NULL,
  `session_id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '',
  `session_version` bigint NOT NULL,
  `request_id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `request_hash` char(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `conversation_id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `question` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `sequence` bigint NOT NULL,
  `status` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `cancel_requested` tinyint(1) NOT NULL DEFAULT '0',
  `text` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `provider` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `model` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `input_tokens` bigint NOT NULL DEFAULT '0',
  `output_tokens` bigint NOT NULL DEFAULT '0',
  `extra_json` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `error_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `error` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `lease_owner` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '',
  `lease_until` datetime(3) DEFAULT NULL,
  `queue_expires_at` datetime(3) NOT NULL,
  `deadline_at` datetime(3) DEFAULT NULL,
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_run_request` (`owner_id`,`request_id`),
  UNIQUE KEY `uk_agent_run_sequence` (`conversation_id`,`sequence`),
  KEY `idx_agent_run_owner_status` (`owner_id`,`status`),
  KEY `idx_agent_run_conversation_status` (`conversation_id`,`status`),
  KEY `idx_agent_run_claim` (`status`,`queue_expires_at`),
  KEY `idx_agent_run_lease` (`lease_until`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_ai_runs`
--

LOCK TABLES `sys_ai_runs` WRITE;
/*!40000 ALTER TABLE `sys_ai_runs` DISABLE KEYS */;
INSERT INTO `sys_ai_runs` VALUES ('d0000000-0000-4000-8000-000000000002',1,1,'',1,'d0000000-0000-4000-8000-000000000003','6656388b1c0862884160d8b34ce467a8ecfd84d7dc8f47936c5798ac813023b1','d0000000-0000-4000-8000-000000000001','【内置演示】第一次使用这个后台，可以从哪些页面开始体验？',1,'succeeded',0,'## 内置演示：开始体验\n\n这段对话是随初始化数据库提供的静态示例，未调用云模型、未执行工具，也未产生模型费用。\n\n| 页面 | 可以体验的内容 |\n| --- | --- |\n| 用户管理、角色管理 | 比较管理员、运营人员和只读观察员的菜单、按钮及接口权限。 |\n| App 用户、App 会员 | 查看普通、会员和待审核示例账号，以及会员权益和到期时间。 |\n| 文件资源 | 上传公开或私有图片，预览、下载，并体验头像引用与删除保护。 |\n| 操作审计 | 查看标注为“内置演示”的成功和拒绝记录，尝试筛选与详情。 |\n\n配置并启用模型后，只有你主动发送新问题才会创建新的 AI 任务；当前历史示例不会自动运行。','内置演示','static-demo',0,0,'[]','','','',NULL,'2026-10-20 14:56:14.587',NULL,'2026-10-20 14:55:14.587','2026-10-20 14:56:14.587','2026-10-20 14:55:14.587','2026-10-20 14:56:14.587');
/*!40000 ALTER TABLE `sys_ai_runs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_apis`
--

DROP TABLE IF EXISTS `sys_apis`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_apis` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `path` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'api路径',
  `description` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'api中文描述',
  `api_group` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'api组',
  `method` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'POST' COMMENT '方法',
  `active_unique` tinyint GENERATED ALWAYS AS ((case when (`deleted_at` is null) then 1 else NULL end)) STORED,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_sys_apis_active_route` (`path`,`method`,`active_unique`),
  KEY `idx_sys_apis_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=94 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_apis`
--

LOCK TABLES `sys_apis` WRITE;
/*!40000 ALTER TABLE `sys_apis` DISABLE KEYS */;
INSERT INTO `sys_apis` (`id`, `created_at`, `updated_at`, `deleted_at`, `path`, `description`, `api_group`, `method`) VALUES (1,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/login','系统用户登录','系统-无需权限','POST'),(2,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/randomImage','获取验证码','系统-无需权限','GET'),(3,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/getUserList','分页获取用户列表','用户','GET'),(4,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/register','新增（注册）用户 - 管理员','用户','POST'),(5,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/updateUserInfo','修改用户信息','用户','PUT'),(6,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/resetUserPassword','重置用户密码 默认密码：goZero','用户','PUT'),(7,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/deleteUser','删除用户','用户','DELETE'),(8,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenu','获取菜单','菜单','GET'),(9,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenuList','分页获取base_menu列表','菜单','GET'),(10,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/addBaseMenu','新增 base_menu','菜单','POST'),(11,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getBaseMenuTree','获取用户动态路由树  -- 用于角色管理的设置权限','菜单','GET'),(12,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenuAuthority','获取指定角色menu  -- 用于角色管理的设置权限','菜单','GET'),(13,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getBaseMenuById','根据id获取菜单','菜单','GET'),(14,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/updateBaseMenu','更新系统菜单','菜单','PUT'),(15,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/deleteBaseMenu','删除系统菜单','菜单','DELETE'),(16,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/getAuthorityList','获取角色列表','角色','GET'),(17,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/addAuthorityMenu','增加角色和base_menu关联关系','角色','POST'),(18,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/updateAuthority','更新角色信息','角色','PUT'),(19,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/createAuthority','创建角色','角色','POST'),(20,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/deleteAuthority','删除角色','角色','DELETE'),(21,'2024-01-15 13:14:52.000','2024-01-15 10:35:23.438',NULL,'/v1/sys/api/getApiList','获取api列表','api','GET'),(22,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/createApi','创建/增加 api列表','api','POST'),(23,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/deleteApi','删除 api列表','api','DELETE'),(24,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/getAllApiList','获取 所有api','api','GET'),(25,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/deleteApisByIds','删除多条api','api','DELETE'),(26,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/updateApi','更新api','api','PUT'),(27,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryList','获取SysDictionary列表','字典','GET'),(28,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/createSysDictionary','新建SysDictionary','字典','POST'),(29,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryDetails','根据ID或者type获取SysDictionary','字典','GET'),(30,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/updateSysDictionary','更新SysDictionary','字典','PUT'),(31,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/deleteSysDictionary','删除SysDictionary','字典','DELETE'),(32,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoList','获取SysDictionaryInfo列表','字典','GET'),(33,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','根据id获取SysDictionaryInfo详情','字典','GET'),(34,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/updateSysDictionaryInfo','更新SysDictionaryInfo','字典','PUT'),(35,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/createSysDictionaryInfo','创建SysDictionaryInfo','字典','POST'),(36,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/deleteSysDictionaryInfo','删除SysDictionaryInfo','字典','DELETE'),(37,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/getPathByAuthorityId','根据角色id获取对应的casbin数据','casbin','GET'),(38,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/updateCasbinData','更新一个角色的对应的casbin数据','casbin','PUT'),(39,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/updateCasbinDataByApiIds','更新一个角色的对应的casbin数据 用api的ids 查数据','casbin','PUT'),(40,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/base/uploadFileImg','上传图片','base','POST'),(41,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','根据Type获取SysDictionaryInfo详情','字典','GET'),(42,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/base/sendEmailCode','邮箱发送验证码 - 暂时用于注册账号','base','POST'),(43,'2026-10-07 15:43:07.889','2026-10-07 15:43:07.889',NULL,'/v1/sys/menu/getAuthorityButtons','角色按钮权限','系统管理','GET'),(44,'2026-10-07 15:43:07.889','2026-10-07 15:43:07.889',NULL,'/v1/sys/menu/updateAuthorityButtons','保存按钮授权','系统管理','PUT'),(46,'2026-10-07 15:43:11.589','2026-10-07 15:43:11.589',NULL,'/v1/sys/api/previewSync','预览Swagger与API资源差异，失效资源仅提示','api','GET'),(47,'2026-10-07 15:43:11.590','2026-10-07 15:43:11.590',NULL,'/v1/sys/api/applySync','按预览版本同步选中API，仅新增资源或更新分组描述，保留授权','api','POST'),(48,'2026-10-07 15:43:16.664','2026-10-07 15:43:16.664',NULL,'/v1/sys/audit/getAuditLogList','分页查询操作审计与登录日志','audit','GET'),(49,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/list','按数据范围查询文件资源','files','GET'),(50,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/url','获取公开或短时签名文件地址','files','GET'),(51,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/resource','删除未被引用的资源，失败可重试','files','DELETE'),(52,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/reference','登记文件业务引用，仅内置管理员','files','POST'),(53,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/reference','移除文件业务引用，仅内置管理员','files','DELETE'),(56,'2026-10-07 15:43:21.706','2026-10-07 15:43:21.706',NULL,'/v1/sys/organization/departments','查询部门树','organization','GET'),(57,'2026-10-07 15:43:21.707','2026-10-07 15:43:21.707',NULL,'/v1/sys/organization/departments','创建部门','organization','POST'),(58,'2026-10-07 15:43:21.708','2026-10-07 15:43:21.708',NULL,'/v1/sys/organization/departments','更新部门','organization','PUT'),(59,'2026-10-07 15:43:21.708','2026-10-07 15:43:21.708',NULL,'/v1/sys/organization/departments','删除部门','organization','DELETE'),(60,'2026-10-07 15:43:21.709','2026-10-07 15:43:21.709',NULL,'/v1/sys/organization/positions','查询岗位','organization','GET'),(61,'2026-10-07 15:43:21.710','2026-10-07 15:43:21.710',NULL,'/v1/sys/organization/positions','创建岗位','organization','POST'),(62,'2026-10-07 15:43:21.710','2026-10-07 15:43:21.710',NULL,'/v1/sys/organization/positions','更新岗位','organization','PUT'),(63,'2026-10-07 15:43:21.711','2026-10-07 15:43:21.711',NULL,'/v1/sys/organization/positions','删除岗位','organization','DELETE'),(64,'2026-10-07 15:43:21.712','2026-10-07 15:43:21.712',NULL,'/v1/sys/organization/membership','查询用户部门岗位','organization','GET'),(65,'2026-10-07 15:43:21.712','2026-10-07 15:43:21.712',NULL,'/v1/sys/organization/membership','更新用户部门岗位','organization','PUT'),(66,'2026-10-07 15:43:21.713','2026-10-07 15:43:21.713',NULL,'/v1/sys/organization/dataScope','查询角色数据范围','organization','GET'),(67,'2026-10-07 15:43:21.713','2026-10-07 15:43:21.713',NULL,'/v1/sys/organization/dataScope','更新角色数据范围','organization','PUT'),(68,'2026-10-07 15:43:21.716','2026-10-07 15:43:21.716',NULL,'/v1/sys/session/admin/devices','查询用户设备会话','session','GET'),(69,'2026-10-07 15:43:21.717','2026-10-07 15:43:21.717',NULL,'/v1/sys/session/admin/device','撤销用户设备会话','session','DELETE'),(70,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/info','查询AI助手配置','ai-agent','GET'),(71,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/runs','提交AI任务','ai-agent','POST'),(72,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/runs/:id','查询本人AI任务','ai-agent','GET'),(73,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/runs/:id/cancel','取消本人AI任务','ai-agent','POST'),(74,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/conversations','查询本人AI会话','ai-agent','GET'),(75,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/conversations/:id/messages','查询本人AI会话消息','ai-agent','GET'),(76,'2026-10-20 22:56:11.425','2026-10-20 22:56:11.425',NULL,'/v1/sys/permissions/edit','读取四类授权的统一资源与选择快照','permission','GET'),(77,'2026-10-20 22:56:11.426','2026-10-20 22:56:11.426',NULL,'/v1/sys/menu/previewMove','预览菜单移动新增可见入口','menu','POST'),(78,'2026-10-20 22:56:12.756','2026-10-20 22:56:12.756',NULL,'/v1/sys/app-members/users','查询 App 用户','appmembers','GET'),(79,'2026-10-20 22:56:12.756','2026-10-20 22:56:12.756',NULL,'/v1/sys/app-members/status','审核或调整 App 账号状态','appmembers','PUT'),(80,'2026-10-20 22:56:12.756','2026-10-20 22:56:12.756',NULL,'/v1/sys/app-members/sessions','查询 App 设备会话','appmembers','GET'),(81,'2026-10-20 22:56:12.756','2026-10-20 22:56:12.756',NULL,'/v1/sys/app-members/session','撤销 App 设备会话','appmembers','DELETE'),(82,'2026-10-20 22:56:12.756','2026-10-20 22:56:12.756',NULL,'/v1/sys/app-members/plans','查询会员方案','appmembers','GET'),(83,'2026-10-20 22:56:12.756','2026-10-20 22:56:12.756',NULL,'/v1/sys/app-members/grants','查询会员授予','appmembers','GET'),(84,'2026-10-20 22:56:12.756','2026-10-20 22:56:12.756',NULL,'/v1/sys/app-members/grants','开通或续期会员','appmembers','POST'),(85,'2026-10-20 22:56:12.756','2026-10-20 22:56:12.756',NULL,'/v1/sys/app-members/grant','撤销会员授予','appmembers','DELETE'),(93,'2026-10-20 22:56:14.493','2026-10-20 22:56:14.493',NULL,'/v1/sys/app-members/avatar','设置 App 用户头像','appmembers','PUT');
/*!40000 ALTER TABLE `sys_apis` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_audit_logs`
--

DROP TABLE IF EXISTS `sys_audit_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_audit_logs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `actor_id` bigint NOT NULL DEFAULT '0',
  `actor_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `authority_id` bigint NOT NULL DEFAULT '0',
  `event_type` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `module` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `action` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `object` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `path` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `method` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `result` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `status_code` bigint NOT NULL DEFAULT '0',
  `ip` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `trace_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `duration_ms` bigint NOT NULL DEFAULT '0',
  `params` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_audit_time` (`created_at`),
  KEY `idx_audit_actor_time` (`actor_id`,`created_at`),
  KEY `idx_audit_type_time` (`event_type`,`created_at`),
  KEY `idx_audit_module_time` (`module`,`created_at`),
  KEY `idx_audit_result_time` (`result`,`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_audit_logs`
--

LOCK TABLES `sys_audit_logs` WRITE;
/*!40000 ALTER TABLE `sys_audit_logs` DISABLE KEYS */;
INSERT INTO `sys_audit_logs` VALUES (1,1,'admin',1,'operation','user','内置演示：账号管理记录','内置演示：后台运营账号','/v1/sys/updateUserInfo','PUT','success',200,'192.0.2.10','',0,'{\"enable\": 1, \"userid\": 2}','2026-10-20 14:53:14.587'),(2,1,'admin',1,'operation','files','内置演示：文件管理记录','内置演示：无真实文件操作','/v1/sys/base/uploadFileImg','POST','success',200,'192.0.2.10','',0,'{}','2026-10-20 14:54:14.587'),(3,2,'operator',2,'operation','authority','内置演示：越权操作被拒绝','内置演示：系统角色配置','/v1/sys/authority/updateAuthority','PUT','failure',403,'192.0.2.20','',0,'{\"authorityid\": 1}','2026-10-20 14:55:14.587');
/*!40000 ALTER TABLE `sys_audit_logs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_authorities`
--

DROP TABLE IF EXISTS `sys_authorities`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_authorities` (
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `authority_id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '角色ID',
  `authority_name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '角色名',
  `parent_id` bigint unsigned DEFAULT NULL COMMENT '父角色ID',
  `default_router` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'index' COMMENT '默认菜单',
  PRIMARY KEY (`authority_id`) USING BTREE,
  UNIQUE KEY `authority_id` (`authority_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_authorities`
--

LOCK TABLES `sys_authorities` WRITE;
/*!40000 ALTER TABLE `sys_authorities` DISABLE KEYS */;
INSERT INTO `sys_authorities` VALUES ('2024-01-15 13:14:52.000','2024-01-16 17:49:10.662',NULL,1,'超级管理员',0,'index'),('2026-10-20 14:56:14.543','2026-10-20 14:56:14.543',NULL,2,'运营人员',0,'index'),('2026-10-20 14:56:14.543','2026-10-20 14:56:14.543',NULL,3,'只读观察员',0,'index');
/*!40000 ALTER TABLE `sys_authorities` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_authority_btns`
--

DROP TABLE IF EXISTS `sys_authority_btns`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_authority_btns` (
  `authority_id` bigint unsigned DEFAULT NULL COMMENT '角色ID',
  `sys_menu_id` bigint unsigned DEFAULT NULL COMMENT '菜单ID',
  `sys_base_menu_btn_id` bigint unsigned DEFAULT NULL COMMENT '菜单按钮ID'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_authority_btns`
--

LOCK TABLES `sys_authority_btns` WRITE;
/*!40000 ALTER TABLE `sys_authority_btns` DISABLE KEYS */;
INSERT INTO `sys_authority_btns` VALUES (1,5,23),(1,5,24),(1,5,25),(1,2,26),(1,2,27),(1,2,28),(1,3,29),(1,3,30),(1,3,31),(1,4,32),(1,4,33),(1,4,34),(1,6,35),(1,6,36),(1,6,37),(1,5,38),(1,2,39),(1,2,40),(1,2,41),(1,6,42),(1,44,43),(1,44,44),(1,4,54),(1,46,55),(1,46,56),(1,46,57),(1,49,58),(1,47,59),(1,47,60),(1,47,61),(1,48,62),(1,48,63),(1,48,64),(1,5,65),(1,2,66),(1,52,70),(1,52,71),(1,53,72),(1,53,74),(1,53,73),(1,54,75),(1,54,76),(1,53,79),(2,53,79),(2,53,72),(2,53,73),(2,44,43),(2,46,55);
/*!40000 ALTER TABLE `sys_authority_btns` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_authority_menus`
--

DROP TABLE IF EXISTS `sys_authority_menus`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_authority_menus` (
  `sys_base_menu_id` bigint unsigned NOT NULL,
  `sys_authority_authority_id` bigint unsigned NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_base_menu_id`,`sys_authority_authority_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_authority_menus`
--

LOCK TABLES `sys_authority_menus` WRITE;
/*!40000 ALTER TABLE `sys_authority_menus` DISABLE KEYS */;
INSERT INTO `sys_authority_menus` VALUES (1,1),(1,2),(2,1),(3,1),(4,1),(5,1),(6,1),(7,1),(7,2),(7,3),(44,1),(44,2),(45,1),(46,1),(46,2),(47,1),(48,1),(49,1),(52,1),(53,1),(53,2),(53,3),(54,1),(54,2),(54,3);
/*!40000 ALTER TABLE `sys_authority_menus` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_base_menu_btns`
--

DROP TABLE IF EXISTS `sys_base_menu_btns`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_base_menu_btns` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '按钮关键key',
  `desc` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `sys_base_menu_id` bigint unsigned DEFAULT NULL COMMENT '菜单ID',
  `permission_key` varchar(383) COLLATE utf8mb4_general_ci NOT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `idx_sys_base_menu_btns_permission_key` (`permission_key`),
  KEY `idx_sys_base_menu_btns_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=80 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_base_menu_btns`
--

LOCK TABLES `sys_base_menu_btns` WRITE;
/*!40000 ALTER TABLE `sys_base_menu_btns` DISABLE KEYS */;
INSERT INTO `sys_base_menu_btns` VALUES (23,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',5,'user:create'),(24,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',5,'user:update'),(25,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',5,'user:delete'),(26,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',2,'authority:create'),(27,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',2,'authority:update'),(28,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',2,'authority:delete'),(29,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',3,'menu:create'),(30,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',3,'menu:update'),(31,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',3,'menu:delete'),(32,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',4,'api:create'),(33,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',4,'api:update'),(34,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',4,'api:delete'),(35,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',6,'dictionary:create'),(36,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',6,'dictionary:update'),(37,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',6,'dictionary:delete'),(38,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'resetPassword','重置密码',5,'user:resetPassword'),(39,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'menus','菜单授权',2,'authority:menus'),(40,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'apis','接口授权',2,'authority:apis'),(41,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'buttons','按钮授权',2,'authority:buttons'),(42,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'items','字典项管理',6,'dictionary:items'),(43,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'upload','上传图片',44,'businessTools:upload'),(44,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'sendEmail','发送邮箱验证码',44,'businessTools:sendEmail'),(54,'2026-10-07 15:43:11.591','2026-10-07 15:43:11.591',NULL,'sync','同步接口资源',4,'api:sync'),(55,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'upload','上传文件',46,'files:upload'),(56,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'delete','删除文件',46,'files:delete'),(57,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'reference','管理文件引用',46,'files:reference'),(58,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'revoke','撤销设备会话',49,'sessions:revoke'),(59,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'create','新增部门',47,'organization-departments:create'),(60,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'update','编辑部门',47,'organization-departments:update'),(61,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'delete','删除部门',47,'organization-departments:delete'),(62,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'create','新增岗位',48,'organization-positions:create'),(63,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'update','编辑岗位',48,'organization-positions:update'),(64,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'delete','删除岗位',48,'organization-positions:delete'),(65,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'membership','分配部门岗位',5,'user:membership'),(66,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'dataScope','设置数据权限',2,'authority:dataScope'),(70,'2026-10-07 15:43:25.300','2026-10-07 15:43:25.300',NULL,'run','提交AI任务',52,'ai-agent:run'),(71,'2026-10-07 15:43:25.300','2026-10-07 15:43:25.300',NULL,'cancel','取消AI任务',52,'ai-agent:cancel'),(72,'2026-10-20 22:56:12.755','2026-10-20 22:56:12.755',NULL,'review','审核 App 账号',53,'app-members-users:review'),(73,'2026-10-20 22:56:12.755','2026-10-20 22:56:12.755',NULL,'status','禁用或恢复 App 账号',53,'app-members-users:status'),(74,'2026-10-20 22:56:12.755','2026-10-20 22:56:12.755',NULL,'revoke-session','撤销 App 设备',53,'app-members-users:revoke-session'),(75,'2026-10-20 22:56:12.755','2026-10-20 22:56:12.755',NULL,'grant','开通或续期会员',54,'app-memberships:grant'),(76,'2026-10-20 22:56:12.755','2026-10-20 22:56:12.755',NULL,'revoke','撤销会员授予',54,'app-memberships:revoke'),(79,'2026-10-20 22:56:14.492','2026-10-20 22:56:14.492',NULL,'avatar','设置 App 用户头像',53,'app-members-users:avatar');
/*!40000 ALTER TABLE `sys_base_menu_btns` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_base_menu_parameters`
--

DROP TABLE IF EXISTS `sys_base_menu_parameters`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_base_menu_parameters` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `sys_base_menu_id` bigint unsigned DEFAULT NULL,
  `type` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '地址栏携带参数为params还是query',
  `key` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '地址栏携带参数的key',
  `value` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '地址栏携带参数的值',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_sys_base_menu_parameters_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_base_menu_parameters`
--

LOCK TABLES `sys_base_menu_parameters` WRITE;
/*!40000 ALTER TABLE `sys_base_menu_parameters` DISABLE KEYS */;
/*!40000 ALTER TABLE `sys_base_menu_parameters` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_base_menus`
--

DROP TABLE IF EXISTS `sys_base_menus`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_base_menus` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `menu_level` bigint unsigned DEFAULT NULL,
  `parent_id` bigint DEFAULT NULL COMMENT '父菜单ID',
  `path` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '路由path',
  `name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '路由name',
  `hidden` tinyint(1) DEFAULT NULL COMMENT '是否在列表隐藏',
  `component` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '对应前端文件路径',
  `sort` bigint DEFAULT NULL COMMENT '排序标记',
  `active_name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '附加属性',
  `keep_alive` tinyint(1) DEFAULT NULL COMMENT '附加属性',
  `default_menu` tinyint(1) DEFAULT NULL COMMENT '附加属性',
  `title` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '附加属性',
  `icon` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '附加属性',
  `close_tab` tinyint(1) DEFAULT NULL COMMENT '附加属性',
  `active_unique` tinyint GENERATED ALWAYS AS ((case when (`deleted_at` is null) then 1 else NULL end)) STORED,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_sys_menus_active_name` (`name`,`active_unique`),
  UNIQUE KEY `uk_sys_menus_active_path` (`path`,`active_unique`),
  KEY `idx_sys_base_menus_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=56 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=REDUNDANT;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_base_menus`
--

LOCK TABLES `sys_base_menus` WRITE;
/*!40000 ALTER TABLE `sys_base_menus` DISABLE KEYS */;
INSERT INTO `sys_base_menus` (`id`, `created_at`, `updated_at`, `deleted_at`, `menu_level`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`) VALUES (1,'2023-11-17 15:00:25.000','2024-01-18 09:54:30.123',NULL,0,0,'admin','superAdmin',0,'views/superAdmin/index.vue',10,'',0,0,'系统管理','lucide:settings',0),(2,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,1,1,'authority','authority',0,'views/superAdmin/authority/authority.vue',20,'',0,0,'角色管理','lucide:shield-check',0),(3,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,1,1,'menu','menu',0,'views/superAdmin/menu/menu.vue',50,'',1,0,'菜单管理','lucide:list-tree',0),(4,'2023-11-17 15:00:25.093','2024-01-10 11:41:33.339',NULL,1,1,'api','api',0,'views/superAdmin/api/api.vue',60,'',1,0,'API 管理','lucide:network',0),(5,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,1,1,'user','user',0,'views/superAdmin/user/user.vue',10,'',0,0,'用户管理','lucide:users',0),(6,'2023-11-17 15:00:25.093','2024-01-12 10:03:17.383',NULL,1,1,'dictionary','dictionary',0,'views/superAdmin/dictionary/dictionary.vue',70,'',0,0,'字典管理','lucide:book-open',0),(7,'2024-01-16 17:27:17.123','2024-01-18 09:54:26.485',NULL,0,0,'index','index',0,'views/index.vue',0,'',0,0,'首页','lucide:house',0),(44,'2026-10-07 15:43:07.887','2026-10-07 15:43:07.887',NULL,1,1,'business-tools','businessTools',0,'views/business/tools/index.vue',120,'',0,0,'管理工具','lucide:wrench',0),(45,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,1,1,'audit','audit',0,'views/business/system/audit/index.vue',100,'',0,0,'操作审计','lucide:clipboard-list',0),(46,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,1,1,'files','files',0,'views/business/system/files/index.vue',80,'',0,0,'文件资源','lucide:folder-open',0),(47,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,1,1,'organization/departments','organization-departments',0,'views/business/system/organization/departments.vue',30,'',0,0,'部门管理','lucide:building-2',0),(48,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,1,1,'organization/positions','organization-positions',0,'views/business/system/organization/positions.vue',40,'',0,0,'岗位管理','lucide:briefcase-business',0),(49,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,1,1,'sessions','sessions',0,'views/business/system/sessions/index.vue',90,'',0,0,'在线会话','lucide:monitor',0),(52,'2026-10-07 15:43:25.297','2026-10-07 15:43:25.297',NULL,1,1,'ai-agent','ai-agent',0,'views/business/ai-agent/index.vue',110,'',0,0,'AI 助手','lucide:bot',0),(53,'2026-10-20 22:56:12.755','2026-10-20 22:56:12.755',NULL,0,0,'app-members/users','app-members-users',0,'views/business/app-members/users.vue',20,'',0,0,'App 用户','lucide:contact',0),(54,'2026-10-20 22:56:12.755','2026-10-20 22:56:12.755',NULL,0,0,'app-members/memberships','app-memberships',0,'views/business/app-members/memberships.vue',30,'',0,0,'App 会员','lucide:crown',0);
/*!40000 ALTER TABLE `sys_base_menus` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_data_authority_id`
--

DROP TABLE IF EXISTS `sys_data_authority_id`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_data_authority_id` (
  `sys_authority_authority_id` bigint unsigned NOT NULL COMMENT '角色ID',
  `data_authority_id_authority_id` bigint unsigned NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_authority_authority_id`,`data_authority_id_authority_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_data_authority_id`
--

LOCK TABLES `sys_data_authority_id` WRITE;
/*!40000 ALTER TABLE `sys_data_authority_id` DISABLE KEYS */;
/*!40000 ALTER TABLE `sys_data_authority_id` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_departments`
--

DROP TABLE IF EXISTS `sys_departments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_departments` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  `parent_id` bigint NOT NULL DEFAULT '0',
  `name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `code` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `sort` bigint NOT NULL DEFAULT '0',
  `status` bigint NOT NULL DEFAULT '1',
  `leader` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_department_code` (`code`),
  KEY `idx_department_parent` (`parent_id`),
  KEY `idx_department_deleted` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_departments`
--

LOCK TABLES `sys_departments` WRITE;
/*!40000 ALTER TABLE `sys_departments` DISABLE KEYS */;
INSERT INTO `sys_departments` VALUES (1,'2026-10-20 22:56:14.547','2026-10-20 22:56:14.547',NULL,0,'示例组织','DEMO',10,1,''),(2,'2026-10-20 22:56:14.547','2026-10-20 22:56:14.547',NULL,1,'运营部','DEMO_OPS',10,1,''),(3,'2026-10-20 22:56:14.547','2026-10-20 22:56:14.547',NULL,1,'研发部','DEMO_DEV',20,1,'');
/*!40000 ALTER TABLE `sys_departments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_device_sessions`
--

DROP TABLE IF EXISTS `sys_device_sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_device_sessions` (
  `id` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `user_id` bigint NOT NULL,
  `authority_id` bigint NOT NULL,
  `session_version` bigint NOT NULL,
  `ip` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `user_agent` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `revoked_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_device_user` (`user_id`,`expires_at`),
  KEY `idx_device_expiry` (`expires_at`),
  KEY `idx_device_revoked` (`revoked_at`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_device_sessions`
--

LOCK TABLES `sys_device_sessions` WRITE;
/*!40000 ALTER TABLE `sys_device_sessions` DISABLE KEYS */;
/*!40000 ALTER TABLE `sys_device_sessions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_dictionaries`
--

DROP TABLE IF EXISTS `sys_dictionaries`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_dictionaries` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '字典名（中）',
  `type` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '字典名（英）',
  `status` tinyint(1) DEFAULT NULL COMMENT '状态',
  `desc` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '描述',
  `active_unique` tinyint GENERATED ALWAYS AS ((case when (`deleted_at` is null) then 1 else NULL end)) STORED,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_sys_dictionaries_active_type` (`type`,`active_unique`),
  KEY `idx_sys_dictionaries_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dictionaries`
--

LOCK TABLES `sys_dictionaries` WRITE;
/*!40000 ALTER TABLE `sys_dictionaries` DISABLE KEYS */;
INSERT INTO `sys_dictionaries` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `type`, `status`, `desc`) VALUES (1,'2023-11-17 15:00:24.877','2023-11-17 15:00:24.896',NULL,'性别','gender',1,'性别字典'),(2,'2024-01-16 17:38:10.566','2024-01-16 17:38:10.566',NULL,'API 请求方法','method',1,'HTTP 请求方法');
/*!40000 ALTER TABLE `sys_dictionaries` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_dictionary_info`
--

DROP TABLE IF EXISTS `sys_dictionary_info`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_dictionary_info` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `label` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '展示值',
  `value` bigint DEFAULT NULL COMMENT '字典值',
  `extend` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '扩展值',
  `status` tinyint(1) DEFAULT NULL COMMENT '启用状态',
  `sort` bigint DEFAULT NULL COMMENT '排序标记',
  `sys_dictionary_id` bigint unsigned DEFAULT NULL COMMENT '关联标记',
  `active_unique` tinyint GENERATED ALWAYS AS ((case when (`deleted_at` is null) then 1 else NULL end)) STORED,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_sys_dictionary_info_active_value` (`sys_dictionary_id`,`value`,`active_unique`),
  KEY `idx_sys_dictionary_details_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dictionary_info`
--

LOCK TABLES `sys_dictionary_info` WRITE;
/*!40000 ALTER TABLE `sys_dictionary_info` DISABLE KEYS */;
INSERT INTO `sys_dictionary_info` (`id`, `created_at`, `updated_at`, `deleted_at`, `label`, `value`, `extend`, `status`, `sort`, `sys_dictionary_id`) VALUES (1,'2024-01-16 17:37:18.206','2024-01-16 17:37:18.206',NULL,'男',1,'男-拓展值',1,10,1),(2,'2024-01-16 17:37:35.413','2024-01-16 17:37:35.413',NULL,'女',2,'女-拓展值',1,20,1),(3,'2024-01-16 17:38:19.939','2024-01-18 17:26:35.798',NULL,'POST',1,'POST',1,20,2),(4,'2024-01-16 17:38:36.624','2024-01-16 17:38:36.624',NULL,'GET',2,'GET',1,10,2),(5,'2024-01-16 17:38:45.246','2024-01-16 17:38:45.246',NULL,'PUT',3,'PUT',1,30,2),(6,'2024-01-16 17:38:56.280','2024-01-16 17:38:56.280',NULL,'DELETE',4,'DELETE',1,40,2);
/*!40000 ALTER TABLE `sys_dictionary_info` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_file_references`
--

DROP TABLE IF EXISTS `sys_file_references`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_file_references` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `file_id` bigint NOT NULL,
  `object_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `object_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_file_reference` (`file_id`,`object_type`,`object_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_file_references`
--

LOCK TABLES `sys_file_references` WRITE;
/*!40000 ALTER TABLE `sys_file_references` DISABLE KEYS */;
INSERT INTO `sys_file_references` VALUES (1,1,'demo_attachment','quickstart','2026-10-20 14:56:14.543');
/*!40000 ALTER TABLE `sys_file_references` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_file_resources`
--

DROP TABLE IF EXISTS `sys_file_resources`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_file_resources` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `object_key` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `mime` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `size` bigint NOT NULL,
  `owner_id` bigint NOT NULL,
  `department_id` bigint NOT NULL DEFAULT '0',
  `visibility` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'public',
  `status` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'active',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_file_object_key` (`object_key`),
  KEY `idx_file_owner_status` (`owner_id`,`status`),
  KEY `idx_file_department_status` (`department_id`,`status`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_file_resources`
--

LOCK TABLES `sys_file_resources` WRITE;
/*!40000 ALTER TABLE `sys_file_resources` DISABLE KEYS */;
INSERT INTO `sys_file_resources` VALUES (1,'go-zero-admin/00000000-0000-4000-8000-000000000101.png','演示图片-引用保护.png','image/png',703,2,2,'public','active','2026-10-20 14:56:14.543','2026-10-20 14:56:14.543'),(2,'go-zero-admin/00000000-0000-4000-8000-000000000102.png','演示图片-可删除.png','image/png',611,1,1,'public','active','2026-10-20 14:56:14.543','2026-10-20 14:56:14.543'),(3,'go-zero-admin/00000000-0000-4000-8000-000000000103.png','演示图片-私有下载.png','image/png',525,2,2,'private','active','2026-10-20 14:56:14.543','2026-10-20 14:56:14.543');
/*!40000 ALTER TABLE `sys_file_resources` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_permission_changes`
--

DROP TABLE IF EXISTS `sys_permission_changes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_permission_changes` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `authority_id` bigint NOT NULL,
  `kind` varchar(16) NOT NULL,
  `before_revision` bigint unsigned NOT NULL,
  `after_revision` bigint unsigned NOT NULL,
  `before` longtext NOT NULL,
  `after` longtext NOT NULL,
  `actor_id` bigint NOT NULL DEFAULT '0',
  `actor_name` varchar(64) NOT NULL DEFAULT '',
  `trace_id` varchar(64) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_permission_history_role` (`authority_id`,`id`),
  KEY `idx_permission_history_revision` (`after_revision`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_permission_changes`
--

LOCK TABLES `sys_permission_changes` WRITE;
/*!40000 ALTER TABLE `sys_permission_changes` DISABLE KEYS */;
/*!40000 ALTER TABLE `sys_permission_changes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_permission_versions`
--

DROP TABLE IF EXISTS `sys_permission_versions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_permission_versions` (
  `id` bigint NOT NULL,
  `revision` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_permission_versions`
--

LOCK TABLES `sys_permission_versions` WRITE;
/*!40000 ALTER TABLE `sys_permission_versions` DISABLE KEYS */;
INSERT INTO `sys_permission_versions` VALUES (1,7);
/*!40000 ALTER TABLE `sys_permission_versions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_policy_versions`
--

DROP TABLE IF EXISTS `sys_policy_versions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_policy_versions` (
  `id` bigint unsigned NOT NULL,
  `version` bigint unsigned NOT NULL DEFAULT '0',
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_policy_versions`
--

LOCK TABLES `sys_policy_versions` WRITE;
/*!40000 ALTER TABLE `sys_policy_versions` DISABLE KEYS */;
INSERT INTO `sys_policy_versions` VALUES (1,5,'2026-10-20 14:56:14.543');
/*!40000 ALTER TABLE `sys_policy_versions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_positions`
--

DROP TABLE IF EXISTS `sys_positions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_positions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `code` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `sort` bigint NOT NULL DEFAULT '0',
  `status` bigint NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_position_code` (`code`),
  KEY `idx_position_deleted` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_positions`
--

LOCK TABLES `sys_positions` WRITE;
/*!40000 ALTER TABLE `sys_positions` DISABLE KEYS */;
INSERT INTO `sys_positions` VALUES (1,'2026-10-20 22:56:14.548','2026-10-20 22:56:14.548',NULL,'运营专员','DEMO_OPERATOR',10,1),(2,'2026-10-20 22:56:14.548','2026-10-20 22:56:14.548',NULL,'观察员','DEMO_VIEWER',20,1);
/*!40000 ALTER TABLE `sys_positions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_role_data_scopes`
--

DROP TABLE IF EXISTS `sys_role_data_scopes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_role_data_scopes` (
  `authority_id` bigint NOT NULL,
  `scope` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'self',
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`authority_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_role_data_scopes`
--

LOCK TABLES `sys_role_data_scopes` WRITE;
/*!40000 ALTER TABLE `sys_role_data_scopes` DISABLE KEYS */;
INSERT INTO `sys_role_data_scopes` VALUES (1,'all','2026-10-20 22:56:14.549'),(2,'self','2026-10-20 22:56:14.549'),(3,'self','2026-10-20 22:56:14.549');
/*!40000 ALTER TABLE `sys_role_data_scopes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_role_scope_departments`
--

DROP TABLE IF EXISTS `sys_role_scope_departments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_role_scope_departments` (
  `authority_id` bigint NOT NULL,
  `department_id` bigint NOT NULL,
  PRIMARY KEY (`authority_id`,`department_id`),
  KEY `idx_scope_department` (`department_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_role_scope_departments`
--

LOCK TABLES `sys_role_scope_departments` WRITE;
/*!40000 ALTER TABLE `sys_role_scope_departments` DISABLE KEYS */;
/*!40000 ALTER TABLE `sys_role_scope_departments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_user_authority`
--

DROP TABLE IF EXISTS `sys_user_authority`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_user_authority` (
  `sys_user_id` bigint unsigned NOT NULL,
  `sys_authority_authority_id` bigint unsigned NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_user_id`,`sys_authority_authority_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_user_authority`
--

LOCK TABLES `sys_user_authority` WRITE;
/*!40000 ALTER TABLE `sys_user_authority` DISABLE KEYS */;
INSERT INTO `sys_user_authority` VALUES (1,1),(2,2),(3,3),(4,3);
/*!40000 ALTER TABLE `sys_user_authority` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_user_departments`
--

DROP TABLE IF EXISTS `sys_user_departments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_user_departments` (
  `user_id` bigint NOT NULL,
  `department_id` bigint NOT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`user_id`),
  KEY `idx_user_department` (`department_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_user_departments`
--

LOCK TABLES `sys_user_departments` WRITE;
/*!40000 ALTER TABLE `sys_user_departments` DISABLE KEYS */;
INSERT INTO `sys_user_departments` VALUES (1,1,'2026-10-20 22:56:14.548'),(2,2,'2026-10-20 22:56:14.548'),(3,3,'2026-10-20 22:56:14.548'),(4,3,'2026-10-20 22:56:14.548');
/*!40000 ALTER TABLE `sys_user_departments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_user_positions`
--

DROP TABLE IF EXISTS `sys_user_positions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_user_positions` (
  `user_id` bigint NOT NULL,
  `position_id` bigint NOT NULL,
  PRIMARY KEY (`user_id`,`position_id`),
  KEY `idx_user_position` (`position_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_user_positions`
--

LOCK TABLES `sys_user_positions` WRITE;
/*!40000 ALTER TABLE `sys_user_positions` DISABLE KEYS */;
INSERT INTO `sys_user_positions` VALUES (2,1),(3,2),(4,2);
/*!40000 ALTER TABLE `sys_user_positions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_users`
--

DROP TABLE IF EXISTS `sys_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `uuid` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户UUID',
  `username` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户登录名',
  `password` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户登录密码',
  `nick_name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '系统用户' COMMENT '用户昵称',
  `side_mode` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'dark' COMMENT '用户侧边主题',
  `header_img` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '/default-avatar.svg' COMMENT '用户头像',
  `base_color` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '#fff' COMMENT '基础颜色',
  `active_color` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '#fff' COMMENT '活跃颜色',
  `authority_id` bigint unsigned DEFAULT '101' COMMENT '用户角色ID',
  `phone` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户手机号',
  `email` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户邮箱',
  `enable` bigint DEFAULT '1' COMMENT '用户是否被冻结 1正常 2冻结',
  `active_username` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci GENERATED ALWAYS AS ((case when (`deleted_at` is null) then `username` else NULL end)) STORED,
  `session_version` bigint NOT NULL DEFAULT '1' COMMENT 'Session version',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_sys_users_active_username` (`active_username`),
  KEY `idx_sys_users_deleted_at` (`deleted_at`) USING BTREE,
  KEY `idx_sys_users_uuid` (`uuid`) USING BTREE,
  KEY `idx_sys_users_username` (`username`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_users`
--

LOCK TABLES `sys_users` WRITE;
/*!40000 ALTER TABLE `sys_users` DISABLE KEYS */;
INSERT INTO `sys_users` (`id`, `created_at`, `updated_at`, `deleted_at`, `uuid`, `username`, `password`, `nick_name`, `side_mode`, `header_img`, `base_color`, `active_color`, `authority_id`, `phone`, `email`, `enable`, `session_version`) VALUES (1,'2024-01-10 11:54:46.167','2024-01-10 11:54:46.167',NULL,'e2ab86b1-8e3d-4864-80ad-e83131353686','admin','$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m','超级管理员','dark','/default-avatar.svg','#000','#fff',1,'','',1,1),(2,'2026-10-20 14:56:14.543','2026-10-20 14:56:14.543',NULL,'00000000-0000-4000-8000-000000000002','operator','$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy','演示运营','dark','/default-avatar.svg','#000','#fff',2,'','operator@example.test',1,1),(3,'2026-10-20 14:56:14.543','2026-10-20 14:56:14.543',NULL,'00000000-0000-4000-8000-000000000003','viewer','$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy','演示观察员','dark','/default-avatar.svg','#000','#fff',3,'','viewer@example.test',1,1),(4,'2026-10-20 14:56:14.543','2026-10-20 14:56:14.543',NULL,'00000000-0000-4000-8000-000000000004','disabled','$2a$10$gUoFRszVNyzR45Qh1ZhtPun31tpprmsYsT82zPLzmfLIjvWe2YqGy','禁用账号示例','dark','/default-avatar.svg','#000','#fff',3,'','disabled-admin@example.test',2,1);
/*!40000 ALTER TABLE `sys_users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Dumping events for database 'goZero-admin'
--

--
-- Dumping routines for database 'goZero-admin'
--
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-10-20 22:56:15
