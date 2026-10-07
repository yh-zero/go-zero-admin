-- go-zero-admin 当前完整初始化 SQL（2026-10-07）
-- MySQL 8；在已创建且选定的空数据库中导入一次，然后启动项目。
-- 包含完整表结构、必要初始数据及截至 20261003_zz_ai_agent.sql 的迁移记录。
-- 初始管理员：admin / 123456。首次登录后修改密码。
-- 不包含测试账号、测试角色、测试菜单、审计日志、设备会话或 AI 历史数据。
-- 已有数据库升级使用 data/db/migrations/；不要向已有业务库导入全量 SQL。
-- 不创建/切换数据库，不删除已有表；导入到非空库会报表已存在错误。


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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `casbin_rule` (`id`, `ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES (251,'p','1','/v1/ai/conversations','GET',NULL,NULL,NULL),(252,'p','1','/v1/ai/conversations/:id/messages','GET',NULL,NULL,NULL),(247,'p','1','/v1/ai/info','GET',NULL,NULL,NULL),(248,'p','1','/v1/ai/runs','POST',NULL,NULL,NULL),(249,'p','1','/v1/ai/runs/:id','GET',NULL,NULL,NULL),(250,'p','1','/v1/ai/runs/:id/cancel','POST',NULL,NULL,NULL),(223,'p','1','/v1/sys/api/applySync','POST',NULL,NULL,NULL),(64,'p','1','/v1/sys/api/createApi','POST','','',''),(65,'p','1','/v1/sys/api/deleteApi','DELETE','','',''),(67,'p','1','/v1/sys/api/deleteApisByIds','DELETE','','',''),(66,'p','1','/v1/sys/api/getAllApiList','GET','','',''),(63,'p','1','/v1/sys/api/getApiList','GET','','',''),(222,'p','1','/v1/sys/api/previewSync','GET',NULL,NULL,NULL),(68,'p','1','/v1/sys/api/updateApi','PUT','','',''),(225,'p','1','/v1/sys/audit/getAuditLogList','GET',NULL,NULL,NULL),(59,'p','1','/v1/sys/authority/addAuthorityMenu','POST','','',''),(61,'p','1','/v1/sys/authority/createAuthority','POST','','',''),(62,'p','1','/v1/sys/authority/deleteAuthority','DELETE','','',''),(58,'p','1','/v1/sys/authority/getAuthorityList','GET','','',''),(60,'p','1','/v1/sys/authority/updateAuthority','PUT','','',''),(84,'p','1','/v1/sys/base/sendEmailCode','POST','','',''),(82,'p','1','/v1/sys/base/uploadFileImg','POST','','',''),(79,'p','1','/v1/sys/casbin/getPathByAuthorityId','GET','','',''),(80,'p','1','/v1/sys/casbin/updateCasbinData','PUT','','',''),(81,'p','1','/v1/sys/casbin/updateCasbinDataByApiIds','PUT','','',''),(49,'p','1','/v1/sys/deleteUser','DELETE','','',''),(70,'p','1','/v1/sys/dictionary/createSysDictionary','POST','','',''),(77,'p','1','/v1/sys/dictionary/createSysDictionaryInfo','POST','','',''),(73,'p','1','/v1/sys/dictionary/deleteSysDictionary','DELETE','','',''),(78,'p','1','/v1/sys/dictionary/deleteSysDictionaryInfo','DELETE','','',''),(71,'p','1','/v1/sys/dictionary/getSysDictionaryDetails','GET','','',''),(74,'p','1','/v1/sys/dictionary/getSysDictionaryInfoList','GET','','',''),(75,'p','1','/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','GET','','',''),(83,'p','1','/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','GET','','',''),(69,'p','1','/v1/sys/dictionary/getSysDictionaryList','GET','','',''),(72,'p','1','/v1/sys/dictionary/updateSysDictionary','PUT','','',''),(76,'p','1','/v1/sys/dictionary/updateSysDictionaryInfo','PUT','','',''),(226,'p','1','/v1/sys/files/list','GET',NULL,NULL,NULL),(227,'p','1','/v1/sys/files/reference','DELETE',NULL,NULL,NULL),(228,'p','1','/v1/sys/files/reference','POST',NULL,NULL,NULL),(229,'p','1','/v1/sys/files/resource','DELETE',NULL,NULL,NULL),(230,'p','1','/v1/sys/files/url','GET',NULL,NULL,NULL),(45,'p','1','/v1/sys/getUserList','GET','','',''),(43,'p','1','/v1/sys/login','POST','','',''),(52,'p','1','/v1/sys/menu/addBaseMenu','POST','','',''),(57,'p','1','/v1/sys/menu/deleteBaseMenu','DELETE','','',''),(219,'p','1','/v1/sys/menu/getAuthorityButtons','GET',NULL,NULL,NULL),(55,'p','1','/v1/sys/menu/getBaseMenuById','GET','','',''),(53,'p','1','/v1/sys/menu/getBaseMenuTree','GET','','',''),(50,'p','1','/v1/sys/menu/getMenu','GET','','',''),(54,'p','1','/v1/sys/menu/getMenuAuthority','GET','','',''),(51,'p','1','/v1/sys/menu/getMenuList','GET','','',''),(220,'p','1','/v1/sys/menu/updateAuthorityButtons','PUT',NULL,NULL,NULL),(56,'p','1','/v1/sys/menu/updateBaseMenu','PUT','','',''),(243,'p','1','/v1/sys/organization/dataScope','GET',NULL,NULL,NULL),(244,'p','1','/v1/sys/organization/dataScope','PUT',NULL,NULL,NULL),(236,'p','1','/v1/sys/organization/departments','DELETE',NULL,NULL,NULL),(233,'p','1','/v1/sys/organization/departments','GET',NULL,NULL,NULL),(234,'p','1','/v1/sys/organization/departments','POST',NULL,NULL,NULL),(235,'p','1','/v1/sys/organization/departments','PUT',NULL,NULL,NULL),(241,'p','1','/v1/sys/organization/membership','GET',NULL,NULL,NULL),(242,'p','1','/v1/sys/organization/membership','PUT',NULL,NULL,NULL),(240,'p','1','/v1/sys/organization/positions','DELETE',NULL,NULL,NULL),(237,'p','1','/v1/sys/organization/positions','GET',NULL,NULL,NULL),(238,'p','1','/v1/sys/organization/positions','POST',NULL,NULL,NULL),(239,'p','1','/v1/sys/organization/positions','PUT',NULL,NULL,NULL),(44,'p','1','/v1/sys/randomImage','GET','','',''),(46,'p','1','/v1/sys/register','POST','','',''),(48,'p','1','/v1/sys/resetUserPassword','PUT','','',''),(246,'p','1','/v1/sys/session/admin/device','DELETE',NULL,NULL,NULL),(245,'p','1','/v1/sys/session/admin/devices','GET',NULL,NULL,NULL),(47,'p','1','/v1/sys/updateUserInfo','PUT','','','');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `schema_migrations` (
  `filename` varchar(191) COLLATE utf8mb4_general_ci NOT NULL,
  `checksum` char(64) COLLATE utf8mb4_general_ci NOT NULL,
  `applied_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`filename`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `schema_migrations` (`filename`, `checksum`, `applied_at`) VALUES ('20260925_business_access.sql','af8cb82d64db93891146c3c69dfcaf0ca0eff96f861a2b8ce74f050204639659','2026-10-07 15:43:08'),('20260925_user_active_username.sql','cf2440bfd60941cbec000dd9eef18e517d0a305ee1ea126cd349cef8ee1d4b7c','2026-10-07 15:43:09'),('20260926_00_method_dictionary.sql','399b0b3901b6f2e7edb71d042f4738379164a8686ec2242a1154a7faab671dba','2026-10-07 15:43:10'),('20260926_api_sync_access.sql','287f534292e3643e23f3c450c47aa5420e7416b4c0b5d1564ca669144481bb9b','2026-10-07 15:43:11'),('20260926_business_unique.sql','5baeac3baed30fae0a4c3ed9b81e8ca4a6ca5ecaf76caeb8017f720684a16aa6','2026-10-07 15:43:13'),('20260926_session_version.sql','9898ed43d66cf2e842e7cd70ca6ccaedcbb0e2964c07cacf5a6eb013f2d3c9ba','2026-10-07 15:43:14'),('20261002_00_policy_sync.sql','bf40d0fa1ef28e3c4e175dddbbe00ae875df7ed0e471f0dfea4fc757e4b4ae73','2026-10-07 15:43:15'),('20261002_01_audit.sql','06d7fde9ac5519750f070fa592e0d49d23d522f939096f6989cb5fb2352fcdea','2026-10-07 15:43:16'),('20261002_02_organization.sql','2a39b8e7682539b92c24b32d18eed8e98eb97bd63ca0a8029ce35832566c6f40','2026-10-07 15:43:18'),('20261002_03_files.sql','6600329d92643c909eff06848d7944b8d65b0b66a91eb77c5a648f806a957f8e','2026-10-07 15:43:19'),('20261002_04_device_sessions.sql','06dcc6a7e7f263f34b5d0373d8eb1bcde40cf91bdb237ff448ee6a752e1a39b4','2026-10-07 15:43:20'),('20261002_05_module_access.sql','cfadb622d1002666708d0cff4a35bb666f051fab81d3440de6eb5f1da11b3e0d','2026-10-07 15:43:21'),('20261003_frontend_modules.sql','e62dc29eb9d409d949c24c7387d2e10deab064ccee2f17abc3da5b48dca7c9fa','2026-10-07 15:43:23'),('20261003_frontend_routes_compat.sql','aaffb0a8d4f204b4a0f94e5c54b10c33fc2a4772d7d989eeef1cf159c4007b2e','2026-10-07 15:43:24'),('20261003_zz_ai_agent.sql','9865c0e14f334a3ae7ee0b861363cea6b0eba97ae70db72a32e33649a378ee6e','2026-10-07 15:43:25');
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_apis` (`id`, `created_at`, `updated_at`, `deleted_at`, `path`, `description`, `api_group`, `method`) VALUES (1,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/login','系统用户登录','系统-无需权限','POST'),(2,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/randomImage','获取验证码','系统-无需权限','GET'),(3,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/getUserList','分页获取用户列表','用户','GET'),(4,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/register','新增（注册）用户 - 管理员','用户','POST'),(5,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/updateUserInfo','修改用户信息','用户','PUT'),(6,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/resetUserPassword','重置用户密码 默认密码：goZero','用户','PUT'),(7,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/deleteUser','删除用户','用户','DELETE'),(8,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenu','获取菜单','菜单','GET'),(9,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenuList','分页获取base_menu列表','菜单','GET'),(10,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/addBaseMenu','新增 base_menu','菜单','POST'),(11,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getBaseMenuTree','获取用户动态路由树  -- 用于角色管理的设置权限','菜单','GET'),(12,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenuAuthority','获取指定角色menu  -- 用于角色管理的设置权限','菜单','GET'),(13,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getBaseMenuById','根据id获取菜单','菜单','GET'),(14,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/updateBaseMenu','更新系统菜单','菜单','PUT'),(15,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/deleteBaseMenu','删除系统菜单','菜单','DELETE'),(16,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/getAuthorityList','获取角色列表','角色','GET'),(17,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/addAuthorityMenu','增加角色和base_menu关联关系','角色','POST'),(18,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/updateAuthority','更新角色信息','角色','PUT'),(19,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/createAuthority','创建角色','角色','POST'),(20,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/deleteAuthority','删除角色','角色','DELETE'),(21,'2024-01-15 13:14:52.000','2024-01-15 10:35:23.438',NULL,'/v1/sys/api/getApiList','获取api列表','api','GET'),(22,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/createApi','创建/增加 api列表','api','POST'),(23,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/deleteApi','删除 api列表','api','DELETE'),(24,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/getAllApiList','获取 所有api','api','GET'),(25,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/deleteApisByIds','删除多条api','api','DELETE'),(26,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/updateApi','更新api','api','PUT'),(27,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryList','获取SysDictionary列表','字典','GET'),(28,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/createSysDictionary','新建SysDictionary','字典','POST'),(29,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryDetails','根据ID或者type获取SysDictionary','字典','GET'),(30,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/updateSysDictionary','更新SysDictionary','字典','PUT'),(31,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/deleteSysDictionary','删除SysDictionary','字典','DELETE'),(32,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoList','获取SysDictionaryInfo列表','字典','GET'),(33,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','根据id获取SysDictionaryInfo详情','字典','GET'),(34,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/updateSysDictionaryInfo','更新SysDictionaryInfo','字典','PUT'),(35,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/createSysDictionaryInfo','创建SysDictionaryInfo','字典','POST'),(36,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/deleteSysDictionaryInfo','删除SysDictionaryInfo','字典','DELETE'),(37,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/getPathByAuthorityId','根据角色id获取对应的casbin数据','casbin','GET'),(38,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/updateCasbinData','更新一个角色的对应的casbin数据','casbin','PUT'),(39,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/updateCasbinDataByApiIds','更新一个角色的对应的casbin数据 用api的ids 查数据','casbin','PUT'),(40,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/base/uploadFileImg','上传图片','base','POST'),(41,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','根据Type获取SysDictionaryInfo详情','字典','GET'),(42,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/base/sendEmailCode','邮箱发送验证码 - 暂时用于注册账号','base','POST'),(43,'2026-10-07 15:43:07.889','2026-10-07 15:43:07.889',NULL,'/v1/sys/menu/getAuthorityButtons','角色按钮权限','系统管理','GET'),(44,'2026-10-07 15:43:07.889','2026-10-07 15:43:07.889',NULL,'/v1/sys/menu/updateAuthorityButtons','保存按钮授权','系统管理','PUT'),(46,'2026-10-07 15:43:11.589','2026-10-07 15:43:11.589',NULL,'/v1/sys/api/previewSync','预览Swagger与API资源差异，失效资源仅提示','api','GET'),(47,'2026-10-07 15:43:11.590','2026-10-07 15:43:11.590',NULL,'/v1/sys/api/applySync','按预览版本同步选中API，仅新增资源或更新分组描述，保留授权','api','POST'),(48,'2026-10-07 15:43:16.664','2026-10-07 15:43:16.664',NULL,'/v1/sys/audit/getAuditLogList','分页查询操作审计与登录日志','audit','GET'),(49,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/list','按数据范围查询文件资源','files','GET'),(50,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/url','获取公开或短时签名文件地址','files','GET'),(51,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/resource','删除未被引用的资源，失败可重试','files','DELETE'),(52,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/reference','登记文件业务引用，仅内置管理员','files','POST'),(53,'2026-10-07 15:43:19.358','2026-10-07 15:43:19.358',NULL,'/v1/sys/files/reference','移除文件业务引用，仅内置管理员','files','DELETE'),(56,'2026-10-07 15:43:21.706','2026-10-07 15:43:21.706',NULL,'/v1/sys/organization/departments','查询部门树','organization','GET'),(57,'2026-10-07 15:43:21.707','2026-10-07 15:43:21.707',NULL,'/v1/sys/organization/departments','创建部门','organization','POST'),(58,'2026-10-07 15:43:21.708','2026-10-07 15:43:21.708',NULL,'/v1/sys/organization/departments','更新部门','organization','PUT'),(59,'2026-10-07 15:43:21.708','2026-10-07 15:43:21.708',NULL,'/v1/sys/organization/departments','删除部门','organization','DELETE'),(60,'2026-10-07 15:43:21.709','2026-10-07 15:43:21.709',NULL,'/v1/sys/organization/positions','查询岗位','organization','GET'),(61,'2026-10-07 15:43:21.710','2026-10-07 15:43:21.710',NULL,'/v1/sys/organization/positions','创建岗位','organization','POST'),(62,'2026-10-07 15:43:21.710','2026-10-07 15:43:21.710',NULL,'/v1/sys/organization/positions','更新岗位','organization','PUT'),(63,'2026-10-07 15:43:21.711','2026-10-07 15:43:21.711',NULL,'/v1/sys/organization/positions','删除岗位','organization','DELETE'),(64,'2026-10-07 15:43:21.712','2026-10-07 15:43:21.712',NULL,'/v1/sys/organization/membership','查询用户部门岗位','organization','GET'),(65,'2026-10-07 15:43:21.712','2026-10-07 15:43:21.712',NULL,'/v1/sys/organization/membership','更新用户部门岗位','organization','PUT'),(66,'2026-10-07 15:43:21.713','2026-10-07 15:43:21.713',NULL,'/v1/sys/organization/dataScope','查询角色数据范围','organization','GET'),(67,'2026-10-07 15:43:21.713','2026-10-07 15:43:21.713',NULL,'/v1/sys/organization/dataScope','更新角色数据范围','organization','PUT'),(68,'2026-10-07 15:43:21.716','2026-10-07 15:43:21.716',NULL,'/v1/sys/session/admin/devices','查询用户设备会话','session','GET'),(69,'2026-10-07 15:43:21.717','2026-10-07 15:43:21.717',NULL,'/v1/sys/session/admin/device','撤销用户设备会话','session','DELETE'),(70,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/info','查询AI助手配置','ai-agent','GET'),(71,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/runs','提交AI任务','ai-agent','POST'),(72,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/runs/:id','查询本人AI任务','ai-agent','GET'),(73,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/runs/:id/cancel','取消本人AI任务','ai-agent','POST'),(74,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/conversations','查询本人AI会话','ai-agent','GET'),(75,'2026-10-07 15:43:25.295','2026-10-07 15:43:25.295',NULL,'/v1/ai/conversations/:id/messages','查询本人AI会话消息','ai-agent','GET');
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_authorities` (`created_at`, `updated_at`, `deleted_at`, `authority_id`, `authority_name`, `parent_id`, `default_router`) VALUES ('2024-01-15 13:14:52.000','2024-01-16 17:49:10.662',NULL,1,'超级用户',0,'index');
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_authority_btns` (
  `authority_id` bigint unsigned DEFAULT NULL COMMENT '角色ID',
  `sys_menu_id` bigint unsigned DEFAULT NULL COMMENT '菜单ID',
  `sys_base_menu_btn_id` bigint unsigned DEFAULT NULL COMMENT '菜单按钮ID'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_authority_btns` (`authority_id`, `sys_menu_id`, `sys_base_menu_btn_id`) VALUES (1,5,23),(1,5,24),(1,5,25),(1,2,26),(1,2,27),(1,2,28),(1,3,29),(1,3,30),(1,3,31),(1,4,32),(1,4,33),(1,4,34),(1,6,35),(1,6,36),(1,6,37),(1,5,38),(1,2,39),(1,2,40),(1,2,41),(1,6,42),(1,44,43),(1,44,44),(1,4,54),(1,46,55),(1,46,56),(1,46,57),(1,49,58),(1,47,59),(1,47,60),(1,47,61),(1,48,62),(1,48,63),(1,48,64),(1,5,65),(1,2,66),(1,52,70),(1,52,71);
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_authority_menus` (
  `sys_base_menu_id` bigint unsigned NOT NULL,
  `sys_authority_authority_id` bigint unsigned NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_base_menu_id`,`sys_authority_authority_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_authority_menus` (`sys_base_menu_id`, `sys_authority_authority_id`) VALUES (1,1),(2,1),(3,1),(4,1),(5,1),(6,1),(7,1),(44,1),(45,1),(46,1),(47,1),(48,1),(49,1),(52,1);
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
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_sys_base_menu_btns_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_base_menu_btns` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `desc`, `sys_base_menu_id`) VALUES (23,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',5),(24,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',5),(25,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',5),(26,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',2),(27,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',2),(28,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',2),(29,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',3),(30,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',3),(31,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',3),(32,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',4),(33,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',4),(34,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',4),(35,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'create','新增',6),(36,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'update','编辑',6),(37,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'delete','删除',6),(38,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'resetPassword','重置密码',5),(39,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'menus','菜单授权',2),(40,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'apis','接口授权',2),(41,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'buttons','按钮授权',2),(42,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'items','字典项管理',6),(43,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'upload','上传图片',44),(44,'2026-10-07 15:43:07.891','2026-10-07 15:43:07.891',NULL,'sendEmail','发送邮箱验证码',44),(54,'2026-10-07 15:43:11.591','2026-10-07 15:43:11.591',NULL,'sync','同步接口资源',4),(55,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'upload','上传文件',46),(56,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'delete','删除文件',46),(57,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'reference','管理文件引用',46),(58,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'revoke','撤销设备会话',49),(59,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'create','新增部门',47),(60,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'update','编辑部门',47),(61,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'delete','删除部门',47),(62,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'create','新增岗位',48),(63,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'update','编辑岗位',48),(64,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'delete','删除岗位',48),(65,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'membership','分配部门岗位',5),(66,'2026-10-07 15:43:22.865','2026-10-07 15:43:22.865',NULL,'dataScope','设置数据权限',2),(70,'2026-10-07 15:43:25.300','2026-10-07 15:43:25.300',NULL,'run','提交AI任务',52),(71,'2026-10-07 15:43:25.300','2026-10-07 15:43:25.300',NULL,'cancel','取消AI任务',52);
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=REDUNDANT;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_base_menus` (`id`, `created_at`, `updated_at`, `deleted_at`, `menu_level`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`) VALUES (1,'2023-11-17 15:00:25.000','2024-01-18 09:54:30.123',NULL,0,0,'admin','superAdmin',0,'views/superAdmin/index.vue',1,'',0,0,'超级管理员','StopFilled',0),(2,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,0,1,'authority','authority',0,'views/superAdmin/authority/authority.vue',1,'',0,0,'角色管理','CheckSquareOutlined',0),(3,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,0,1,'menu','menu',0,'views/superAdmin/menu/menu.vue',2,'',1,0,'菜单管理','BugOutlined',0),(4,'2023-11-17 15:00:25.093','2024-01-10 11:41:33.339',NULL,0,1,'api','api',0,'views/superAdmin/api/api.vue',3,'',1,0,'api管理','ColumnHeightOutlined',0),(5,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,0,1,'user','user',0,'views/superAdmin/user/user.vue',4,'',0,0,'用户管理','BugOutlined',0),(6,'2023-11-17 15:00:25.093','2024-01-12 10:03:17.383',NULL,0,1,'dictionary','dictionary',0,'views/superAdmin/dictionary/dictionary.vue',5,'',0,0,'字典管理','BugOutlined',0),(7,'2024-01-16 17:27:17.123','2024-01-18 09:54:26.485',NULL,0,0,'index','index',0,'views/index.vue',0,'',0,0,'首页','AntDesignOutlined',0),(44,'2026-10-07 15:43:07.887','2026-10-07 15:43:07.887',NULL,0,1,'business-tools','businessTools',0,'views/business/tools/index.vue',90,'',0,0,'管理工具','lucide:wrench',0),(45,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,0,1,'audit','audit',0,'views/business/system/audit/index.vue',60,'',0,0,'操作审计','',0),(46,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,0,1,'files','files',0,'views/business/system/files/index.vue',61,'',0,0,'文件资源','',0),(47,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,0,1,'organization/departments','organization-departments',0,'views/business/system/organization/departments.vue',63,'',0,0,'部门管理','',0),(48,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,0,1,'organization/positions','organization-positions',0,'views/business/system/organization/positions.vue',64,'',0,0,'岗位管理','',0),(49,'2026-10-07 15:43:22.861','2026-10-07 15:43:22.861',NULL,0,1,'sessions','sessions',0,'views/business/system/sessions/index.vue',62,'',0,0,'在线会话','',0),(52,'2026-10-07 15:43:25.297','2026-10-07 15:43:25.297',NULL,0,1,'ai-agent','ai-agent',0,'views/business/ai-agent/index.vue',65,'',0,0,'AI助手','',0);
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_data_authority_id` (
  `sys_authority_authority_id` bigint unsigned NOT NULL COMMENT '角色ID',
  `data_authority_id_authority_id` bigint unsigned NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_authority_authority_id`,`data_authority_id_authority_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

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
  KEY `idx_device_expiry` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_dictionaries` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `type`, `status`, `desc`) VALUES (1,'2023-11-17 15:00:24.877','2023-11-17 15:00:24.896',NULL,'性别','gender',1,'性别字典'),(2,'2024-01-16 17:38:10.566','2024-01-16 17:38:10.566',NULL,'api请求','method',1,'api请求');
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_dictionary_info` (`id`, `created_at`, `updated_at`, `deleted_at`, `label`, `value`, `extend`, `status`, `sort`, `sys_dictionary_id`) VALUES (1,'2024-01-16 17:37:18.206','2024-01-16 17:37:18.206',NULL,'男',1,'男-拓展值',1,0,1),(2,'2024-01-16 17:37:35.413','2024-01-16 17:37:35.413',NULL,'女',2,'女-拓展值',1,2,1),(3,'2024-01-16 17:38:19.939','2024-01-18 17:26:35.798',NULL,'POST',1,'POST',1,0,2),(4,'2024-01-16 17:38:36.624','2024-01-16 17:38:36.624',NULL,'GET',2,'GET',1,0,2),(5,'2024-01-16 17:38:45.246','2024-01-16 17:38:45.246',NULL,'PUT',3,'PUT',1,0,2),(6,'2024-01-16 17:38:56.280','2024-01-16 17:38:56.280',NULL,'DELETE',4,'DELETE',1,0,2);
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

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_policy_versions` (
  `id` bigint unsigned NOT NULL,
  `version` bigint unsigned NOT NULL DEFAULT '0',
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_policy_versions` (`id`, `version`, `updated_at`) VALUES (1,1,'2026-10-07 15:43:25.297');
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

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_role_data_scopes` (
  `authority_id` bigint NOT NULL,
  `scope` varchar(32) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'self',
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`authority_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_role_scope_departments` (
  `authority_id` bigint NOT NULL,
  `department_id` bigint NOT NULL,
  PRIMARY KEY (`authority_id`,`department_id`),
  KEY `idx_scope_department` (`department_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_user_authority` (
  `sys_user_id` bigint unsigned NOT NULL,
  `sys_authority_authority_id` bigint unsigned NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_user_id`,`sys_authority_authority_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_user_authority` (`sys_user_id`, `sys_authority_authority_id`) VALUES (1,1);
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

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_user_positions` (
  `user_id` bigint NOT NULL,
  `position_id` bigint NOT NULL,
  PRIMARY KEY (`user_id`,`position_id`),
  KEY `idx_user_position` (`position_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO `sys_users` (`id`, `created_at`, `updated_at`, `deleted_at`, `uuid`, `username`, `password`, `nick_name`, `side_mode`, `header_img`, `base_color`, `active_color`, `authority_id`, `phone`, `email`, `enable`, `session_version`) VALUES (1,'2024-01-10 11:54:46.167','2024-01-10 11:54:46.167',NULL,'e2ab86b1-8e3d-4864-80ad-e83131353686','admin','$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m','管理员','dark','','#000','#fff',1,'','',1,1);
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
