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
) ENGINE=InnoDB AUTO_INCREMENT=170 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `casbin_rule`
--

LOCK TABLES `casbin_rule` WRITE;
/*!40000 ALTER TABLE `casbin_rule` DISABLE KEYS */;
INSERT INTO `casbin_rule` VALUES (166,'p','1','/v1/ai/conversations','GET',NULL,NULL,NULL),(167,'p','1','/v1/ai/conversations/:id/messages','GET',NULL,NULL,NULL),(162,'p','1','/v1/ai/info','GET',NULL,NULL,NULL),(163,'p','1','/v1/ai/runs','POST',NULL,NULL,NULL),(164,'p','1','/v1/ai/runs/:id','GET',NULL,NULL,NULL),(165,'p','1','/v1/ai/runs/:id/cancel','POST',NULL,NULL,NULL),(131,'p','1','/v1/sys/api/applySync','POST',NULL,NULL,NULL),(64,'p','1','/v1/sys/api/createApi','POST','','',''),(65,'p','1','/v1/sys/api/deleteApi','DELETE','','',''),(67,'p','1','/v1/sys/api/deleteApisByIds','DELETE','','',''),(66,'p','1','/v1/sys/api/getAllApiList','GET','','',''),(63,'p','1','/v1/sys/api/getApiList','GET','','',''),(130,'p','1','/v1/sys/api/previewSync','GET',NULL,NULL,NULL),(68,'p','1','/v1/sys/api/updateApi','PUT','','',''),(140,'p','1','/v1/sys/audit/getAuditLogList','GET',NULL,NULL,NULL),(59,'p','1','/v1/sys/authority/addAuthorityMenu','POST','','',''),(61,'p','1','/v1/sys/authority/createAuthority','POST','','',''),(62,'p','1','/v1/sys/authority/deleteAuthority','DELETE','','',''),(58,'p','1','/v1/sys/authority/getAuthorityList','GET','','',''),(60,'p','1','/v1/sys/authority/updateAuthority','PUT','','',''),(84,'p','1','/v1/sys/base/sendEmailCode','POST','','',''),(82,'p','1','/v1/sys/base/uploadFileImg','POST','','',''),(79,'p','1','/v1/sys/casbin/getPathByAuthorityId','GET','','',''),(80,'p','1','/v1/sys/casbin/updateCasbinData','PUT','','',''),(81,'p','1','/v1/sys/casbin/updateCasbinDataByApiIds','PUT','','',''),(49,'p','1','/v1/sys/deleteUser','DELETE','','',''),(70,'p','1','/v1/sys/dictionary/createSysDictionary','POST','','',''),(77,'p','1','/v1/sys/dictionary/createSysDictionaryInfo','POST','','',''),(73,'p','1','/v1/sys/dictionary/deleteSysDictionary','DELETE','','',''),(78,'p','1','/v1/sys/dictionary/deleteSysDictionaryInfo','DELETE','','',''),(71,'p','1','/v1/sys/dictionary/getSysDictionaryDetails','GET','','',''),(74,'p','1','/v1/sys/dictionary/getSysDictionaryInfoList','GET','','',''),(75,'p','1','/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','GET','','',''),(83,'p','1','/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','GET','','',''),(69,'p','1','/v1/sys/dictionary/getSysDictionaryList','GET','','',''),(72,'p','1','/v1/sys/dictionary/updateSysDictionary','PUT','','',''),(76,'p','1','/v1/sys/dictionary/updateSysDictionaryInfo','PUT','','',''),(141,'p','1','/v1/sys/files/list','GET',NULL,NULL,NULL),(142,'p','1','/v1/sys/files/reference','DELETE',NULL,NULL,NULL),(143,'p','1','/v1/sys/files/reference','POST',NULL,NULL,NULL),(144,'p','1','/v1/sys/files/resource','DELETE',NULL,NULL,NULL),(145,'p','1','/v1/sys/files/url','GET',NULL,NULL,NULL),(45,'p','1','/v1/sys/getUserList','GET','','',''),(43,'p','1','/v1/sys/login','POST','','',''),(52,'p','1','/v1/sys/menu/addBaseMenu','POST','','',''),(57,'p','1','/v1/sys/menu/deleteBaseMenu','DELETE','','',''),(127,'p','1','/v1/sys/menu/getAuthorityButtons','GET',NULL,NULL,NULL),(55,'p','1','/v1/sys/menu/getBaseMenuById','GET','','',''),(53,'p','1','/v1/sys/menu/getBaseMenuTree','GET','','',''),(50,'p','1','/v1/sys/menu/getMenu','GET','','',''),(54,'p','1','/v1/sys/menu/getMenuAuthority','GET','','',''),(51,'p','1','/v1/sys/menu/getMenuList','GET','','',''),(169,'p','1','/v1/sys/menu/previewMove','POST','','',''),(128,'p','1','/v1/sys/menu/updateAuthorityButtons','PUT',NULL,NULL,NULL),(56,'p','1','/v1/sys/menu/updateBaseMenu','PUT','','',''),(158,'p','1','/v1/sys/organization/dataScope','GET',NULL,NULL,NULL),(159,'p','1','/v1/sys/organization/dataScope','PUT',NULL,NULL,NULL),(151,'p','1','/v1/sys/organization/departments','DELETE',NULL,NULL,NULL),(148,'p','1','/v1/sys/organization/departments','GET',NULL,NULL,NULL),(149,'p','1','/v1/sys/organization/departments','POST',NULL,NULL,NULL),(150,'p','1','/v1/sys/organization/departments','PUT',NULL,NULL,NULL),(156,'p','1','/v1/sys/organization/membership','GET',NULL,NULL,NULL),(157,'p','1','/v1/sys/organization/membership','PUT',NULL,NULL,NULL),(155,'p','1','/v1/sys/organization/positions','DELETE',NULL,NULL,NULL),(152,'p','1','/v1/sys/organization/positions','GET',NULL,NULL,NULL),(153,'p','1','/v1/sys/organization/positions','POST',NULL,NULL,NULL),(154,'p','1','/v1/sys/organization/positions','PUT',NULL,NULL,NULL),(168,'p','1','/v1/sys/permissions/edit','GET','','',''),(44,'p','1','/v1/sys/randomImage','GET','','',''),(46,'p','1','/v1/sys/register','POST','','',''),(48,'p','1','/v1/sys/resetUserPassword','PUT','','',''),(161,'p','1','/v1/sys/session/admin/device','DELETE',NULL,NULL,NULL),(160,'p','1','/v1/sys/session/admin/devices','GET',NULL,NULL,NULL),(47,'p','1','/v1/sys/updateUserInfo','PUT','','',''),(22,'p','10','/v1/sys/api/createApi','POST','','',''),(23,'p','10','/v1/sys/api/deleteApi','DELETE','','',''),(25,'p','10','/v1/sys/api/deleteApisByIds','DELETE','','',''),(24,'p','10','/v1/sys/api/getAllApiList','GET','','',''),(21,'p','10','/v1/sys/api/getApiList','GET','','',''),(26,'p','10','/v1/sys/api/updateApi','PUT','','',''),(17,'p','10','/v1/sys/authority/addAuthorityMenu','POST','','',''),(19,'p','10','/v1/sys/authority/createAuthority','POST','','',''),(20,'p','10','/v1/sys/authority/deleteAuthority','DELETE','','',''),(16,'p','10','/v1/sys/authority/getAuthorityList','GET','','',''),(18,'p','10','/v1/sys/authority/updateAuthority','PUT','','',''),(42,'p','10','/v1/sys/base/sendEmailCode','POST','','',''),(40,'p','10','/v1/sys/base/uploadFileImg','POST','','',''),(37,'p','10','/v1/sys/casbin/getPathByAuthorityId','GET','','',''),(38,'p','10','/v1/sys/casbin/updateCasbinData','PUT','','',''),(39,'p','10','/v1/sys/casbin/updateCasbinDataByApiIds','PUT','','',''),(7,'p','10','/v1/sys/deleteUser','DELETE','','',''),(28,'p','10','/v1/sys/dictionary/createSysDictionary','POST','','',''),(35,'p','10','/v1/sys/dictionary/createSysDictionaryInfo','POST','','',''),(31,'p','10','/v1/sys/dictionary/deleteSysDictionary','DELETE','','',''),(36,'p','10','/v1/sys/dictionary/deleteSysDictionaryInfo','DELETE','','',''),(29,'p','10','/v1/sys/dictionary/getSysDictionaryDetails','GET','','',''),(32,'p','10','/v1/sys/dictionary/getSysDictionaryInfoList','GET','','',''),(33,'p','10','/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','GET','','',''),(41,'p','10','/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','GET','','',''),(27,'p','10','/v1/sys/dictionary/getSysDictionaryList','GET','','',''),(30,'p','10','/v1/sys/dictionary/updateSysDictionary','PUT','','',''),(34,'p','10','/v1/sys/dictionary/updateSysDictionaryInfo','PUT','','',''),(3,'p','10','/v1/sys/getUserList','GET','','',''),(1,'p','10','/v1/sys/login','POST','','',''),(10,'p','10','/v1/sys/menu/addBaseMenu','POST','','',''),(15,'p','10','/v1/sys/menu/deleteBaseMenu','DELETE','','',''),(13,'p','10','/v1/sys/menu/getBaseMenuById','GET','','',''),(11,'p','10','/v1/sys/menu/getBaseMenuTree','GET','','',''),(8,'p','10','/v1/sys/menu/getMenu','GET','','',''),(12,'p','10','/v1/sys/menu/getMenuAuthority','GET','','',''),(9,'p','10','/v1/sys/menu/getMenuList','GET','','',''),(14,'p','10','/v1/sys/menu/updateBaseMenu','PUT','','',''),(2,'p','10','/v1/sys/randomImage','GET','','',''),(4,'p','10','/v1/sys/register','POST','','',''),(6,'p','10','/v1/sys/resetUserPassword','PUT','','',''),(5,'p','10','/v1/sys/updateUserInfo','PUT','','',''),(106,'p','102','/v1/sys/api/createApi','POST','','',''),(107,'p','102','/v1/sys/api/deleteApi','DELETE','','',''),(109,'p','102','/v1/sys/api/deleteApisByIds','DELETE','','',''),(108,'p','102','/v1/sys/api/getAllApiList','GET','','',''),(105,'p','102','/v1/sys/api/getApiList','GET','','',''),(110,'p','102','/v1/sys/api/updateApi','PUT','','',''),(101,'p','102','/v1/sys/authority/addAuthorityMenu','POST','','',''),(103,'p','102','/v1/sys/authority/createAuthority','POST','','',''),(104,'p','102','/v1/sys/authority/deleteAuthority','DELETE','','',''),(100,'p','102','/v1/sys/authority/getAuthorityList','GET','','',''),(102,'p','102','/v1/sys/authority/updateAuthority','PUT','','',''),(126,'p','102','/v1/sys/base/sendEmailCode','POST','','',''),(124,'p','102','/v1/sys/base/uploadFileImg','POST','','',''),(121,'p','102','/v1/sys/casbin/getPathByAuthorityId','GET','','',''),(122,'p','102','/v1/sys/casbin/updateCasbinData','PUT','','',''),(123,'p','102','/v1/sys/casbin/updateCasbinDataByApiIds','PUT','','',''),(91,'p','102','/v1/sys/deleteUser','DELETE','','',''),(112,'p','102','/v1/sys/dictionary/createSysDictionary','POST','','',''),(119,'p','102','/v1/sys/dictionary/createSysDictionaryInfo','POST','','',''),(115,'p','102','/v1/sys/dictionary/deleteSysDictionary','DELETE','','',''),(120,'p','102','/v1/sys/dictionary/deleteSysDictionaryInfo','DELETE','','',''),(113,'p','102','/v1/sys/dictionary/getSysDictionaryDetails','GET','','',''),(116,'p','102','/v1/sys/dictionary/getSysDictionaryInfoList','GET','','',''),(117,'p','102','/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','GET','','',''),(125,'p','102','/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','GET','','',''),(111,'p','102','/v1/sys/dictionary/getSysDictionaryList','GET','','',''),(114,'p','102','/v1/sys/dictionary/updateSysDictionary','PUT','','',''),(118,'p','102','/v1/sys/dictionary/updateSysDictionaryInfo','PUT','','',''),(87,'p','102','/v1/sys/getUserList','GET','','',''),(85,'p','102','/v1/sys/login','POST','','',''),(94,'p','102','/v1/sys/menu/addBaseMenu','POST','','',''),(99,'p','102','/v1/sys/menu/deleteBaseMenu','DELETE','','',''),(97,'p','102','/v1/sys/menu/getBaseMenuById','GET','','',''),(95,'p','102','/v1/sys/menu/getBaseMenuTree','GET','','',''),(92,'p','102','/v1/sys/menu/getMenu','GET','','',''),(96,'p','102','/v1/sys/menu/getMenuAuthority','GET','','',''),(93,'p','102','/v1/sys/menu/getMenuList','GET','','',''),(98,'p','102','/v1/sys/menu/updateBaseMenu','PUT','','',''),(86,'p','102','/v1/sys/randomImage','GET','','',''),(88,'p','102','/v1/sys/register','POST','','',''),(90,'p','102','/v1/sys/resetUserPassword','PUT','','',''),(89,'p','102','/v1/sys/updateUserInfo','PUT','','','');
/*!40000 ALTER TABLE `casbin_rule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `schema_migrations`
--

DROP TABLE IF EXISTS `schema_migrations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `schema_migrations` (
  `filename` varchar(191) NOT NULL,
  `checksum` char(64) NOT NULL,
  `applied_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`filename`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `schema_migrations`
--

LOCK TABLES `schema_migrations` WRITE;
/*!40000 ALTER TABLE `schema_migrations` DISABLE KEYS */;
INSERT INTO `schema_migrations` VALUES ('20260925_business_access.sql','af8cb82d64db93891146c3c69dfcaf0ca0eff96f861a2b8ce74f050204639659','2026-09-26 01:01:05'),('20260925_user_active_username.sql','cf2440bfd60941cbec000dd9eef18e517d0a305ee1ea126cd349cef8ee1d4b7c','2026-09-26 01:01:06'),('20260926_00_method_dictionary.sql','399b0b3901b6f2e7edb71d042f4738379164a8686ec2242a1154a7faab671dba','2026-09-26 01:01:08'),('20260926_api_sync_access.sql','287f534292e3643e23f3c450c47aa5420e7416b4c0b5d1564ca669144481bb9b','2026-09-26 01:01:09'),('20260926_business_unique.sql','5baeac3baed30fae0a4c3ed9b81e8ca4a6ca5ecaf76caeb8017f720684a16aa6','2026-09-26 01:01:10'),('20260926_session_version.sql','9898ed43d66cf2e842e7cd70ca6ccaedcbb0e2964c07cacf5a6eb013f2d3c9ba','2026-09-26 01:01:12'),('20261002_00_policy_sync.sql','bf40d0fa1ef28e3c4e175dddbbe00ae875df7ed0e471f0dfea4fc757e4b4ae73','2026-10-03 00:40:19'),('20261002_01_audit.sql','06d7fde9ac5519750f070fa592e0d49d23d522f939096f6989cb5fb2352fcdea','2026-10-03 00:40:21'),('20261002_02_organization.sql','2a39b8e7682539b92c24b32d18eed8e98eb97bd63ca0a8029ce35832566c6f40','2026-10-03 00:40:22'),('20261002_03_files.sql','6600329d92643c909eff06848d7944b8d65b0b66a91eb77c5a648f806a957f8e','2026-10-03 00:40:23'),('20261002_04_device_sessions.sql','06dcc6a7e7f263f34b5d0373d8eb1bcde40cf91bdb237ff448ee6a752e1a39b4','2026-10-03 00:40:25'),('20261002_05_module_access.sql','cfadb622d1002666708d0cff4a35bb666f051fab81d3440de6eb5f1da11b3e0d','2026-10-03 00:40:26'),('20261003_frontend_modules.sql','e62dc29eb9d409d949c24c7387d2e10deab064ccee2f17abc3da5b48dca7c9fa','2026-10-03 00:53:47'),('20261003_frontend_routes_compat.sql','aaffb0a8d4f204b4a0f94e5c54b10c33fc2a4772d7d989eeef1cf159c4007b2e','2026-10-03 03:16:03'),('20261003_zz_ai_agent.sql','9865c0e14f334a3ae7ee0b861363cea6b0eba97ae70db72a32e33649a378ee6e','2026-10-03 14:36:04'),('20261008_permission_revision_history.sql','bc8c6e02847cc29f0dcabff6098326e3854a37303c28ca9cc03e7cbeb58ebfe9','2026-10-09 22:45:25');
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
  `title` varchar(160) COLLATE utf8mb4_general_ci NOT NULL,
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
INSERT INTO `sys_ai_conversations` VALUES ('5859800a-f7d2-477b-80b4-2dd0d365cfa8',1,'推荐golang开源项目',1,'2026-10-04 14:10:06.504','2026-10-04 14:10:08.078'),('b7aab08a-0f7a-4719-981c-fe013a39087a',1,'您好，你是什么模型',4,'2026-10-04 13:52:54.251','2026-10-04 13:54:45.276');
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
  `role` varchar(16) COLLATE utf8mb4_general_ci NOT NULL,
  `content` longtext COLLATE utf8mb4_general_ci NOT NULL,
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
INSERT INTO `sys_ai_messages` VALUES ('058fd715-9071-468f-b7e1-384eb1443307',1,'b7aab08a-0f7a-4719-981c-fe013a39087a','1090cf40-5023-46ee-998e-608897423601','assistant','我目前只能使用本次注册的只读工具，提供以下三类查询：\n\n1. **审计日志查询**：按时间范围（默认上海时间今天，最长31天）以及模块、事件类型（操作/登录）、结果（成功/失败）筛选，返回按模块的计数和最多20条最近记录。不包含请求原文和IP。\n2. **文件状态查询**：按 fileId 查询文件的状态、大小和引用数，并说明引用对删除的影响；不返回下载链接。\n3. **本人登录设备查询**：查看本人最多20个有效登录设备（信息脱敏），仅限本人。\n\n说明：我只具备只读权限，不能执行任何修改、删除或其他写操作。如需以上任一查询，请告诉我具体条件（例如时间范围、模块等）。','2026-10-04 13:53:15.099'),('0c8e8cc4-56ab-4261-a1bf-d52725dcfb8c',1,'b7aab08a-0f7a-4719-981c-fe013a39087a','fb4e0b1d-2d84-4593-bb1d-4ad69d455890','assistant','审计日志查询结果（默认今天：2026-10-04 00:00:00 至 13:54:42，上海时间）：\n\n**按模块计数（总计 8 条）**\n| 模块 | 条数 |\n|---|---|\n| ai-agent | 3 |\n| sys | 3 |\n| session | 1 |\n| user | 1 |\n\n**最近记录（共 8 条，均为成功）**\n| ID | 模块 | 操作 | 操作者 | 结果 | 状态码 | 时间（上海） |\n|---|---|---|---|---|---|---|\n| 18 | sys | runs | admin | 成功 | 200 | 13:53:40 |\n| 17 | ai-agent | submitRun | — | 成功 | 0 | 13:53:40 |\n| 16 | sys | runs | admin | 成功 | 200 | 13:53:13 |\n| 15 | ai-agent | submitRun | — | 成功 | 0 | 13:53:13 |\n| 14 | sys | runs | admin | 成功 | 200 | 13:52:54 |\n| 13 | ai-agent | submitRun | — | 成功 | 0 | 13:52:54 |\n| 12 | user | login | admin | 成功 | 200 | 13:52:35 |\n| 11 | session | createDeviceSession | admin | 成功 | 0 | 13:52:35 |\n\n说明：以上为按模块计数与最近最多20条记录，未含请求原文和IP。如需按模块、事件类型（操作/登录）或结果（成功/失败）筛选，或指定其他时间范围（最长31天），请告知。','2026-10-04 13:54:45.276'),('23e952a9-b609-4188-8738-8465160bd8a0',1,'b7aab08a-0f7a-4719-981c-fe013a39087a','95014f8e-efa3-4851-9ea7-a0428f62ed1f','assistant','您好，我是本系统的管理只读助手，可协助查询审计日志、文件状态和本人登录设备等信息。关于我底层具体是哪一款模型，这里不便说明。请问需要我帮您查询什么？','2026-10-04 13:52:58.201'),('70181691-9a1f-4b61-823b-69af441b2cff',1,'b7aab08a-0f7a-4719-981c-fe013a39087a','839d1d99-6299-4f09-bf2b-6fae18646b71','assistant','查询结果（本人有效登录设备，最多20条）：\n\n| 设备ID | 客户端 | 登录时间 | 过期时间 | 是否当前设备 |\n|---|---|---|---|---|\n| 1 | Chrome | 2026-10-04 13:52:35（上海时间） | 2026-10-04 15:52:35（上海时间） | 是 |\n\n共 1 台有效登录设备。','2026-10-04 13:53:42.133'),('73848b5a-1a1c-4f89-9339-6b640d202961',1,'5859800a-f7d2-477b-80b4-2dd0d365cfa8','2ce775be-00c5-4d60-b574-9bb84a6dee61','user','推荐golang开源项目','2026-10-04 14:10:06.504'),('99db4216-ee82-4343-898e-b607a1bda392',1,'b7aab08a-0f7a-4719-981c-fe013a39087a','1090cf40-5023-46ee-998e-608897423601','user','你可以做什么','2026-10-04 13:53:13.725'),('9bd20dcd-351d-48a3-a153-ef3d7b4dd9ae',1,'b7aab08a-0f7a-4719-981c-fe013a39087a','95014f8e-efa3-4851-9ea7-a0428f62ed1f','user','您好，你是什么模型','2026-10-04 13:52:54.251'),('9d7622cf-4386-4bf4-8491-9e98c8bdafcc',1,'b7aab08a-0f7a-4719-981c-fe013a39087a','839d1d99-6299-4f09-bf2b-6fae18646b71','user','本人登录设备查询','2026-10-04 13:53:40.519'),('f9528ef6-108c-4e93-af70-1ef91d5ed5a3',1,'5859800a-f7d2-477b-80b4-2dd0d365cfa8','2ce775be-00c5-4d60-b574-9bb84a6dee61','assistant','我没有可用的工具能查询或推荐 Golang 开源项目——本次注册的工具只覆盖审计日志、文件状态和本人登录设备三类只读查询，不包含任何项目检索能力。\n\n如果你需要的是与这个系统相关的信息，我可以帮你查：\n- 本人最近的登录/操作审计记录\n- 某个文件的状态、大小和引用数\n- 本人当前有效的登录设备\n\n需要的话告诉我具体要查哪一项。','2026-10-04 14:10:08.078'),('fd0d9bb2-a2c2-4ef3-a82e-318bf1688c2a',1,'b7aab08a-0f7a-4719-981c-fe013a39087a','fb4e0b1d-2d84-4593-bb1d-4ad69d455890','user','审计日志查询','2026-10-04 13:54:42.164');
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
  `question` longtext COLLATE utf8mb4_general_ci NOT NULL,
  `sequence` bigint NOT NULL,
  `status` varchar(16) COLLATE utf8mb4_general_ci NOT NULL,
  `cancel_requested` tinyint(1) NOT NULL DEFAULT '0',
  `text` longtext COLLATE utf8mb4_general_ci NOT NULL,
  `provider` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `model` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `input_tokens` bigint NOT NULL DEFAULT '0',
  `output_tokens` bigint NOT NULL DEFAULT '0',
  `extra_json` longtext COLLATE utf8mb4_general_ci NOT NULL,
  `error_code` varchar(32) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `error` varchar(256) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
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
INSERT INTO `sys_ai_runs` VALUES ('1090cf40-5023-46ee-998e-608897423601',1,1,'1e17de9b-ed5e-4b48-bff3-c128b2f34071',1,'eda5b16a-89fc-48a2-b3dd-b1961d5d61d1','1d372d93ef1afee8f90a174c2c9daf1c93b1f252bfecd83eb52231ae2c9ad5a1','b7aab08a-0f7a-4719-981c-fe013a39087a','你可以做什么',2,'succeeded',0,'我目前只能使用本次注册的只读工具，提供以下三类查询：\n\n1. **审计日志查询**：按时间范围（默认上海时间今天，最长31天）以及模块、事件类型（操作/登录）、结果（成功/失败）筛选，返回按模块的计数和最多20条最近记录。不包含请求原文和IP。\n2. **文件状态查询**：按 fileId 查询文件的状态、大小和引用数，并说明引用对删除的影响；不返回下载链接。\n3. **本人登录设备查询**：查看本人最多20个有效登录设备（信息脱敏），仅限本人。\n\n说明：我只具备只读权限，不能执行任何修改、删除或其他写操作。如需以上任一查询，请告诉我具体条件（例如时间范围、模块等）。','deepseek','deepseek-flash',776,168,'[]','','','56c6ca66-e445-422d-8f5e-22dfc699b9e7',NULL,'2026-10-04 13:58:13.725','2026-10-04 13:54:43.742','2026-10-04 13:53:13.742','2026-10-04 13:53:15.099','2026-10-04 13:53:13.725','2026-10-04 13:53:15.099'),('2ce775be-00c5-4d60-b574-9bb84a6dee61',1,1,'1e17de9b-ed5e-4b48-bff3-c128b2f34071',1,'fd038e78-60f9-41d3-8d6f-dc082cb9ad28','b98af778354ff8f2a09c62c0fa05b0505113153655f044f715efa2b595c83cbb','5859800a-f7d2-477b-80b4-2dd0d365cfa8','推荐golang开源项目',1,'succeeded',0,'我没有可用的工具能查询或推荐 Golang 开源项目——本次注册的工具只覆盖审计日志、文件状态和本人登录设备三类只读查询，不包含任何项目检索能力。\n\n如果你需要的是与这个系统相关的信息，我可以帮你查：\n- 本人最近的登录/操作审计记录\n- 某个文件的状态、大小和引用数\n- 本人当前有效的登录设备\n\n需要的话告诉我具体要查哪一项。','deepseek','deepseek-flash',726,91,'[]','','','56c6ca66-e445-422d-8f5e-22dfc699b9e7',NULL,'2026-10-04 14:15:06.504','2026-10-04 14:11:36.521','2026-10-04 14:10:06.521','2026-10-04 14:10:08.078','2026-10-04 14:10:06.504','2026-10-04 14:10:08.078'),('839d1d99-6299-4f09-bf2b-6fae18646b71',1,1,'1e17de9b-ed5e-4b48-bff3-c128b2f34071',1,'8ea83814-4214-428b-acb2-6e778ab8eed8','a980948542544b015f4b17af10efa2d05f631c5da90a148aa1bc84adc4c51cf9','b7aab08a-0f7a-4719-981c-fe013a39087a','本人登录设备查询',3,'succeeded',0,'查询结果（本人有效登录设备，最多20条）：\n\n| 设备ID | 客户端 | 登录时间 | 过期时间 | 是否当前设备 |\n|---|---|---|---|---|\n| 1 | Chrome | 2026-10-04 13:52:35（上海时间） | 2026-10-04 15:52:35（上海时间） | 是 |\n\n共 1 台有效登录设备。','deepseek','deepseek-flash',2006,115,'[{\"name\":\"list_my_devices\",\"status\":\"succeeded\",\"error\":\"\",\"summary\":\"已查询本人有效设备，隐藏会话编号与IP\"}]','','','56c6ca66-e445-422d-8f5e-22dfc699b9e7',NULL,'2026-10-04 13:58:40.519','2026-10-04 13:55:10.538','2026-10-04 13:53:40.538','2026-10-04 13:53:42.133','2026-10-04 13:53:40.519','2026-10-04 13:53:42.133'),('95014f8e-efa3-4851-9ea7-a0428f62ed1f',1,1,'1e17de9b-ed5e-4b48-bff3-c128b2f34071',1,'1dcf2261-4f25-4c1f-ad28-c725c2673c02','882be9a6135e972559aa7051f2030ead74f37891afe19d458666fd1ae15dfe78','b7aab08a-0f7a-4719-981c-fe013a39087a','您好，你是什么模型',1,'succeeded',0,'您好，我是本系统的管理只读助手，可协助查询审计日志、文件状态和本人登录设备等信息。关于我底层具体是哪一款模型，这里不便说明。请问需要我帮您查询什么？','deepseek','deepseek-flash',726,44,'[]','','','56c6ca66-e445-422d-8f5e-22dfc699b9e7',NULL,'2026-10-04 13:57:54.251','2026-10-04 13:54:24.274','2026-10-04 13:52:54.274','2026-10-04 13:52:58.201','2026-10-04 13:52:54.251','2026-10-04 13:52:58.201'),('fb4e0b1d-2d84-4593-bb1d-4ad69d455890',1,1,'1e17de9b-ed5e-4b48-bff3-c128b2f34071',1,'0d7ec61c-3a54-43b7-99ca-8798c5ab5f51','e6303049638ac77b134efd6f158fd9452ac0705b7b4f65f5264dce61a513139e','b7aab08a-0f7a-4719-981c-fe013a39087a','审计日志查询',4,'succeeded',0,'审计日志查询结果（默认今天：2026-10-04 00:00:00 至 13:54:42，上海时间）：\n\n**按模块计数（总计 8 条）**\n| 模块 | 条数 |\n|---|---|\n| ai-agent | 3 |\n| sys | 3 |\n| session | 1 |\n| user | 1 |\n\n**最近记录（共 8 条，均为成功）**\n| ID | 模块 | 操作 | 操作者 | 结果 | 状态码 | 时间（上海） |\n|---|---|---|---|---|---|---|\n| 18 | sys | runs | admin | 成功 | 200 | 13:53:40 |\n| 17 | ai-agent | submitRun | — | 成功 | 0 | 13:53:40 |\n| 16 | sys | runs | admin | 成功 | 200 | 13:53:13 |\n| 15 | ai-agent | submitRun | — | 成功 | 0 | 13:53:13 |\n| 14 | sys | runs | admin | 成功 | 200 | 13:52:54 |\n| 13 | ai-agent | submitRun | — | 成功 | 0 | 13:52:54 |\n| 12 | user | login | admin | 成功 | 200 | 13:52:35 |\n| 11 | session | createDeviceSession | admin | 成功 | 0 | 13:52:35 |\n\n说明：以上为按模块计数与最近最多20条记录，未含请求原文和IP。如需按模块、事件类型（操作/登录）或结果（成功/失败）筛选，或指定其他时间范围（最长31天），请告知。','deepseek','deepseek-flash',2722,514,'[{\"name\":\"query_audit_logs\",\"status\":\"succeeded\",\"error\":\"\",\"summary\":\"匹配8条审计日志，展示最近8条\"}]','','','56c6ca66-e445-422d-8f5e-22dfc699b9e7',NULL,'2026-10-04 13:59:42.164','2026-10-04 13:56:12.184','2026-10-04 13:54:42.184','2026-10-04 13:54:45.276','2026-10-04 13:54:42.164','2026-10-04 13:54:45.276');
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
) ENGINE=InnoDB AUTO_INCREMENT=83 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_apis`
--

LOCK TABLES `sys_apis` WRITE;
/*!40000 ALTER TABLE `sys_apis` DISABLE KEYS */;
INSERT INTO `sys_apis` (`id`, `created_at`, `updated_at`, `deleted_at`, `path`, `description`, `api_group`, `method`) VALUES (1,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/login','系统用户登录','系统-无需权限','POST'),(2,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/randomImage','获取验证码','系统-无需权限','GET'),(3,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/getUserList','分页获取用户列表','用户','GET'),(4,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/register','新增（注册）用户 - 管理员','用户','POST'),(5,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/updateUserInfo','修改用户信息','用户','PUT'),(6,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/resetUserPassword','重置用户密码 默认密码：goZero','用户','PUT'),(7,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/deleteUser','删除用户','用户','DELETE'),(8,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenu','获取菜单','菜单','GET'),(9,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenuList','分页获取base_menu列表','菜单','GET'),(10,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/addBaseMenu','新增 base_menu','菜单','POST'),(11,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getBaseMenuTree','获取用户动态路由树  -- 用于角色管理的设置权限','菜单','GET'),(12,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenuAuthority','获取指定角色menu  -- 用于角色管理的设置权限','菜单','GET'),(13,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getBaseMenuById','根据id获取菜单','菜单','GET'),(14,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/updateBaseMenu','更新系统菜单','菜单','PUT'),(15,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/deleteBaseMenu','删除系统菜单','菜单','DELETE'),(16,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/getAuthorityList','获取角色列表','角色','GET'),(17,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/addAuthorityMenu','增加角色和base_menu关联关系','角色','POST'),(18,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/updateAuthority','更新角色信息','角色','PUT'),(19,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/createAuthority','创建角色','角色','POST'),(20,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/deleteAuthority','删除角色','角色','DELETE'),(21,'2024-01-15 13:14:52.000','2024-01-15 10:35:23.438',NULL,'/v1/sys/api/getApiList','获取api列表','api','GET'),(22,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/createApi','创建/增加 api列表','api','POST'),(23,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/deleteApi','删除 api列表','api','DELETE'),(24,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/getAllApiList','获取 所有api','api','GET'),(25,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/deleteApisByIds','删除多条api','api','DELETE'),(26,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/updateApi','更新api','api','PUT'),(27,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryList','获取SysDictionary列表','字典','GET'),(28,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/createSysDictionary','新建SysDictionary','字典','POST'),(29,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryDetails','根据ID或者type获取SysDictionary','字典','GET'),(30,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/updateSysDictionary','更新SysDictionary','字典','PUT'),(31,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/deleteSysDictionary','删除SysDictionary','字典','DELETE'),(32,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoList','获取SysDictionaryInfo列表','字典','GET'),(33,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','根据id获取SysDictionaryInfo详情','字典','GET'),(34,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/updateSysDictionaryInfo','更新SysDictionaryInfo','字典','PUT'),(35,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/createSysDictionaryInfo','创建SysDictionaryInfo','字典','POST'),(36,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/deleteSysDictionaryInfo','删除SysDictionaryInfo','字典','DELETE'),(37,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/getPathByAuthorityId','根据角色id获取对应的casbin数据','casbin','GET'),(38,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/updateCasbinData','更新一个角色的对应的casbin数据','casbin','PUT'),(39,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/updateCasbinDataByApiIds','更新一个角色的对应的casbin数据 用api的ids 查数据','casbin','PUT'),(40,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/base/uploadFileImg','上传图片','base','POST'),(41,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','根据Type获取SysDictionaryInfo详情','字典','GET'),(42,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/base/sendEmailCode','邮箱发送验证码 - 暂时用于注册账号','base','POST'),(46,'2026-09-25 20:09:55.901','2026-09-25 20:09:55.901',NULL,'/v1/sys/menu/getAuthorityButtons','角色按钮权限','系统管理','GET'),(47,'2026-09-25 20:09:55.901','2026-09-25 20:09:55.901',NULL,'/v1/sys/menu/updateAuthorityButtons','保存按钮授权','系统管理','PUT'),(49,'2026-09-26 01:01:09.004','2026-09-26 01:12:31.755',NULL,'/v1/sys/api/previewSync','预览Swagger与API资源差异，失效资源仅提示','api','GET'),(50,'2026-09-26 01:01:09.005','2026-09-26 01:01:09.005',NULL,'/v1/sys/api/applySync','按预览版本同步选中API，仅新增资源或更新分组描述，保留授权','api','POST'),(53,'2026-10-03 00:40:20.790','2026-10-03 00:40:20.790',NULL,'/v1/sys/audit/getAuditLogList','分页查询操作审计与登录日志','audit','GET'),(54,'2026-10-03 00:40:23.628','2026-10-03 00:40:23.628',NULL,'/v1/sys/files/list','按数据范围查询文件资源','files','GET'),(55,'2026-10-03 00:40:23.628','2026-10-03 00:40:23.628',NULL,'/v1/sys/files/url','获取公开或短时签名文件地址','files','GET'),(56,'2026-10-03 00:40:23.628','2026-10-03 00:40:23.628',NULL,'/v1/sys/files/resource','删除未被引用的资源，失败可重试','files','DELETE'),(57,'2026-10-03 00:40:23.628','2026-10-03 00:40:23.628',NULL,'/v1/sys/files/reference','登记文件业务引用，仅内置管理员','files','POST'),(58,'2026-10-03 00:40:23.628','2026-10-03 00:40:23.628',NULL,'/v1/sys/files/reference','移除文件业务引用，仅内置管理员','files','DELETE'),(61,'2026-10-03 00:40:26.206','2026-10-03 00:40:26.206',NULL,'/v1/sys/organization/departments','查询部门树','organization','GET'),(62,'2026-10-03 00:40:26.207','2026-10-03 00:40:26.207',NULL,'/v1/sys/organization/departments','创建部门','organization','POST'),(63,'2026-10-03 00:40:26.208','2026-10-03 00:40:26.208',NULL,'/v1/sys/organization/departments','更新部门','organization','PUT'),(64,'2026-10-03 00:40:26.209','2026-10-03 00:40:26.209',NULL,'/v1/sys/organization/departments','删除部门','organization','DELETE'),(65,'2026-10-03 00:40:26.210','2026-10-03 00:40:26.210',NULL,'/v1/sys/organization/positions','查询岗位','organization','GET'),(66,'2026-10-03 00:40:26.211','2026-10-03 00:40:26.211',NULL,'/v1/sys/organization/positions','创建岗位','organization','POST'),(67,'2026-10-03 00:40:26.211','2026-10-03 00:40:26.211',NULL,'/v1/sys/organization/positions','更新岗位','organization','PUT'),(68,'2026-10-03 00:40:26.212','2026-10-03 00:40:26.212',NULL,'/v1/sys/organization/positions','删除岗位','organization','DELETE'),(69,'2026-10-03 00:40:26.213','2026-10-03 00:40:26.213',NULL,'/v1/sys/organization/membership','查询用户部门岗位','organization','GET'),(70,'2026-10-03 00:40:26.214','2026-10-03 00:40:26.214',NULL,'/v1/sys/organization/membership','更新用户部门岗位','organization','PUT'),(71,'2026-10-03 00:40:26.215','2026-10-03 00:40:26.215',NULL,'/v1/sys/organization/dataScope','查询角色数据范围','organization','GET'),(72,'2026-10-03 00:40:26.215','2026-10-03 00:40:26.215',NULL,'/v1/sys/organization/dataScope','更新角色数据范围','organization','PUT'),(73,'2026-10-03 00:40:26.221','2026-10-03 00:40:26.221',NULL,'/v1/sys/session/admin/devices','查询用户设备会话','session','GET'),(74,'2026-10-03 00:40:26.222','2026-10-03 00:40:26.222',NULL,'/v1/sys/session/admin/device','撤销用户设备会话','session','DELETE'),(75,'2026-10-03 14:36:03.794','2026-10-03 14:36:03.794',NULL,'/v1/ai/info','查询AI助手配置','ai-agent','GET'),(76,'2026-10-03 14:36:03.794','2026-10-03 14:36:03.794',NULL,'/v1/ai/runs','提交AI任务','ai-agent','POST'),(77,'2026-10-03 14:36:03.794','2026-10-03 14:36:03.794',NULL,'/v1/ai/runs/:id','查询本人AI任务','ai-agent','GET'),(78,'2026-10-03 14:36:03.794','2026-10-03 14:36:03.794',NULL,'/v1/ai/runs/:id/cancel','取消本人AI任务','ai-agent','POST'),(79,'2026-10-03 14:36:03.794','2026-10-03 14:36:03.794',NULL,'/v1/ai/conversations','查询本人AI会话','ai-agent','GET'),(80,'2026-10-03 14:36:03.794','2026-10-03 14:36:03.794',NULL,'/v1/ai/conversations/:id/messages','查询本人AI会话消息','ai-agent','GET'),(81,'2026-10-09 22:46:29.332','2026-10-09 22:46:29.332',NULL,'/v1/sys/permissions/edit','读取四类授权的统一资源与选择快照','permission','GET'),(82,'2026-10-09 22:46:29.333','2026-10-09 22:46:29.333',NULL,'/v1/sys/menu/previewMove','预览菜单移动新增可见入口','menu','POST');
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
  `actor_name` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `authority_id` bigint NOT NULL DEFAULT '0',
  `event_type` varchar(16) COLLATE utf8mb4_general_ci NOT NULL,
  `module` varchar(64) COLLATE utf8mb4_general_ci NOT NULL,
  `action` varchar(64) COLLATE utf8mb4_general_ci NOT NULL,
  `object` varchar(256) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `path` varchar(256) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `method` varchar(16) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `result` varchar(16) COLLATE utf8mb4_general_ci NOT NULL,
  `status_code` bigint NOT NULL DEFAULT '0',
  `ip` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `trace_id` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `duration_ms` bigint NOT NULL DEFAULT '0',
  `params` text COLLATE utf8mb4_general_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_audit_time` (`created_at`),
  KEY `idx_audit_actor_time` (`actor_id`,`created_at`),
  KEY `idx_audit_type_time` (`event_type`,`created_at`),
  KEY `idx_audit_module_time` (`module`,`created_at`),
  KEY `idx_audit_result_time` (`result`,`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=30 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_audit_logs`
--

LOCK TABLES `sys_audit_logs` WRITE;
/*!40000 ALTER TABLE `sys_audit_logs` DISABLE KEYS */;
INSERT INTO `sys_audit_logs` VALUES (1,0,'',0,'operation','menu','seedFrontendModules','20261003_frontend_modules.sql','','','success',0,'','',0,'{}','2026-10-03 00:53:47.544'),(2,1,'admin',1,'operation','session','createDeviceSession','1e268a72-a00a-4b9a-97aa-2240209494f6','/v1/sys/login','POST','success',0,'127.0.0.1','5e6f1143f2eb706160333260aecd191e',0,'{}','2026-10-03 03:00:32.474'),(3,1,'admin',1,'login','user','login','','/v1/sys/login','POST','success',200,'127.0.0.1','5e6f1143f2eb706160333260aecd191e',231,'{}','2026-10-03 03:00:32.499'),(4,0,'',0,'operation','ai-agent','seedAgentModule','20261003_zz_ai_agent.sql','','','success',0,'','',0,'{}','2026-10-03 14:36:03.811'),(5,0,'',0,'operation','sys','runs','','/v1/ai/runs','POST','failure',200,'127.0.0.1','ae90b0bbfd230307eeba3b91f5955ebc',1,'{}','2026-10-03 14:41:59.843'),(6,1,'admin',1,'operation','session','createDeviceSession','021fe8d4-acd3-406e-a105-054f90af35a1','/v1/sys/login','POST','success',0,'127.0.0.1','3eaeec9efce3e977b459db2d8e6c2944',0,'{}','2026-10-03 14:56:19.302'),(7,1,'admin',1,'login','user','login','','/v1/sys/login','POST','success',200,'127.0.0.1','3eaeec9efce3e977b459db2d8e6c2944',168,'{}','2026-10-03 14:56:19.322'),(8,0,'',0,'operation','sys','runs','','/v1/ai/runs','POST','failure',200,'127.0.0.1','d64ec2d2607f123a45b5d7123f604dc3',0,'{}','2026-10-03 15:12:33.414'),(9,0,'',0,'operation','sys','runs','','/v1/ai/runs','POST','failure',200,'127.0.0.1','627638668c24a8828f7f1ae4ad680b04',0,'{}','2026-10-03 19:45:12.418'),(10,0,'',0,'operation','runs','cancel','','/v1/ai/runs/00000000-0000-4000-8000-000000000001/cancel','POST','failure',200,'127.0.0.1','ab6a3ef3aa5faf012c64f1e8011f35a7',0,'{}','2026-10-03 19:45:12.441'),(11,1,'admin',1,'operation','session','createDeviceSession','1e17de9b-ed5e-4b48-bff3-c128b2f34071','/v1/sys/login','POST','success',0,'127.0.0.1','f90f9424a305d3d965eb3ff3a3279de6',0,'{}','2026-10-04 13:52:35.526'),(12,1,'admin',1,'login','user','login','','/v1/sys/login','POST','success',200,'127.0.0.1','f90f9424a305d3d965eb3ff3a3279de6',143,'{}','2026-10-04 13:52:35.534'),(13,1,'',1,'operation','ai-agent','submitRun','95014f8e-efa3-4851-9ea7-a0428f62ed1f','/v1/ai/runs','POST','success',0,'127.0.0.1','b08bf175310251ebdfa8d1df8a2fa457',0,'{}','2026-10-04 13:52:54.263'),(14,1,'admin',1,'operation','sys','runs','','/v1/ai/runs','POST','success',200,'127.0.0.1','b08bf175310251ebdfa8d1df8a2fa457',44,'{}','2026-10-04 13:52:54.273'),(15,1,'',1,'operation','ai-agent','submitRun','1090cf40-5023-46ee-998e-608897423601','/v1/ai/runs','POST','success',0,'127.0.0.1','1b683c50c3a2b2c3bf3040e1c6065c89',0,'{}','2026-10-04 13:53:13.731'),(16,1,'admin',1,'operation','sys','runs','','/v1/ai/runs','POST','success',200,'127.0.0.1','1b683c50c3a2b2c3bf3040e1c6065c89',38,'{}','2026-10-04 13:53:13.739'),(17,1,'',1,'operation','ai-agent','submitRun','839d1d99-6299-4f09-bf2b-6fae18646b71','/v1/ai/runs','POST','success',0,'127.0.0.1','7ad1f8f5a6916385feffb55fb94318e8',0,'{}','2026-10-04 13:53:40.524'),(18,1,'admin',1,'operation','sys','runs','','/v1/ai/runs','POST','success',200,'127.0.0.1','7ad1f8f5a6916385feffb55fb94318e8',37,'{}','2026-10-04 13:53:40.533'),(19,1,'',1,'operation','ai-agent','submitRun','fb4e0b1d-2d84-4593-bb1d-4ad69d455890','/v1/ai/runs','POST','success',0,'127.0.0.1','9439465b7c426990ad02f1772d7b4e05',0,'{}','2026-10-04 13:54:42.169'),(20,1,'admin',1,'operation','sys','runs','','/v1/ai/runs','POST','success',200,'127.0.0.1','9439465b7c426990ad02f1772d7b4e05',42,'{}','2026-10-04 13:54:42.182'),(21,1,'',1,'operation','ai-agent','submitRun','2ce775be-00c5-4d60-b574-9bb84a6dee61','/v1/ai/runs','POST','success',0,'127.0.0.1','0f716b0774e969be8b7b36af85282084',0,'{}','2026-10-04 14:10:06.512'),(22,1,'admin',1,'operation','sys','runs','','/v1/ai/runs','POST','success',200,'127.0.0.1','0f716b0774e969be8b7b36af85282084',62,'{}','2026-10-04 14:10:06.519'),(23,0,'',0,'operation','sys','runs','','/v1/ai/runs','POST','failure',200,'127.0.0.1','555eea28d20cf5207876ebcc49e891a5',0,'{}','2026-10-04 14:29:04.371'),(24,0,'',0,'operation','runs','cancel','','/v1/ai/runs/00000000-0000-4000-8000-000000000001/cancel','POST','failure',200,'127.0.0.1','b17a83b39733b6e323cf04db4628a211',0,'{}','2026-10-04 14:29:04.400'),(25,1,'admin',1,'operation','session','createDeviceSession','e773f437-ecf5-4992-8234-bc0ba81a8843','/v1/sys/login','POST','success',0,'127.0.0.1','11f1d16fdc93d397ff51dc7b6f3bcc3b',0,'{}','2026-10-09 22:55:01.185'),(26,1,'admin',1,'login','user','login','','/v1/sys/login','POST','success',200,'127.0.0.1','11f1d16fdc93d397ff51dc7b6f3bcc3b',188,'{}','2026-10-09 22:55:01.196'),(27,1,'admin',1,'operation','base','sendEmailCode','','/v1/sys/base/sendEmailCode','POST','failure',200,'127.0.0.1','74ae670aae9046e9fdd464905570a726',12,'{}','2026-10-09 22:56:12.076'),(28,1,'admin',1,'operation','session','createDeviceSession','2249840a-5916-4309-842c-af403c2233d9','/v1/sys/login','POST','success',0,'127.0.0.1','e97d20b15bcb2b02ff121fca5ec82266',0,'{}','2026-10-19 19:34:49.660'),(29,1,'admin',1,'login','user','login','','/v1/sys/login','POST','success',200,'127.0.0.1','e97d20b15bcb2b02ff121fca5ec82266',204,'{}','2026-10-19 19:34:49.690');
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
  `default_router` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'dashboard' COMMENT '默认菜单',
  PRIMARY KEY (`authority_id`) USING BTREE,
  UNIQUE KEY `authority_id` (`authority_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1790456426 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_authorities`
--

LOCK TABLES `sys_authorities` WRITE;
/*!40000 ALTER TABLE `sys_authorities` DISABLE KEYS */;
INSERT INTO `sys_authorities` VALUES ('2024-01-15 13:14:52.000','2024-01-16 17:49:10.662',NULL,1,'超级用户',0,'authority'),('2024-01-16 16:56:06.053','2024-01-16 17:27:37.537',NULL,10,'测试角色',0,'index'),('2024-01-16 16:56:06.053','2024-01-24 17:43:35.123',NULL,102,'测试角色102',0,'index');
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
INSERT INTO `sys_authority_btns` VALUES (1,5,23),(1,5,24),(1,5,25),(1,2,26),(1,2,27),(1,2,28),(1,3,29),(1,3,30),(1,3,31),(1,4,32),(1,4,33),(1,4,34),(1,6,35),(1,6,36),(1,6,37),(1,5,38),(1,2,39),(1,2,40),(1,2,41),(1,6,42),(1,46,43),(1,46,44),(1,4,54),(1,49,57),(1,49,58),(1,49,59),(1,52,60),(1,50,61),(1,50,62),(1,50,63),(1,51,64),(1,51,65),(1,51,66),(1,5,67),(1,2,68),(1,53,69),(1,53,70);
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
INSERT INTO `sys_authority_menus` VALUES (1,1),(1,10),(1,102),(2,1),(2,10),(2,102),(3,1),(3,10),(3,102),(4,1),(4,10),(4,102),(5,1),(5,10),(5,102),(6,1),(6,10),(6,102),(7,1),(7,10),(7,102),(39,1),(39,10),(39,102),(40,1),(40,10),(41,10),(41,102),(42,10),(43,1),(43,10),(46,1),(48,1),(49,1),(50,1),(51,1),(52,1),(53,1);
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
) ENGINE=InnoDB AUTO_INCREMENT=71 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_base_menu_btns`
--

LOCK TABLES `sys_base_menu_btns` WRITE;
/*!40000 ALTER TABLE `sys_base_menu_btns` DISABLE KEYS */;
INSERT INTO `sys_base_menu_btns` VALUES (23,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',5,'user:create'),(24,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',5,'user:update'),(25,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',5,'user:delete'),(26,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',2,'authority:create'),(27,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',2,'authority:update'),(28,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',2,'authority:delete'),(29,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',3,'menu:create'),(30,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',3,'menu:update'),(31,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',3,'menu:delete'),(32,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',4,'api:create'),(33,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',4,'api:update'),(34,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',4,'api:delete'),(35,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',6,'dictionary:create'),(36,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',6,'dictionary:update'),(37,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',6,'dictionary:delete'),(38,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'resetPassword','重置密码',5,'user:resetPassword'),(39,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'menus','菜单授权',2,'authority:menus'),(40,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'apis','接口授权',2,'authority:apis'),(41,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'buttons','按钮授权',2,'authority:buttons'),(42,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'items','字典项管理',6,'dictionary:items'),(43,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'upload','上传图片',46,'businessTools:upload'),(44,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'sendEmail','发送邮箱验证码',46,'businessTools:sendEmail'),(54,'2026-09-26 01:01:09.006','2026-09-26 01:01:09.006',NULL,'sync','同步接口资源',4,'api:sync'),(57,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'upload','上传文件',49,'files:upload'),(58,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'delete','删除文件',49,'files:delete'),(59,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'reference','管理文件引用',49,'files:reference'),(60,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'revoke','撤销设备会话',52,'sessions:revoke'),(61,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'create','新增部门',50,'organization-departments:create'),(62,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'update','编辑部门',50,'organization-departments:update'),(63,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'delete','删除部门',50,'organization-departments:delete'),(64,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'create','新增岗位',51,'organization-positions:create'),(65,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'update','编辑岗位',51,'organization-positions:update'),(66,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'delete','删除岗位',51,'organization-positions:delete'),(67,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'membership','分配部门岗位',5,'user:membership'),(68,'2026-10-03 00:53:47.541','2026-10-03 00:53:47.541',NULL,'dataScope','设置数据权限',2,'authority:dataScope'),(69,'2026-10-03 14:36:03.804','2026-10-03 14:36:03.804',NULL,'run','提交AI任务',53,'ai-agent:run'),(70,'2026-10-03 14:36:03.804','2026-10-03 14:36:03.804',NULL,'cancel','取消AI任务',53,'ai-agent:cancel');
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
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
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
) ENGINE=InnoDB AUTO_INCREMENT=54 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=REDUNDANT;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_base_menus`
--

LOCK TABLES `sys_base_menus` WRITE;
/*!40000 ALTER TABLE `sys_base_menus` DISABLE KEYS */;
INSERT INTO `sys_base_menus` (`id`, `created_at`, `updated_at`, `deleted_at`, `menu_level`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`) VALUES (1,'2023-11-17 15:00:25.000','2024-01-18 09:54:30.123',NULL,0,0,'admin','superAdmin',0,'views/superAdmin/index.vue',1,'',0,0,'超级管理员','StopFilled',0),(2,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,0,1,'authority','authority',0,'views/superAdmin/authority/authority.vue',1,'',0,0,'角色管理','CheckSquareOutlined',0),(3,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,0,1,'menu','menu',0,'views/superAdmin/menu/menu.vue',2,'',1,0,'菜单管理','BugOutlined',0),(4,'2023-11-17 15:00:25.093','2024-01-10 11:41:33.339',NULL,0,1,'api','api',0,'views/superAdmin/api/api.vue',3,'',1,0,'api管理','ColumnHeightOutlined',0),(5,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,0,1,'user','user',0,'views/superAdmin/user/user.vue',4,'',0,0,'用户管理','BugOutlined',0),(6,'2023-11-17 15:00:25.093','2024-01-12 10:03:17.383',NULL,0,1,'dictionary','dictionary',0,'views/superAdmin/dictionary/dictionary.vue',5,'',0,0,'字典管理','BugOutlined',0),(7,'2024-01-16 17:27:17.123','2026-09-25 22:21:37.867',NULL,0,0,'index','index',0,'views/index.vue',0,'',0,0,'首页','AntDesignOutlined',0),(39,'2024-01-17 14:12:08.123','2024-01-18 10:11:35.493',NULL,0,0,'11','test1',0,'views/test/index.vue',3,'',0,0,'test1','',0),(40,'2024-01-17 14:12:21.554','2024-01-17 14:12:21.554',NULL,0,39,'test2','test2',0,'views/routerHolder.vue',0,'',0,0,'test2','',0),(41,'2024-01-17 15:38:46.677','2024-01-17 15:38:46.677',NULL,0,39,'test3','test3',0,'views/routerHolder.vue',0,'',0,0,'test3','AccountBookOutlined',0),(42,'2024-01-17 16:05:34.342','2024-01-17 16:05:34.342',NULL,0,40,'test4','test4',0,'views/routerHolder.vue',0,'',0,0,'test4','AccountBookTwoTone',0),(43,'2024-01-17 16:05:57.207','2024-01-17 16:05:57.207',NULL,0,40,'test5','test5',0,'views/routerHolder.vue',0,'',0,0,'test5','AccountBookOutlined',0),(46,'2026-09-25 20:09:55.894','2026-09-25 20:09:55.894',NULL,0,1,'business-tools','businessTools',0,'views/business/tools/index.vue',90,'',0,0,'管理工具','lucide:wrench',0),(47,'2026-09-26 01:05:12.749','2026-09-26 01:05:12.828','2026-09-26 01:05:14.268',0,0,'smoke_1790355912584','smoke_1790355912584',1,'views/index.vue',0,'',0,0,'smoke_1790355912584_edited','lucide:flask-conical',0),(48,'2026-10-03 00:53:47.535','2026-10-03 00:53:47.535',NULL,0,1,'audit','audit',0,'views/business/system/audit/index.vue',60,'',0,0,'操作审计','',0),(49,'2026-10-03 00:53:47.535','2026-10-03 00:53:47.535',NULL,0,1,'files','files',0,'views/business/system/files/index.vue',61,'',0,0,'文件资源','',0),(50,'2026-10-03 00:53:47.535','2026-10-03 00:53:47.535',NULL,0,1,'organization/departments','organization-departments',0,'views/business/system/organization/departments.vue',63,'',0,0,'部门管理','',0),(51,'2026-10-03 00:53:47.535','2026-10-03 00:53:47.535',NULL,0,1,'organization/positions','organization-positions',0,'views/business/system/organization/positions.vue',64,'',0,0,'岗位管理','',0),(52,'2026-10-03 00:53:47.535','2026-10-03 00:53:47.535',NULL,0,1,'sessions','sessions',0,'views/business/system/sessions/index.vue',62,'',0,0,'在线会话','',0),(53,'2026-10-03 14:36:03.799','2026-10-03 14:36:03.799',NULL,0,1,'ai-agent','ai-agent',0,'views/business/ai-agent/index.vue',65,'',0,0,'AI助手','',0);
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
  `name` varchar(64) COLLATE utf8mb4_general_ci NOT NULL,
  `code` varchar(64) COLLATE utf8mb4_general_ci NOT NULL,
  `sort` bigint NOT NULL DEFAULT '0',
  `status` bigint NOT NULL DEFAULT '1',
  `leader` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_department_code` (`code`),
  KEY `idx_department_parent` (`parent_id`),
  KEY `idx_department_deleted` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_departments`
--

LOCK TABLES `sys_departments` WRITE;
/*!40000 ALTER TABLE `sys_departments` DISABLE KEYS */;
/*!40000 ALTER TABLE `sys_departments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_device_sessions`
--

DROP TABLE IF EXISTS `sys_device_sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_device_sessions` (
  `id` varchar(36) COLLATE utf8mb4_general_ci NOT NULL,
  `user_id` bigint NOT NULL,
  `authority_id` bigint NOT NULL,
  `session_version` bigint NOT NULL,
  `ip` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `user_agent` varchar(512) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
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
INSERT INTO `sys_device_sessions` VALUES ('021fe8d4-acd3-406e-a105-054f90af35a1',1,1,1,'127.0.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36','2026-10-03 14:56:19.290','2026-10-03 16:56:19.290',NULL),('1e17de9b-ed5e-4b48-bff3-c128b2f34071',1,1,1,'127.0.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36','2026-10-04 13:52:35.522','2026-10-04 15:52:35.522',NULL),('1e268a72-a00a-4b9a-97aa-2240209494f6',1,1,1,'127.0.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36','2026-10-03 03:00:32.463','2026-10-03 05:00:32.463',NULL),('2249840a-5916-4309-842c-af403c2233d9',1,1,1,'127.0.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36','2026-10-19 19:34:49.646','2026-10-19 21:34:49.646',NULL),('e773f437-ecf5-4992-8234-bc0ba81a8843',1,1,1,'127.0.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36','2026-10-09 22:55:01.180','2026-10-10 00:55:01.180',NULL);
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
) ENGINE=InnoDB AUTO_INCREMENT=36 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dictionaries`
--

LOCK TABLES `sys_dictionaries` WRITE;
/*!40000 ALTER TABLE `sys_dictionaries` DISABLE KEYS */;
INSERT INTO `sys_dictionaries` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `type`, `status`, `desc`) VALUES (1,'2023-11-17 15:00:24.877','2023-11-17 15:00:24.896',NULL,'性别','gender',1,'性别字典'),(2,'2024-01-16 17:38:10.566','2024-01-16 17:38:10.566',NULL,'api请求','method',1,'api请求'),(34,'2026-09-25 20:19:36.684','2026-09-25 20:40:04.524','2026-09-25 20:40:04.524','联调测试_UI_20260925','ui_smoke_20260925',1,''),(35,'2026-09-26 01:05:13.851','2026-09-26 01:05:13.905','2026-09-26 01:05:14.216','smoke_1790355912584_edited','smoke_1790355912584',1,'');
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
) ENGINE=InnoDB AUTO_INCREMENT=51 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dictionary_info`
--

LOCK TABLES `sys_dictionary_info` WRITE;
/*!40000 ALTER TABLE `sys_dictionary_info` DISABLE KEYS */;
INSERT INTO `sys_dictionary_info` (`id`, `created_at`, `updated_at`, `deleted_at`, `label`, `value`, `extend`, `status`, `sort`, `sys_dictionary_id`) VALUES (1,'2024-01-16 17:37:18.206','2024-01-16 17:37:18.206',NULL,'男',1,'男-拓展值',1,0,1),(2,'2024-01-16 17:37:35.413','2024-01-16 17:37:35.413',NULL,'女',2,'女-拓展值',1,2,1),(3,'2024-01-16 17:38:19.939','2024-01-18 17:26:35.798',NULL,'POST',1,'POST',1,0,2),(4,'2024-01-16 17:38:36.624','2024-01-16 17:38:36.624',NULL,'GET',2,'GET',1,0,2),(5,'2024-01-16 17:38:45.246','2024-01-16 17:38:45.246',NULL,'PUT',3,'PUT',1,0,2),(6,'2024-01-16 17:38:56.280','2024-01-16 17:38:56.280',NULL,'DELETE',4,'DELETE',1,0,2),(49,'2026-09-25 20:24:46.665','2026-09-25 20:40:04.519','2026-09-25 20:40:04.519','零值联调',0,'',1,0,34),(50,'2026-09-26 01:05:13.949','2026-09-26 01:05:13.998','2026-09-26 01:05:14.183','smoke_1790355912584',0,'',1,0,35);
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_file_references`
--

