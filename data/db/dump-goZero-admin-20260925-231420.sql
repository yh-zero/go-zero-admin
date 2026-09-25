-- MySQL dump 10.13  Distrib 8.0.34, for Linux (x86_64)
--
-- Host: localhost    Database: goZero-admin
-- ------------------------------------------------------
-- Server version	8.0.34

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

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `goZero-admin` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci */ /*!80016 DEFAULT ENCRYPTION='N' */;

USE `goZero-admin`;

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
) ENGINE=InnoDB AUTO_INCREMENT=130 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `casbin_rule`
--

LOCK TABLES `casbin_rule` WRITE;
/*!40000 ALTER TABLE `casbin_rule` DISABLE KEYS */;
INSERT INTO `casbin_rule` VALUES (64,'p','1','/v1/sys/api/createApi','POST','','',''),(65,'p','1','/v1/sys/api/deleteApi','DELETE','','',''),(67,'p','1','/v1/sys/api/deleteApisByIds','DELETE','','',''),(66,'p','1','/v1/sys/api/getAllApiList','GET','','',''),(63,'p','1','/v1/sys/api/getApiList','GET','','',''),(68,'p','1','/v1/sys/api/updateApi','PUT','','',''),(59,'p','1','/v1/sys/authority/addAuthorityMenu','POST','','',''),(61,'p','1','/v1/sys/authority/createAuthority','POST','','',''),(62,'p','1','/v1/sys/authority/deleteAuthority','DELETE','','',''),(58,'p','1','/v1/sys/authority/getAuthorityList','GET','','',''),(60,'p','1','/v1/sys/authority/updateAuthority','PUT','','',''),(84,'p','1','/v1/sys/base/sendEmailCode','POST','','',''),(82,'p','1','/v1/sys/base/uploadFileImg','POST','','',''),(79,'p','1','/v1/sys/casbin/getPathByAuthorityId','GET','','',''),(80,'p','1','/v1/sys/casbin/updateCasbinData','PUT','','',''),(81,'p','1','/v1/sys/casbin/updateCasbinDataByApiIds','PUT','','',''),(49,'p','1','/v1/sys/deleteUser','DELETE','','',''),(70,'p','1','/v1/sys/dictionary/createSysDictionary','POST','','',''),(77,'p','1','/v1/sys/dictionary/createSysDictionaryInfo','POST','','',''),(73,'p','1','/v1/sys/dictionary/deleteSysDictionary','DELETE','','',''),(78,'p','1','/v1/sys/dictionary/deleteSysDictionaryInfo','DELETE','','',''),(71,'p','1','/v1/sys/dictionary/getSysDictionaryDetails','GET','','',''),(74,'p','1','/v1/sys/dictionary/getSysDictionaryInfoList','GET','','',''),(75,'p','1','/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','GET','','',''),(83,'p','1','/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','GET','','',''),(69,'p','1','/v1/sys/dictionary/getSysDictionaryList','GET','','',''),(72,'p','1','/v1/sys/dictionary/updateSysDictionary','PUT','','',''),(76,'p','1','/v1/sys/dictionary/updateSysDictionaryInfo','PUT','','',''),(45,'p','1','/v1/sys/getUserList','GET','','',''),(43,'p','1','/v1/sys/login','POST','','',''),(52,'p','1','/v1/sys/menu/addBaseMenu','POST','','',''),(57,'p','1','/v1/sys/menu/deleteBaseMenu','DELETE','','',''),(127,'p','1','/v1/sys/menu/getAuthorityButtons','GET',NULL,NULL,NULL),(55,'p','1','/v1/sys/menu/getBaseMenuById','GET','','',''),(53,'p','1','/v1/sys/menu/getBaseMenuTree','GET','','',''),(50,'p','1','/v1/sys/menu/getMenu','GET','','',''),(54,'p','1','/v1/sys/menu/getMenuAuthority','GET','','',''),(51,'p','1','/v1/sys/menu/getMenuList','GET','','',''),(128,'p','1','/v1/sys/menu/updateAuthorityButtons','PUT',NULL,NULL,NULL),(56,'p','1','/v1/sys/menu/updateBaseMenu','PUT','','',''),(44,'p','1','/v1/sys/randomImage','GET','','',''),(46,'p','1','/v1/sys/register','POST','','',''),(48,'p','1','/v1/sys/resetUserPassword','PUT','','',''),(47,'p','1','/v1/sys/updateUserInfo','PUT','','',''),(22,'p','10','/v1/sys/api/createApi','POST','','',''),(23,'p','10','/v1/sys/api/deleteApi','DELETE','','',''),(25,'p','10','/v1/sys/api/deleteApisByIds','DELETE','','',''),(24,'p','10','/v1/sys/api/getAllApiList','GET','','',''),(21,'p','10','/v1/sys/api/getApiList','GET','','',''),(26,'p','10','/v1/sys/api/updateApi','PUT','','',''),(17,'p','10','/v1/sys/authority/addAuthorityMenu','POST','','',''),(19,'p','10','/v1/sys/authority/createAuthority','POST','','',''),(20,'p','10','/v1/sys/authority/deleteAuthority','DELETE','','',''),(16,'p','10','/v1/sys/authority/getAuthorityList','GET','','',''),(18,'p','10','/v1/sys/authority/updateAuthority','PUT','','',''),(42,'p','10','/v1/sys/base/sendEmailCode','POST','','',''),(40,'p','10','/v1/sys/base/uploadFileImg','POST','','',''),(37,'p','10','/v1/sys/casbin/getPathByAuthorityId','GET','','',''),(38,'p','10','/v1/sys/casbin/updateCasbinData','PUT','','',''),(39,'p','10','/v1/sys/casbin/updateCasbinDataByApiIds','PUT','','',''),(7,'p','10','/v1/sys/deleteUser','DELETE','','',''),(28,'p','10','/v1/sys/dictionary/createSysDictionary','POST','','',''),(35,'p','10','/v1/sys/dictionary/createSysDictionaryInfo','POST','','',''),(31,'p','10','/v1/sys/dictionary/deleteSysDictionary','DELETE','','',''),(36,'p','10','/v1/sys/dictionary/deleteSysDictionaryInfo','DELETE','','',''),(29,'p','10','/v1/sys/dictionary/getSysDictionaryDetails','GET','','',''),(32,'p','10','/v1/sys/dictionary/getSysDictionaryInfoList','GET','','',''),(33,'p','10','/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','GET','','',''),(41,'p','10','/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','GET','','',''),(27,'p','10','/v1/sys/dictionary/getSysDictionaryList','GET','','',''),(30,'p','10','/v1/sys/dictionary/updateSysDictionary','PUT','','',''),(34,'p','10','/v1/sys/dictionary/updateSysDictionaryInfo','PUT','','',''),(3,'p','10','/v1/sys/getUserList','GET','','',''),(1,'p','10','/v1/sys/login','POST','','',''),(10,'p','10','/v1/sys/menu/addBaseMenu','POST','','',''),(15,'p','10','/v1/sys/menu/deleteBaseMenu','DELETE','','',''),(13,'p','10','/v1/sys/menu/getBaseMenuById','GET','','',''),(11,'p','10','/v1/sys/menu/getBaseMenuTree','GET','','',''),(8,'p','10','/v1/sys/menu/getMenu','GET','','',''),(12,'p','10','/v1/sys/menu/getMenuAuthority','GET','','',''),(9,'p','10','/v1/sys/menu/getMenuList','GET','','',''),(14,'p','10','/v1/sys/menu/updateBaseMenu','PUT','','',''),(2,'p','10','/v1/sys/randomImage','GET','','',''),(4,'p','10','/v1/sys/register','POST','','',''),(6,'p','10','/v1/sys/resetUserPassword','PUT','','',''),(5,'p','10','/v1/sys/updateUserInfo','PUT','','',''),(106,'p','102','/v1/sys/api/createApi','POST','','',''),(107,'p','102','/v1/sys/api/deleteApi','DELETE','','',''),(109,'p','102','/v1/sys/api/deleteApisByIds','DELETE','','',''),(108,'p','102','/v1/sys/api/getAllApiList','GET','','',''),(105,'p','102','/v1/sys/api/getApiList','GET','','',''),(110,'p','102','/v1/sys/api/updateApi','PUT','','',''),(101,'p','102','/v1/sys/authority/addAuthorityMenu','POST','','',''),(103,'p','102','/v1/sys/authority/createAuthority','POST','','',''),(104,'p','102','/v1/sys/authority/deleteAuthority','DELETE','','',''),(100,'p','102','/v1/sys/authority/getAuthorityList','GET','','',''),(102,'p','102','/v1/sys/authority/updateAuthority','PUT','','',''),(126,'p','102','/v1/sys/base/sendEmailCode','POST','','',''),(124,'p','102','/v1/sys/base/uploadFileImg','POST','','',''),(121,'p','102','/v1/sys/casbin/getPathByAuthorityId','GET','','',''),(122,'p','102','/v1/sys/casbin/updateCasbinData','PUT','','',''),(123,'p','102','/v1/sys/casbin/updateCasbinDataByApiIds','PUT','','',''),(91,'p','102','/v1/sys/deleteUser','DELETE','','',''),(112,'p','102','/v1/sys/dictionary/createSysDictionary','POST','','',''),(119,'p','102','/v1/sys/dictionary/createSysDictionaryInfo','POST','','',''),(115,'p','102','/v1/sys/dictionary/deleteSysDictionary','DELETE','','',''),(120,'p','102','/v1/sys/dictionary/deleteSysDictionaryInfo','DELETE','','',''),(113,'p','102','/v1/sys/dictionary/getSysDictionaryDetails','GET','','',''),(116,'p','102','/v1/sys/dictionary/getSysDictionaryInfoList','GET','','',''),(117,'p','102','/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','GET','','',''),(125,'p','102','/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','GET','','',''),(111,'p','102','/v1/sys/dictionary/getSysDictionaryList','GET','','',''),(114,'p','102','/v1/sys/dictionary/updateSysDictionary','PUT','','',''),(118,'p','102','/v1/sys/dictionary/updateSysDictionaryInfo','PUT','','',''),(87,'p','102','/v1/sys/getUserList','GET','','',''),(85,'p','102','/v1/sys/login','POST','','',''),(94,'p','102','/v1/sys/menu/addBaseMenu','POST','','',''),(99,'p','102','/v1/sys/menu/deleteBaseMenu','DELETE','','',''),(97,'p','102','/v1/sys/menu/getBaseMenuById','GET','','',''),(95,'p','102','/v1/sys/menu/getBaseMenuTree','GET','','',''),(92,'p','102','/v1/sys/menu/getMenu','GET','','',''),(96,'p','102','/v1/sys/menu/getMenuAuthority','GET','','',''),(93,'p','102','/v1/sys/menu/getMenuList','GET','','',''),(98,'p','102','/v1/sys/menu/updateBaseMenu','PUT','','',''),(86,'p','102','/v1/sys/randomImage','GET','','',''),(88,'p','102','/v1/sys/register','POST','','',''),(90,'p','102','/v1/sys/resetUserPassword','PUT','','',''),(89,'p','102','/v1/sys/updateUserInfo','PUT','','','');
/*!40000 ALTER TABLE `casbin_rule` ENABLE KEYS */;
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
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_sys_apis_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=49 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_apis`
--

LOCK TABLES `sys_apis` WRITE;
/*!40000 ALTER TABLE `sys_apis` DISABLE KEYS */;
INSERT INTO `sys_apis` VALUES (1,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/login','系统用户登录','系统-无需权限','POST'),(2,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/randomImage','获取验证码','系统-无需权限','GET'),(3,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/getUserList','分页获取用户列表','用户','GET'),(4,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/register','新增（注册）用户 - 管理员','用户','POST'),(5,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/updateUserInfo','修改用户信息','用户','PUT'),(6,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/resetUserPassword','重置用户密码 默认密码：goZero','用户','PUT'),(7,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/deleteUser','删除用户','用户','DELETE'),(8,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenu','获取菜单','菜单','GET'),(9,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenuList','分页获取base_menu列表','菜单','GET'),(10,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/addBaseMenu','新增 base_menu','菜单','POST'),(11,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getBaseMenuTree','获取用户动态路由树  -- 用于角色管理的设置权限','菜单','GET'),(12,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getMenuAuthority','获取指定角色menu  -- 用于角色管理的设置权限','菜单','GET'),(13,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/getBaseMenuById','根据id获取菜单','菜单','GET'),(14,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/updateBaseMenu','更新系统菜单','菜单','PUT'),(15,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/menu/deleteBaseMenu','删除系统菜单','菜单','DELETE'),(16,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/getAuthorityList','获取角色列表','角色','GET'),(17,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/addAuthorityMenu','增加角色和base_menu关联关系','角色','POST'),(18,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/updateAuthority','更新角色信息','角色','PUT'),(19,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/createAuthority','创建角色','角色','POST'),(20,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/authority/deleteAuthority','删除角色','角色','DELETE'),(21,'2024-01-15 13:14:52.000','2024-01-15 10:35:23.438',NULL,'/v1/sys/api/getApiList','获取api列表','api','GET'),(22,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/createApi','创建/增加 api列表','api','POST'),(23,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/deleteApi','删除 api列表','api','DELETE'),(24,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/getAllApiList','获取 所有api','api','GET'),(25,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/deleteApisByIds','删除多条api','api','DELETE'),(26,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/api/updateApi','更新api','api','PUT'),(27,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryList','获取SysDictionary列表','字典','GET'),(28,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/createSysDictionary','新建SysDictionary','字典','POST'),(29,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryDetails','根据ID或者type获取SysDictionary','字典','GET'),(30,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/updateSysDictionary','更新SysDictionary','字典','PUT'),(31,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/deleteSysDictionary','删除SysDictionary','字典','DELETE'),(32,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoList','获取SysDictionaryInfo列表','字典','GET'),(33,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoListDetailsById','根据id获取SysDictionaryInfo详情','字典','GET'),(34,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/updateSysDictionaryInfo','更新SysDictionaryInfo','字典','PUT'),(35,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/createSysDictionaryInfo','创建SysDictionaryInfo','字典','POST'),(36,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/deleteSysDictionaryInfo','删除SysDictionaryInfo','字典','DELETE'),(37,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/getPathByAuthorityId','根据角色id获取对应的casbin数据','casbin','GET'),(38,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/updateCasbinData','更新一个角色的对应的casbin数据','casbin','PUT'),(39,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/casbin/updateCasbinDataByApiIds','更新一个角色的对应的casbin数据 用api的ids 查数据','casbin','PUT'),(40,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/base/uploadFileImg','上传图片','base','POST'),(41,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType','根据Type获取SysDictionaryInfo详情','字典','GET'),(42,'2024-01-15 13:14:52.000','2024-01-15 13:14:52.000',NULL,'/v1/sys/base/sendEmailCode','邮箱发送验证码 - 暂时用于注册账号','base','POST'),(46,'2026-09-25 20:09:55.901','2026-09-25 20:09:55.901',NULL,'/v1/sys/menu/getAuthorityButtons','角色按钮权限','系统管理','GET'),(47,'2026-09-25 20:09:55.901','2026-09-25 20:09:55.901',NULL,'/v1/sys/menu/updateAuthorityButtons','保存按钮授权','系统管理','PUT');
/*!40000 ALTER TABLE `sys_apis` ENABLE KEYS */;
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
) ENGINE=InnoDB AUTO_INCREMENT=103 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
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
INSERT INTO `sys_authority_btns` VALUES (1,5,23),(1,5,24),(1,5,25),(1,2,26),(1,2,27),(1,2,28),(1,3,29),(1,3,30),(1,3,31),(1,4,32),(1,4,33),(1,4,34),(1,6,35),(1,6,36),(1,6,37),(1,5,38),(1,2,39),(1,2,40),(1,2,41),(1,6,42),(1,46,43),(1,46,44);
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
INSERT INTO `sys_authority_menus` VALUES (1,1),(1,10),(1,102),(2,1),(2,10),(2,102),(3,1),(3,10),(3,102),(4,1),(4,10),(4,102),(5,1),(5,10),(5,102),(6,1),(6,10),(6,102),(7,1),(7,10),(7,102),(39,1),(39,10),(39,102),(40,1),(40,10),(41,10),(41,102),(42,10),(43,1),(43,10),(46,1);
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
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_sys_base_menu_btns_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=54 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_base_menu_btns`
--