LOCK TABLES `sys_file_references` WRITE;
/*!40000 ALTER TABLE `sys_file_references` DISABLE KEYS */;
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
  `name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL,
  `mime` varchar(128) COLLATE utf8mb4_general_ci NOT NULL,
  `size` bigint NOT NULL,
  `owner_id` bigint NOT NULL,
  `department_id` bigint NOT NULL DEFAULT '0',
  `visibility` varchar(16) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'public',
  `status` varchar(16) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'active',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_file_object_key` (`object_key`),
  KEY `idx_file_owner_status` (`owner_id`,`status`),
  KEY `idx_file_department_status` (`department_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_file_resources`
--

LOCK TABLES `sys_file_resources` WRITE;
/*!40000 ALTER TABLE `sys_file_resources` DISABLE KEYS */;
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
INSERT INTO `sys_permission_versions` VALUES (1,2);
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
INSERT INTO `sys_policy_versions` VALUES (1,3,'2026-10-09 22:46:29.334');
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
  `name` varchar(64) COLLATE utf8mb4_general_ci NOT NULL,
  `code` varchar(64) COLLATE utf8mb4_general_ci NOT NULL,
  `sort` bigint NOT NULL DEFAULT '0',
  `status` bigint NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_position_code` (`code`),
  KEY `idx_position_deleted` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_positions`
--

LOCK TABLES `sys_positions` WRITE;
/*!40000 ALTER TABLE `sys_positions` DISABLE KEYS */;
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
  `scope` varchar(32) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'self',
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`authority_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_role_data_scopes`
--

LOCK TABLES `sys_role_data_scopes` WRITE;
/*!40000 ALTER TABLE `sys_role_data_scopes` DISABLE KEYS */;
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
INSERT INTO `sys_user_authority` VALUES (1,1),(2,102),(3,1),(62,10),(63,10),(64,10),(65,10);
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
  `header_img` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '用户头像',
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
) ENGINE=InnoDB AUTO_INCREMENT=77 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_users`
--

LOCK TABLES `sys_users` WRITE;
/*!40000 ALTER TABLE `sys_users` DISABLE KEYS */;
INSERT INTO `sys_users` (`id`, `created_at`, `updated_at`, `deleted_at`, `uuid`, `username`, `password`, `nick_name`, `side_mode`, `header_img`, `base_color`, `active_color`, `authority_id`, `phone`, `email`, `enable`, `session_version`) VALUES (1,'2024-01-10 11:54:46.167','2024-01-10 11:54:46.167',NULL,'e2ab86b1-8e3d-4864-80ad-e83131353686','admin','$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m','Mr.root','dark','\r\nhttps://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#000','#fff',1,'15888888888','TypeScript@111.com',1,1),(2,'2024-01-10 09:27:22.905','2024-01-24 17:43:18.347',NULL,'e2ab86b1-8e3d-4864-80ad-e83131353686','yanghao','$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m','Mr.杨浩','dark','\r\nhttps://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#000','#fff',102,'15888888888','TypeScript@111.com',1,1),(3,'2024-01-10 09:26:44.047','2024-01-17 16:16:34.486',NULL,'e2ab86b1-8e3d-4864-80ad-e83131353686','wenlong','$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m','Mr.文龙','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705479393275_49814642.jfif','#000','#fff',1,'15888888888','TypeScript@111.com',1,1),(62,'2024-01-17 10:08:17.097','2024-01-17 10:08:17.097',NULL,'f3149922-22b3-4866-ad1f-e3d58ae4d9a5','用户001','$2a$10$fpVOkDRMLD3V9l34VfIN6uSvzOhgARViFsKk6pUehoYt//n..2F9C','用户001','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#fff',888,'','',1,1),(63,'2024-01-17 11:05:43.707','2024-01-17 11:05:43.707',NULL,'db637c17-6461-4c93-b1ea-18b33d8b0b09','11','$2a$10$xH2J8jDZC0XpD.jefci4l.hgLtN7fNZbDevh6nIwuZeQFb17kS4US','11','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',888,'','',1,1),(64,'2024-01-17 11:05:59.012','2024-01-17 11:05:59.012',NULL,'bfaba08a-4e9c-48b3-aaa5-fa8ebfe9d585','22','$2a$10$LnpbXmU51bWHhmRSVKusb.mXG7MHtlpYAfvNFQ2Dw.DxOSVSQManC','22','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',888,'','',1,1),(65,'2024-01-17 11:07:41.376','2024-01-17 11:07:41.376',NULL,'d8f57f39-e366-4895-ad8e-d2af02321d80','33','$2a$10$woWz6ZlbewPKbNPKmNI/9.KhLhjpClPtPeLaSroDWCk6UQPdju8Om','33','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',888,'','',1,1),(66,'2026-09-26 01:05:13.162','2026-09-26 01:05:13.441','2026-09-26 01:05:14.092','ec42fbcd-b9db-45a6-9281-e17578e2ab85','smoke_1790355912584','$2a$10$kpO1XwepfOfoWKj5MAbEPeH1FUgSNad.h8gyuQXAjy6YoDlFQgzV2','smoke_1790355912584_edited','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790455912,'','',1,4),(67,'2026-09-26 01:05:39.798','2026-09-26 01:06:51.733','2026-09-26 01:10:37.320','e211e55f-3d98-48e5-8198-786f7c6f86a3','session_smoke_1790355939619_2343_profile','$2a$10$vtE8fMt8glVE2UjdPdP7wulW6X7Dpjqqgkip0IIW04NIsAxdTXf1W','Updated display','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790455939,'','',1,1),(68,'2026-09-26 01:06:51.822','2026-09-26 01:07:14.057','2026-09-26 01:10:37.274','a982e7c0-3dd2-4a27-91cd-f823854a7a64','session_smoke_1790355939619_2343_password','$2a$10$.bInErEqVv8bWlXpDIgZc.0ovSoQUP/ytwhfEvRilHCIIn8z/j/7O','session_smoke_1790355939619_2343_password','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790455939,'','',1,2),(69,'2026-09-26 01:07:32.488','2026-09-26 01:07:49.541','2026-09-26 01:10:37.244','293edf1e-e9ea-45d6-bdfa-2aebda07d087','session_smoke_1790355939619_2343_freeze','$2a$10$EvNzD2KWRM8A1ObKKieo1uHtvdGkXSuP9IO1vFOq8I.8GF7wgP3ey','session_smoke_1790355939619_2343_freeze','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790455939,'','',1,3),(70,'2026-09-26 01:07:49.628','2026-09-26 01:07:49.628','2026-09-26 01:10:37.198','819fcc9b-85e3-41a0-bccd-ce4fec83b52e','session_smoke_1790355939619_2343_default_role','$2a$10$nktAVUVDqmlJpzLypf2tm.pqvOQb3pGtBXdA84erxCU6hTNcZjWZy','session_smoke_1790355939619_2343_default_role','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790455939,'','',1,1),(71,'2026-09-26 01:08:08.551','2026-09-26 01:08:56.100','2026-09-26 01:10:37.152','37552a7a-c923-4e5f-bab5-9301613ca308','session_smoke_1790355939619_2343_linked_role','$2a$10$WCcv2isSFiZT49FXTh7jWeD4O/LpWnOKBq6Ev/3umSLZDiY4z5Yee','session_smoke_1790355939619_2343_linked_role','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790455939,'','',1,2),(72,'2026-09-26 01:08:56.225','2026-09-26 01:08:56.225','2026-09-26 01:10:37.116','90820b68-6b6a-4dbf-8a60-29b4aafe3cff','session_smoke_1790355939619_2343_reset','$2a$10$ZcATEuKXRmXEwyDS0GvUx.okgqf6s1Gpfi27.kg8U3xqSndwR7joS','session_smoke_1790355939619_2343_reset','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790455939,'','',1,1),(73,'2026-09-26 01:09:11.023','2026-09-26 01:10:07.411','2026-09-26 01:10:37.052','aad81c5c-7f75-4da0-8a94-e71989e68b05','session_smoke_1790355939619_2343_logout','$2a$10$/xLZt4rE8XCXO23/es5k.OB6hEBct/EgqUX6q83QXdPlV/FLtqPFO','session_smoke_1790355939619_2343_logout','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790455939,'','',1,2),(74,'2026-09-26 01:10:23.430','2026-09-26 01:10:23.430','2026-09-26 01:10:36.950','2c5c07f6-34c5-482a-b6e5-db572fdd3406','session_smoke_1790355939619_2343_delete','$2a$10$In/JpeDWOTL4udmTcP0qH.zARuUliB20JM9tfNQvQEkLXqjdGv40u','session_smoke_1790355939619_2343_delete','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790455939,'','',1,1),(75,'2026-09-26 01:13:44.199','2026-09-26 01:14:19.952','2026-09-26 01:14:46.984','f78d68aa-4be6-436e-9d1d-fb6b47181843','session_smoke_1790356424019_8429_default_role','$2a$10$8yu3C8lLyVFPH8RMucQaxeKDbmohtGsDjCk0k6f48hRf06ert0Dye','session_smoke_1790356424019_8429_default_role','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790456425,'','',1,2),(76,'2026-09-26 01:14:20.066','2026-09-26 01:14:33.910','2026-09-26 01:14:46.940','3fa14d0d-a7a0-43ee-b9cf-91dfb97cfdec','session_smoke_1790356424019_8429_reset','$2a$10$dXb6u3nCLzK9fS5BwUUk0emoIIMK2BQFe4sHTX8Ib7I7VyKOVfgZa','session_smoke_1790356424019_8429_reset','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',1790456424,'','',1,2);
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

-- Dump completed on 2026-10-19 23:55:32