LOCK TABLES `sys_base_menu_btns` WRITE;
/*!40000 ALTER TABLE `sys_base_menu_btns` DISABLE KEYS */;
INSERT INTO `sys_base_menu_btns` VALUES (23,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',5),(24,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',5),(25,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',5),(26,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',2),(27,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',2),(28,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',2),(29,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',3),(30,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',3),(31,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',3),(32,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',4),(33,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',4),(34,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',4),(35,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'create','新增',6),(36,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'update','编辑',6),(37,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'delete','删除',6),(38,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'resetPassword','重置密码',5),(39,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'menus','菜单授权',2),(40,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'apis','接口授权',2),(41,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'buttons','按钮授权',2),(42,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'items','字典项管理',6),(43,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'upload','上传图片',46),(44,'2026-09-25 20:09:55.909','2026-09-25 20:09:55.909',NULL,'sendEmail','发送邮箱验证码',46);
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
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_sys_base_menus_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=47 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=REDUNDANT;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_base_menus`
--

LOCK TABLES `sys_base_menus` WRITE;
/*!40000 ALTER TABLE `sys_base_menus` DISABLE KEYS */;
INSERT INTO `sys_base_menus` VALUES (1,'2023-11-17 15:00:25.000','2024-01-18 09:54:30.123',NULL,0,0,'admin','superAdmin',0,'views/superAdmin/index.vue',1,'',0,0,'超级管理员','StopFilled',0),(2,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,0,1,'authority','authority',0,'views/superAdmin/authority/authority.vue',1,'',0,0,'角色管理','CheckSquareOutlined',0),(3,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,0,1,'menu','menu',0,'views/superAdmin/menu/menu.vue',2,'',1,0,'菜单管理','BugOutlined',0),(4,'2023-11-17 15:00:25.093','2024-01-10 11:41:33.339',NULL,0,1,'api','api',0,'views/superAdmin/api/api.vue',3,'',1,0,'api管理','ColumnHeightOutlined',0),(5,'2023-11-17 15:00:25.093','2023-11-17 15:00:25.093',NULL,0,1,'user','user',0,'views/superAdmin/user/user.vue',4,'',0,0,'用户管理','BugOutlined',0),(6,'2023-11-17 15:00:25.093','2024-01-12 10:03:17.383',NULL,0,1,'dictionary','dictionary',0,'views/superAdmin/dictionary/dictionary.vue',5,'',0,0,'字典管理','BugOutlined',0),(7,'2024-01-16 17:27:17.123','2026-09-25 22:21:37.867',NULL,0,0,'index','index',0,'views/index.vue',0,'',0,0,'首页','AntDesignOutlined',0),(39,'2024-01-17 14:12:08.123','2024-01-18 10:11:35.493',NULL,0,0,'11','test1',0,'views/test/index.vue',3,'',0,0,'test1','',0),(40,'2024-01-17 14:12:21.554','2024-01-17 14:12:21.554',NULL,0,39,'test2','test2',0,'views/routerHolder.vue',0,'',0,0,'test2','',0),(41,'2024-01-17 15:38:46.677','2024-01-17 15:38:46.677',NULL,0,39,'test3','test3',0,'views/routerHolder.vue',0,'',0,0,'test3','AccountBookOutlined',0),(42,'2024-01-17 16:05:34.342','2024-01-17 16:05:34.342',NULL,0,40,'test4','test4',0,'views/routerHolder.vue',0,'',0,0,'test4','AccountBookTwoTone',0),(43,'2024-01-17 16:05:57.207','2024-01-17 16:05:57.207',NULL,0,40,'test5','test5',0,'views/routerHolder.vue',0,'',0,0,'test5','AccountBookOutlined',0),(46,'2026-09-25 20:09:55.894','2026-09-25 20:09:55.894',NULL,0,1,'business-tools','businessTools',0,'views/business/tools/index.vue',90,'',0,0,'管理工具','lucide:wrench',0);
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
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_sys_dictionaries_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=35 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dictionaries`
--

LOCK TABLES `sys_dictionaries` WRITE;
/*!40000 ALTER TABLE `sys_dictionaries` DISABLE KEYS */;
INSERT INTO `sys_dictionaries` VALUES (1,'2023-11-17 15:00:24.877','2023-11-17 15:00:24.896',NULL,'性别','gender',1,'性别字典'),(2,'2024-01-16 17:38:10.566','2024-01-16 17:38:10.566',NULL,'api请求','method',1,'api请求'),(34,'2026-09-25 20:19:36.684','2026-09-25 20:40:04.524','2026-09-25 20:40:04.524','联调测试_UI_20260925','ui_smoke_20260925',1,'');
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
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_sys_dictionary_details_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=50 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dictionary_info`
--

LOCK TABLES `sys_dictionary_info` WRITE;
/*!40000 ALTER TABLE `sys_dictionary_info` DISABLE KEYS */;
INSERT INTO `sys_dictionary_info` VALUES (1,'2024-01-16 17:37:18.206','2024-01-16 17:37:18.206',NULL,'男',1,'男-拓展值',1,0,1),(2,'2024-01-16 17:37:35.413','2024-01-16 17:37:35.413',NULL,'女',2,'女-拓展值',1,2,1),(3,'2024-01-16 17:38:19.939','2024-01-18 17:26:35.798',NULL,'POST',2,'POST',1,0,2),(4,'2024-01-16 17:38:36.624','2024-01-16 17:38:36.624',NULL,'GET',2,'GET',1,0,2),(5,'2024-01-16 17:38:45.246','2024-01-16 17:38:45.246',NULL,'PUT',3,'PUT',1,0,2),(6,'2024-01-16 17:38:56.280','2024-01-16 17:38:56.280',NULL,'DELETE',4,'DELETE',1,0,2),(49,'2026-09-25 20:24:46.665','2026-09-25 20:40:04.519','2026-09-25 20:40:04.519','零值联调',0,'',1,0,34);
/*!40000 ALTER TABLE `sys_dictionary_info` ENABLE KEYS */;
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
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_sys_users_active_username` (`active_username`),
  KEY `idx_sys_users_deleted_at` (`deleted_at`) USING BTREE,
  KEY `idx_sys_users_uuid` (`uuid`) USING BTREE,
  KEY `idx_sys_users_username` (`username`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=66 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_users`
--

LOCK TABLES `sys_users` WRITE;
/*!40000 ALTER TABLE `sys_users` DISABLE KEYS */;
INSERT INTO `sys_users` (`id`, `created_at`, `updated_at`, `deleted_at`, `uuid`, `username`, `password`, `nick_name`, `side_mode`, `header_img`, `base_color`, `active_color`, `authority_id`, `phone`, `email`, `enable`) VALUES (1,'2024-01-10 11:54:46.167','2024-01-10 11:54:46.167',NULL,'e2ab86b1-8e3d-4864-80ad-e83131353686','admin','$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m','Mr.root','dark','\r\nhttps://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#000','#fff',1,'15888888888','TypeScript@111.com',1),(2,'2024-01-10 09:27:22.905','2024-01-24 17:43:18.347',NULL,'e2ab86b1-8e3d-4864-80ad-e83131353686','yanghao','$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m','Mr.杨浩','dark','\r\nhttps://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#000','#fff',102,'15888888888','TypeScript@111.com',1),(3,'2024-01-10 09:26:44.047','2024-01-17 16:16:34.486',NULL,'e2ab86b1-8e3d-4864-80ad-e83131353686','wenlong','$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m','Mr.文龙','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705479393275_49814642.jfif','#000','#fff',1,'15888888888','TypeScript@111.com',1),(62,'2024-01-17 10:08:17.097','2024-01-17 10:08:17.097',NULL,'f3149922-22b3-4866-ad1f-e3d58ae4d9a5','用户001','$2a$10$fpVOkDRMLD3V9l34VfIN6uSvzOhgARViFsKk6pUehoYt//n..2F9C','用户001','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#fff',888,'','',1),(63,'2024-01-17 11:05:43.707','2024-01-17 11:05:43.707',NULL,'db637c17-6461-4c93-b1ea-18b33d8b0b09','11','$2a$10$xH2J8jDZC0XpD.jefci4l.hgLtN7fNZbDevh6nIwuZeQFb17kS4US','11','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',888,'','',1),(64,'2024-01-17 11:05:59.012','2024-01-17 11:05:59.012',NULL,'bfaba08a-4e9c-48b3-aaa5-fa8ebfe9d585','22','$2a$10$LnpbXmU51bWHhmRSVKusb.mXG7MHtlpYAfvNFQ2Dw.DxOSVSQManC','22','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',888,'','',1),(65,'2024-01-17 11:07:41.376','2024-01-17 11:07:41.376',NULL,'d8f57f39-e366-4895-ad8e-d2af02321d80','33','$2a$10$woWz6ZlbewPKbNPKmNI/9.KhLhjpClPtPeLaSroDWCk6UQPdju8Om','33','dark','https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg','#fff','#1890ff',888,'','',1);
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

-- Dump completed on 2026-09-25 23:14:21
