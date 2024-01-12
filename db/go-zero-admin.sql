/*
 Navicat Premium Data Transfer

 Source Server         : 本地
 Source Server Type    : MySQL
 Source Server Version : 80034
 Source Host           : localhost:3306
 Source Schema         : go-zero-admin

 Target Server Type    : MySQL
 Target Server Version : 80034
 File Encoding         : 65001

 Date: 12/01/2024 09:57:17
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for casbin_rule
-- ----------------------------
DROP TABLE IF EXISTS `casbin_rule`;
CREATE TABLE `casbin_rule`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `ptype` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,
  `v0` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,
  `v1` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,
  `v2` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,
  `v3` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,
  `v4` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,
  `v5` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_casbin_rule`(`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 65 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of casbin_rule
-- ----------------------------
INSERT INTO `casbin_rule` VALUES (8, 'p', '0', '/base/login1111', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (9, 'p', '0', '/jwt/jsonInBlacklist1222', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (3, 'p', '100001', '//v1/sys/base/uploadFileImg', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (62, 'p', '100001', '//v1/sys/dictionary/createSysDictionary', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (68, 'p', '100001', '//v1/sys/dictionary/createSysDictionaryInfo', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (65, 'p', '100001', '//v1/sys/dictionary/deleteSysDictionary', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (69, 'p', '100001', '//v1/sys/dictionary/deleteSysDictionaryInfo', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (64, 'p', '100001', '//v1/sys/dictionary/getSysDictionaryDetails', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (58, 'p', '100001', '//v1/sys/dictionary/getSysDictionaryInfoList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (56, 'p', '100001', '//v1/sys/dictionary/getSysDictionaryList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (63, 'p', '100001', '//v1/sys/dictionary/updateSysDictionary', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (38, 'p', '100001', '/v1/sys/api/createApi', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (1, 'p', '100001', '/v1/sys/api/deleteApisByIds', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (13, 'p', '100001', '/v1/sys/api/getAllApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (2, 'p', '100001', '/v1/sys/api/getApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (37, 'p', '100001', '/v1/sys/api/updateApi', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (21, 'p', '100001', '/v1/sys/authority/addAuthorityMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (22, 'p', '100001', '/v1/sys/authority/createAuthority', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (42, 'p', '100001', '/v1/sys/authority/deleteAuthority', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (4, 'p', '100001', '/v1/sys/authority/getAuthorityList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (20, 'p', '100001', '/v1/sys/base/uploadFileImg', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (5, 'p', '100001', '/v1/sys/casbin/updateCasbinData', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (14, 'p', '100001', '/v1/sys/casbin/updateCasbinDataByApiIds', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (61, 'p', '100001', '/v1/sys/dictionary/getSysDictionaryInfoList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (66, 'p', '100001', '/v1/sys/dictionary/getSysDictionaryInfoListDetailsById', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (67, 'p', '100001', '/v1/sys/dictionary/updateSysDictionaryInfo', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (7, 'p', '100001', '/v1/sys/getUserList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (40, 'p', '100001', '/v1/sys/menu/addBaseMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (41, 'p', '100001', '/v1/sys/menu/deleteBaseMenu', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (12, 'p', '100001', '/v1/sys/menu/getBaseMenuTree', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (19, 'p', '100001', '/v1/sys/menu/getMenu', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (6, 'p', '100001', '/v1/sys/menu/getMenuList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (39, 'p', '100001', '/v1/sys/updateUserInfo', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (10, 'p', '100002', '/base/login1111', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (11, 'p', '100002', '/jwt/jsonInBlacklist1222', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (15, 'p', '100003', 'v1/sys/login', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (17, 'p', '100003', 'v1/sys/menu/getMenu', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (18, 'p', '100003', 'v1/sys/menu/getMenuList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (16, 'p', '100003', 'v1/sys/randomImage', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (46, 'p', '100006', '/v1/sys/menu/getMenu', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (25, 'p', '1000093', '/user/admin_register', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (26, 'p', '1000093', '/user/changePassword', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (29, 'p', '1000093', '/user/getUserInfo', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (27, 'p', '1000093', '/user/setUserAuthority', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (28, 'p', '1000093', '/user/setUserInfo', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (23, 'p', '1000093', '/v1/sys/login', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (24, 'p', '1000093', '/v1/sys/menu/getMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (49, 'p', '11111', '/v1/sys/api/getAllApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (50, 'p', '11111', '/v1/sys/api/getApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (57, 'p', '11111', '/v1/sys/authority/addAuthorityMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (44, 'p', '11111', '/v1/sys/authority/getAuthorityList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (53, 'p', '11111', '/v1/sys/authority/updateAuthority', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (45, 'p', '11111', '/v1/sys/base/uploadFileImg', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (51, 'p', '11111', '/v1/sys/getUserList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (54, 'p', '11111', '/v1/sys/menu/addBaseMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (55, 'p', '11111', '/v1/sys/menu/deleteBaseMenu', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (48, 'p', '11111', '/v1/sys/menu/getBaseMenuTree', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (43, 'p', '11111', '/v1/sys/menu/getMenu', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (47, 'p', '11111', '/v1/sys/menu/getMenuList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (59, 'p', '11111', '/v1/sys/menu/updateBaseMenu', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (60, 'p', '11111', '/v1/sys/resetUserPassword', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (52, 'p', '11111', '/v1/sys/updateUserInfo', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (32, 'p', '123213', '/user/admin_register', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (33, 'p', '123213', '/user/changePassword', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (36, 'p', '123213', '/user/getUserInfo', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (34, 'p', '123213', '/user/setUserAuthority', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (35, 'p', '123213', '/user/setUserInfo', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (30, 'p', '123213', '/v1/sys/login', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (31, 'p', '123213', '/v1/sys/menu/getMenu', 'POST', '', '', '');

-- ----------------------------
-- Table structure for sys_apis
-- ----------------------------
DROP TABLE IF EXISTS `sys_apis`;
CREATE TABLE `sys_apis`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `path` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT 'api路径',
  `description` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT 'api中文描述',
  `api_group` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT 'api组',
  `method` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT 'POST' COMMENT '方法',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_sys_apis_deleted_at`(`deleted_at`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 747 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_apis
-- ----------------------------
INSERT INTO `sys_apis` VALUES (1, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/login', '系统用户登录', '系统用户', 'POST');
INSERT INTO `sys_apis` VALUES (2, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/randomImage', '获取验证码', '系统用户', 'GET');
INSERT INTO `sys_apis` VALUES (20, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/menu/getMenu', '获取页面菜单-路由', '菜单', 'GET');
INSERT INTO `sys_apis` VALUES (21, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/menu/getMenuList', '获取系统基础菜单', '菜单', 'GET');
INSERT INTO `sys_apis` VALUES (22, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/menu/addBaseMenu', '添加系统基础菜单', '菜单', 'POST');
INSERT INTO `sys_apis` VALUES (30, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/authority/getAuthorityList', '获取角色列表', '角色', 'GET');
INSERT INTO `sys_apis` VALUES (31, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/getApiList', '获取api列表', 'api', 'GET');
INSERT INTO `sys_apis` VALUES (32, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (704, '2023-12-26 14:58:48.822', '2023-12-26 14:58:48.822', '2024-01-09 16:37:29.741', 'test', '1', 'test', 'GET');
INSERT INTO `sys_apis` VALUES (705, '2024-01-03 16:31:43.688', '2024-01-03 16:31:43.688', NULL, 'test1', '1', 'test', 'GET');
INSERT INTO `sys_apis` VALUES (706, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (707, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (708, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (709, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (710, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (711, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (712, '0000-00-00 00:00:00.000', '2024-01-09 18:18:31.911', '2024-01-09 18:54:32.516', 'v1/sys/api/createApi123213', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (713, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', '2024-01-09 19:01:10.540', 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (714, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', '2024-01-09 19:01:13.833', 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (715, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (716, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (717, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (718, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (719, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (720, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (721, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (722, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (723, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (724, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (725, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (726, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (727, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (728, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (729, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (741, '0000-00-00 00:00:00.000', '2024-01-09 18:06:50.820', NULL, '/api/updateApi', '更新Api', 'api11', 'POST');
INSERT INTO `sys_apis` VALUES (742, '0000-00-00 00:00:00.000', '2024-01-09 18:07:09.752', NULL, '/api/updateApi1', '更新Api', 'api11', 'POST');
INSERT INTO `sys_apis` VALUES (743, '2024-01-08 16:59:13.150', '2024-01-08 16:59:13.150', NULL, 'test11111111111111', '1', 'test', 'GET');
INSERT INTO `sys_apis` VALUES (744, '2024-01-08 16:59:13.150', '2024-01-08 16:59:13.150', NULL, 'test11111111111111', '1', 'test', 'GET');
INSERT INTO `sys_apis` VALUES (745, '2024-01-09 18:14:50.409', '2024-01-09 18:14:50.409', NULL, 'test11', '1', 'test', 'GET');
INSERT INTO `sys_apis` VALUES (746, '2024-01-09 18:15:29.967', '2024-01-09 18:15:29.967', NULL, '123', '', 'asdasd', 'POST');
INSERT INTO `sys_apis` VALUES (747, '2024-01-09 18:17:28.705', '2024-01-09 18:17:28.705', NULL, '123123', 'asdasdsasdsd', 'asdasd123213', 'POST');

-- ----------------------------
-- Table structure for sys_authorities
-- ----------------------------
DROP TABLE IF EXISTS `sys_authorities`;
CREATE TABLE `sys_authorities`  (
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `authority_id` bigint UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '角色ID',
  `authority_name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '角色名',
  `parent_id` bigint UNSIGNED NULL DEFAULT NULL COMMENT '父角色ID',
  `default_router` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT 'dashboard' COMMENT '默认菜单',
  PRIMARY KEY (`authority_id`) USING BTREE,
  UNIQUE INDEX `authority_id`(`authority_id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 12321321321 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_authorities
-- ----------------------------
INSERT INTO `sys_authorities` VALUES ('2024-01-04 10:05:36.725', '2024-01-10 11:41:08.385', NULL, 11111, 'yanghao', 0, 'authority');
INSERT INTO `sys_authorities` VALUES (NULL, '2024-01-03 20:45:22.096', NULL, 100001, '超级用户', 0, 'superAdmin');
INSERT INTO `sys_authorities` VALUES (NULL, NULL, NULL, 100002, 'admin-test2', 100001, 'superAdmin');
INSERT INTO `sys_authorities` VALUES (NULL, NULL, NULL, 100003, 'admin-test3', 100001, 'superAdmin');
INSERT INTO `sys_authorities` VALUES ('2024-01-08 15:18:37.000', NULL, NULL, 100004, 'admin-test4', 0, 'superAdmin');
INSERT INTO `sys_authorities` VALUES (NULL, NULL, NULL, 100005, 'admin-test5', 100002, 'superAdmin');
INSERT INTO `sys_authorities` VALUES ('2024-01-08 15:20:05.851', '2024-01-08 15:20:05.852', NULL, 100006, 'test201', 100002, 'superAdmin');
INSERT INTO `sys_authorities` VALUES ('2024-01-09 17:38:31.632', '2024-01-09 17:38:31.633', NULL, 123213, 't092', 0, 'superAdmin');
INSERT INTO `sys_authorities` VALUES ('2024-01-09 16:57:37.949', '2024-01-09 16:57:37.957', NULL, 1000093, '', 0, 'superAdmin');

-- ----------------------------
-- Table structure for sys_authority_btns
-- ----------------------------
DROP TABLE IF EXISTS `sys_authority_btns`;
CREATE TABLE `sys_authority_btns`  (
  `authority_id` bigint UNSIGNED NULL DEFAULT NULL COMMENT '角色ID',
  `sys_menu_id` bigint UNSIGNED NULL DEFAULT NULL COMMENT '菜单ID',
  `sys_base_menu_btn_id` bigint UNSIGNED NULL DEFAULT NULL COMMENT '菜单按钮ID'
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_authority_btns
-- ----------------------------
INSERT INTO `sys_authority_btns` VALUES (100001, 1, 1);

-- ----------------------------
-- Table structure for sys_authority_menus
-- ----------------------------
DROP TABLE IF EXISTS `sys_authority_menus`;
CREATE TABLE `sys_authority_menus`  (
  `sys_base_menu_id` bigint UNSIGNED NOT NULL,
  `sys_authority_authority_id` bigint UNSIGNED NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_base_menu_id`, `sys_authority_authority_id`) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_authority_menus
-- ----------------------------
INSERT INTO `sys_authority_menus` VALUES (1, 11111);
INSERT INTO `sys_authority_menus` VALUES (1, 100001);
INSERT INTO `sys_authority_menus` VALUES (1, 100006);
INSERT INTO `sys_authority_menus` VALUES (1, 100007);
INSERT INTO `sys_authority_menus` VALUES (1, 100008);
INSERT INTO `sys_authority_menus` VALUES (1, 100009);
INSERT INTO `sys_authority_menus` VALUES (1, 123213);
INSERT INTO `sys_authority_menus` VALUES (1, 1000091);
INSERT INTO `sys_authority_menus` VALUES (1, 1000092);
INSERT INTO `sys_authority_menus` VALUES (1, 1000093);
INSERT INTO `sys_authority_menus` VALUES (1, 1000095);
INSERT INTO `sys_authority_menus` VALUES (1, 12321321321);
INSERT INTO `sys_authority_menus` VALUES (2, 11111);
INSERT INTO `sys_authority_menus` VALUES (2, 100001);
INSERT INTO `sys_authority_menus` VALUES (3, 11111);
INSERT INTO `sys_authority_menus` VALUES (3, 100001);
INSERT INTO `sys_authority_menus` VALUES (4, 11111);
INSERT INTO `sys_authority_menus` VALUES (4, 100001);
INSERT INTO `sys_authority_menus` VALUES (5, 11111);
INSERT INTO `sys_authority_menus` VALUES (5, 100001);
INSERT INTO `sys_authority_menus` VALUES (6, 11111);
INSERT INTO `sys_authority_menus` VALUES (6, 100001);
INSERT INTO `sys_authority_menus` VALUES (6, 100002);
INSERT INTO `sys_authority_menus` VALUES (7, 100001);
INSERT INTO `sys_authority_menus` VALUES (7, 100002);
INSERT INTO `sys_authority_menus` VALUES (25, 11111);
INSERT INTO `sys_authority_menus` VALUES (25, 100001);
INSERT INTO `sys_authority_menus` VALUES (25, 100002);
INSERT INTO `sys_authority_menus` VALUES (28, 11111);
INSERT INTO `sys_authority_menus` VALUES (28, 100001);
INSERT INTO `sys_authority_menus` VALUES (35, 11111);
INSERT INTO `sys_authority_menus` VALUES (36, 11111);

-- ----------------------------
-- Table structure for sys_base_menu_btns
-- ----------------------------
DROP TABLE IF EXISTS `sys_base_menu_btns`;
CREATE TABLE `sys_base_menu_btns`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '按钮关键key',
  `desc` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL,
  `sys_base_menu_id` bigint UNSIGNED NULL DEFAULT NULL COMMENT '菜单ID',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_sys_base_menu_btns_deleted_at`(`deleted_at`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 22 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_base_menu_btns
-- ----------------------------
INSERT INTO `sys_base_menu_btns` VALUES (1, '2023-12-11 18:19:50.230', '2023-12-11 18:19:50.230', NULL, '按钮test', '按钮test-备注', 2);
INSERT INTO `sys_base_menu_btns` VALUES (3, '2023-12-21 10:04:29.287', '2023-12-21 10:04:29.287', NULL, '按钮名称', '按钮备注', 15);
INSERT INTO `sys_base_menu_btns` VALUES (4, '2023-12-21 10:04:40.991', '2023-12-21 10:04:40.991', NULL, '按钮名称', '按钮备注', 16);
INSERT INTO `sys_base_menu_btns` VALUES (5, '2023-12-21 10:05:13.372', '2023-12-21 10:05:13.372', NULL, '按钮名称', '按钮备注', 17);
INSERT INTO `sys_base_menu_btns` VALUES (6, '2023-12-21 10:17:25.522', '2023-12-21 10:17:25.522', NULL, '按钮名称', '按钮备注', 18);
INSERT INTO `sys_base_menu_btns` VALUES (7, '2023-12-21 10:18:06.395', '2023-12-21 10:18:06.395', NULL, '按钮名称', '按钮备注', 19);
INSERT INTO `sys_base_menu_btns` VALUES (8, '2023-12-21 10:30:41.121', '2023-12-21 10:30:41.121', NULL, '按钮名称', '按钮备注', 20);
INSERT INTO `sys_base_menu_btns` VALUES (9, '2023-12-21 10:32:16.662', '2023-12-21 10:32:16.662', NULL, '按钮名称', '按钮备注', 21);
INSERT INTO `sys_base_menu_btns` VALUES (10, '2023-12-21 10:33:21.793', '2023-12-21 10:33:21.793', NULL, '按钮名称', '按钮备注', 22);
INSERT INTO `sys_base_menu_btns` VALUES (11, '2023-12-21 10:36:48.554', '2023-12-21 10:36:48.554', NULL, '按钮名称', '按钮备注', 23);
INSERT INTO `sys_base_menu_btns` VALUES (12, '2023-12-21 10:49:03.386', '2023-12-21 10:49:03.386', NULL, '按钮名称', '按钮备注', 24);
INSERT INTO `sys_base_menu_btns` VALUES (13, '2023-12-25 06:58:23.577', '2023-12-25 06:58:23.577', NULL, '按钮名称', '按钮备注', 25);
INSERT INTO `sys_base_menu_btns` VALUES (14, '2023-12-27 15:59:04.476', '2023-12-27 15:59:04.476', NULL, '222', '222', 26);
INSERT INTO `sys_base_menu_btns` VALUES (15, '2023-12-27 15:59:56.160', '2023-12-27 15:59:56.160', NULL, '按钮名称', '按钮备注', 27);
INSERT INTO `sys_base_menu_btns` VALUES (16, '2023-12-27 16:02:29.820', '2023-12-27 16:02:29.820', '2024-01-09 18:43:33.038', '22', '22', 28);
INSERT INTO `sys_base_menu_btns` VALUES (18, '2024-01-10 10:42:31.636', '2024-01-10 10:42:31.636', '0000-00-00 00:00:00.000', '222', '222', 30);
INSERT INTO `sys_base_menu_btns` VALUES (19, '2024-01-10 10:47:19.555', '2024-01-10 10:47:19.555', '0000-00-00 00:00:00.000', '222', '222', 31);
INSERT INTO `sys_base_menu_btns` VALUES (20, '2024-01-10 10:47:31.086', '2024-01-10 10:47:31.086', '0000-00-00 00:00:00.000', '222', '222', 32);
INSERT INTO `sys_base_menu_btns` VALUES (21, '2024-01-10 10:48:34.601', '2024-01-10 10:48:34.601', '0000-00-00 00:00:00.000', '222', '222', 33);
INSERT INTO `sys_base_menu_btns` VALUES (22, '2024-01-10 10:50:38.336', '2024-01-10 10:50:38.336', '0000-00-00 00:00:00.000', '222', '222', 34);

-- ----------------------------
-- Table structure for sys_base_menu_parameters
-- ----------------------------
DROP TABLE IF EXISTS `sys_base_menu_parameters`;
CREATE TABLE `sys_base_menu_parameters`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `sys_base_menu_id` bigint UNSIGNED NULL DEFAULT NULL,
  `type` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '地址栏携带参数为params还是query',
  `key` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '地址栏携带参数的key',
  `value` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '地址栏携带参数的值',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_sys_base_menu_parameters_deleted_at`(`deleted_at`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 20 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_base_menu_parameters
-- ----------------------------
INSERT INTO `sys_base_menu_parameters` VALUES (1, '2023-12-21 10:04:29.274', '2023-12-21 10:04:29.274', NULL, 15, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (2, '2023-12-21 10:04:40.990', '2023-12-21 10:04:40.990', NULL, 16, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (3, '2023-12-21 10:05:13.371', '2023-12-21 10:05:13.371', NULL, 17, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (4, '2023-12-21 10:17:25.521', '2023-12-21 10:17:25.521', NULL, 18, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (5, '2023-12-21 10:18:06.394', '2023-12-21 10:18:06.394', NULL, 19, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (6, '2023-12-21 10:30:41.120', '2023-12-21 10:30:41.120', NULL, 20, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (7, '2023-12-21 10:32:16.661', '2023-12-21 10:32:16.661', NULL, 21, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (8, '2023-12-21 10:33:21.791', '2023-12-21 10:33:21.791', NULL, 22, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (9, '2023-12-21 10:36:48.554', '2023-12-21 10:36:48.554', NULL, 23, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (10, '2023-12-21 10:49:03.385', '2023-12-21 10:49:03.385', NULL, 24, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (11, '2023-12-25 06:58:23.575', '2023-12-25 06:58:23.575', NULL, 25, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (12, '2023-12-27 15:59:04.475', '2023-12-27 15:59:04.475', NULL, 26, 'query', '111', '111');
INSERT INTO `sys_base_menu_parameters` VALUES (13, '2023-12-27 15:59:56.160', '2023-12-27 15:59:56.160', NULL, 27, 'query', '参数key', '参数值');
INSERT INTO `sys_base_menu_parameters` VALUES (14, '2023-12-27 16:02:29.819', '2023-12-27 16:02:29.819', '2024-01-09 18:43:33.016', 28, 'query', '11', '11');
INSERT INTO `sys_base_menu_parameters` VALUES (16, '2024-01-10 10:42:31.635', '2024-01-10 10:42:31.635', '0000-00-00 00:00:00.000', 30, 'query', '111', '111');
INSERT INTO `sys_base_menu_parameters` VALUES (17, '2024-01-10 10:47:19.554', '2024-01-10 10:47:19.554', '0000-00-00 00:00:00.000', 31, 'query', '111', '111');
INSERT INTO `sys_base_menu_parameters` VALUES (18, '2024-01-10 10:47:31.085', '2024-01-10 10:47:31.085', '0000-00-00 00:00:00.000', 32, 'query', '111', '111');
INSERT INTO `sys_base_menu_parameters` VALUES (19, '2024-01-10 10:48:34.601', '2024-01-10 10:48:34.601', '0000-00-00 00:00:00.000', 33, 'query', '111', '111');
INSERT INTO `sys_base_menu_parameters` VALUES (20, '2024-01-10 10:50:38.335', '2024-01-10 10:50:38.335', '0000-00-00 00:00:00.000', 34, 'query', '111', '111');

-- ----------------------------
-- Table structure for sys_base_menus
-- ----------------------------
DROP TABLE IF EXISTS `sys_base_menus`;
CREATE TABLE `sys_base_menus`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `menu_level` bigint UNSIGNED NULL DEFAULT NULL,
  `parent_id` bigint NULL DEFAULT NULL COMMENT '父菜单ID',
  `path` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '路由path',
  `name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '路由name',
  `hidden` tinyint(1) NULL DEFAULT NULL COMMENT '是否在列表隐藏',
  `component` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '对应前端文件路径',
  `sort` bigint NULL DEFAULT NULL COMMENT '排序标记',
  `active_name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '附加属性',
  `keep_alive` tinyint(1) NULL DEFAULT NULL COMMENT '附加属性',
  `default_menu` tinyint(1) NULL DEFAULT NULL COMMENT '附加属性',
  `title` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '附加属性',
  `icon` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '附加属性',
  `close_tab` tinyint(1) NULL DEFAULT NULL COMMENT '附加属性',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_sys_base_menus_deleted_at`(`deleted_at`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 37 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = REDUNDANT;

-- ----------------------------
-- Records of sys_base_menus
-- ----------------------------
INSERT INTO `sys_base_menus` VALUES (1, '2023-11-17 15:00:25.000', '2024-01-10 11:40:42.004', NULL, 0, 0, 'admin', 'superAdmin', 0, 'views/superAdmin/index.vue', 0, '', 0, 0, '超级管理员', 'StopFilled', 0);
INSERT INTO `sys_base_menus` VALUES (2, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, 1, 'authority', 'authority', 0, 'views/superAdmin/authority/authority.vue', 1, '', 0, 0, '角色管理', 'CheckSquareOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (3, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, 1, 'menu', 'menu', 0, 'views/superAdmin/menu/menu.vue', 2, '', 1, 0, '菜单管理', 'BugOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (4, '2023-11-17 15:00:25.093', '2024-01-10 11:41:33.339', NULL, 0, 1, 'api', 'api', 0, 'views/superAdmin/api/api.vue', 3, '', 1, 0, 'api管理', 'ColumnHeightOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (5, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, 1, 'user', 'user', 0, 'views/superAdmin/user/user.vue', 4, '', 0, 0, '用户管理', 'BugOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (6, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, 1, 'dictionary', 'dictionary', 0, 'views/superAdmin/dictionary/sysDictionary.vue', 5, '', 0, 0, '字典管理', 'BugOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (7, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', '2024-01-10 09:39:01.000', 0, 1, 'operation', 'operation', 0, 'views/superAdmin/operation/sysOperationRecord.vue', 6, '', 0, 0, '操作历史', 'BugOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (25, '2023-12-25 06:58:23.563', '2023-12-25 06:58:23.563', NULL, 0, 0, 'route-name', 'route-name6', 0, 'views/routerHolder.vue', 1, '', 0, 0, '展示名称', 'BugOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (28, '2023-12-27 16:02:29.818', '2023-12-27 16:02:29.818', NULL, 0, 0, 'test', 'test', 0, 'views/test/index.vue', 12, '123', 0, 0, 'TEST', 'BugOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (34, '2024-01-10 10:50:38.334', '2024-01-10 10:50:38.334', '2024-01-10 10:59:13.521', 0, 0, 'test11', 'test11', 0, 'view/test/index.vue', 0, '', 0, 1, 'test11', 'jack', 0);
INSERT INTO `sys_base_menus` VALUES (35, '2024-01-10 10:58:39.496', '2024-01-10 10:58:39.496', NULL, 0, 0, '123123', '23123', 0, 'views/routerHolder.vue', 0, '', 0, 0, '展示名称', '', 0);
INSERT INTO `sys_base_menus` VALUES (36, '2024-01-10 10:59:35.131', '2024-01-10 10:59:35.131', NULL, 0, 28, '路由Path', '路由Name', 0, 'views/routerHolder.vue', 0, '', 0, 0, '展示名称', '', 0);
INSERT INTO `sys_base_menus` VALUES (37, '2024-01-10 11:01:00.310', '2024-01-10 11:01:00.310', '2024-01-10 11:23:42.708', 0, 36, 'aaaa', 'aaaaa', 0, 'views/routerHolder.vue', 0, '', 0, 0, '123', '', 0);

-- ----------------------------
-- Table structure for sys_data_authority_id
-- ----------------------------
DROP TABLE IF EXISTS `sys_data_authority_id`;
CREATE TABLE `sys_data_authority_id`  (
  `sys_authority_authority_id` bigint UNSIGNED NOT NULL COMMENT '角色ID',
  `data_authority_id_authority_id` bigint UNSIGNED NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_authority_authority_id`, `data_authority_id_authority_id`) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of sys_data_authority_id
-- ----------------------------
INSERT INTO `sys_data_authority_id` VALUES (888, 888);
INSERT INTO `sys_data_authority_id` VALUES (888, 8881);
INSERT INTO `sys_data_authority_id` VALUES (888, 9528);
INSERT INTO `sys_data_authority_id` VALUES (9528, 8881);
INSERT INTO `sys_data_authority_id` VALUES (9528, 9528);

-- ----------------------------
-- Table structure for sys_dictionaries
-- ----------------------------
DROP TABLE IF EXISTS `sys_dictionaries`;
CREATE TABLE `sys_dictionaries`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '字典名（中）',
  `type` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '字典名（英）',
  `status` tinyint(1) NULL DEFAULT NULL COMMENT '状态',
  `desc` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '描述',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_sys_dictionaries_deleted_at`(`deleted_at`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 13 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of sys_dictionaries
-- ----------------------------
INSERT INTO `sys_dictionaries` VALUES (1, '2023-11-17 15:00:24.877', '2023-11-17 15:00:24.896', NULL, '性别', 'gender', 1, '性别字典');
INSERT INTO `sys_dictionaries` VALUES (2, '2023-11-17 15:00:24.877', '2023-11-17 15:00:24.917', NULL, '数据库int类型', 'int', 1, 'int类型对应的数据库类型');
INSERT INTO `sys_dictionaries` VALUES (3, '2023-11-17 15:00:24.877', '2023-11-17 15:00:24.938', NULL, '数据库时间日期类型', 'time.Time', 1, '数据库时间日期类型');
INSERT INTO `sys_dictionaries` VALUES (4, '2023-11-17 15:00:24.877', '2023-11-17 15:00:24.958', NULL, '数据库浮点型', 'float64', 1, '数据库浮点型');
INSERT INTO `sys_dictionaries` VALUES (5, '2023-11-17 15:00:24.877', '2023-11-17 15:00:24.980', NULL, '数据库字符串', 'string', 1, '数据库字符串');
INSERT INTO `sys_dictionaries` VALUES (6, '2023-11-17 15:00:24.877', '2023-11-17 15:00:25.066', '2024-01-11 15:44:14.999', '数据库bool类型', 'bool', 1, '数据库bool类型');
INSERT INTO `sys_dictionaries` VALUES (10, '2024-01-10 14:29:37.876', '2024-01-10 14:29:37.876', NULL, '字典名（中）', 'zidia111n', 1, '字典-描述');
INSERT INTO `sys_dictionaries` VALUES (11, '2024-01-10 14:30:03.489', '2024-01-10 14:30:03.489', NULL, '字典名（中）', 'zidia1111n', 1, '字典-描述');
INSERT INTO `sys_dictionaries` VALUES (12, '2024-01-10 14:30:05.926', '2024-01-10 14:41:13.848', '2024-01-11 15:44:05.765', '数据库浮点型1', 'float641', 1, '数据库浮点型');
INSERT INTO `sys_dictionaries` VALUES (13, '2024-01-11 15:13:39.895', '2024-01-11 15:13:39.895', NULL, '字典名（中）', 'zidian', 1, '字典-描述');
INSERT INTO `sys_dictionaries` VALUES (14, '2024-01-11 15:15:14.230', '2024-01-11 15:15:14.230', NULL, '字典名（中）', 'zidia1n', 1, '字典-描述');

-- ----------------------------
-- Table structure for sys_dictionary_info
-- ----------------------------
DROP TABLE IF EXISTS `sys_dictionary_info`;
CREATE TABLE `sys_dictionary_info`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `label` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '展示值',
  `value` bigint NULL DEFAULT NULL COMMENT '字典值',
  `extend` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '扩展值',
  `status` tinyint(1) NULL DEFAULT NULL COMMENT '启用状态',
  `sort` bigint NULL DEFAULT NULL COMMENT '排序标记',
  `sys_dictionary_id` bigint UNSIGNED NULL DEFAULT NULL COMMENT '关联标记',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_sys_dictionary_details_deleted_at`(`deleted_at`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 26 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of sys_dictionary_info
-- ----------------------------
INSERT INTO `sys_dictionary_info` VALUES (1, '2023-11-17 15:00:24.897', '2023-11-17 15:00:24.897', NULL, '男', 1, '', 1, 1, 1);
INSERT INTO `sys_dictionary_info` VALUES (2, '2023-11-17 15:00:24.897', '2023-11-17 15:00:24.897', NULL, '女', 2, '', 1, 2, 1);
INSERT INTO `sys_dictionary_info` VALUES (3, '2023-11-17 15:00:24.917', '2023-11-17 15:00:24.917', NULL, 'smallint', 1, '', 1, 1, 2);
INSERT INTO `sys_dictionary_info` VALUES (4, '2023-11-17 15:00:24.917', '2023-11-17 15:00:24.917', NULL, 'mediumint', 2, '', 1, 2, 2);
INSERT INTO `sys_dictionary_info` VALUES (5, '2023-11-17 15:00:24.917', '2023-11-17 15:00:24.917', NULL, 'int', 3, '', 1, 3, 2);
INSERT INTO `sys_dictionary_info` VALUES (6, '2023-11-17 15:00:24.917', '2023-11-17 15:00:24.917', NULL, 'bigint', 4, '', 1, 4, 2);
INSERT INTO `sys_dictionary_info` VALUES (7, '2023-11-17 15:00:24.939', '2023-11-17 15:00:24.939', NULL, 'date', 0, '', 1, 0, 3);
INSERT INTO `sys_dictionary_info` VALUES (8, '2023-11-17 15:00:24.939', '2023-11-17 15:00:24.939', NULL, 'time', 1, '', 1, 1, 3);
INSERT INTO `sys_dictionary_info` VALUES (9, '2023-11-17 15:00:24.939', '2023-11-17 15:00:24.939', NULL, 'year', 2, '', 1, 2, 3);
INSERT INTO `sys_dictionary_info` VALUES (10, '2023-11-17 15:00:24.939', '2023-11-17 15:00:24.939', NULL, 'datetime', 3, '', 1, 3, 3);
INSERT INTO `sys_dictionary_info` VALUES (11, '2023-11-17 15:00:24.939', '2023-11-17 15:00:24.939', NULL, 'timestamp', 5, '', 1, 5, 3);
INSERT INTO `sys_dictionary_info` VALUES (12, '2023-11-17 15:00:24.960', '2023-11-17 15:00:24.960', NULL, 'float', 0, '', 1, 0, 4);
INSERT INTO `sys_dictionary_info` VALUES (13, '2023-11-17 15:00:24.960', '2023-11-17 15:00:24.960', NULL, 'double', 1, '', 1, 1, 4);
INSERT INTO `sys_dictionary_info` VALUES (14, '2023-11-17 15:00:24.960', '2023-11-17 15:00:24.960', NULL, 'decimal', 2, '', 1, 2, 4);
INSERT INTO `sys_dictionary_info` VALUES (15, '2023-11-17 15:00:24.981', '2023-11-17 15:00:24.981', NULL, 'char', 0, '', 1, 0, 5);
INSERT INTO `sys_dictionary_info` VALUES (16, '2023-11-17 15:00:24.981', '2023-11-17 15:00:24.981', NULL, 'varchar', 1, '', 1, 1, 5);
INSERT INTO `sys_dictionary_info` VALUES (17, '2023-11-17 15:00:24.981', '2023-11-17 15:00:24.981', NULL, 'tinyblob', 2, '', 1, 2, 5);
INSERT INTO `sys_dictionary_info` VALUES (18, '2023-11-17 15:00:24.981', '2023-11-17 15:00:24.981', NULL, 'tinytext', 3, '', 1, 3, 5);
INSERT INTO `sys_dictionary_info` VALUES (19, '2023-11-17 15:00:24.981', '2023-11-17 15:00:24.981', NULL, 'text', 4, '', 1, 4, 5);
INSERT INTO `sys_dictionary_info` VALUES (20, '2023-11-17 15:00:24.981', '2023-11-17 15:00:24.981', NULL, 'blob', 5, '', 1, 5, 5);
INSERT INTO `sys_dictionary_info` VALUES (21, '2023-11-17 15:00:24.981', '2023-11-17 15:00:24.981', NULL, 'mediumblob', 6, '', 1, 6, 5);
INSERT INTO `sys_dictionary_info` VALUES (22, '2023-11-17 15:00:24.981', '2023-11-17 15:00:24.981', NULL, 'mediumtext', 7, '', 1, 7, 5);
INSERT INTO `sys_dictionary_info` VALUES (23, '2023-11-17 15:00:24.981', '2023-11-17 15:00:24.981', NULL, 'longblob', 8, '', 1, 8, 5);
INSERT INTO `sys_dictionary_info` VALUES (24, '0000-00-00 00:00:00.000', '2024-01-11 16:57:24.755', NULL, '', 0, '', 0, 0, 0);
INSERT INTO `sys_dictionary_info` VALUES (25, '2023-11-17 15:00:25.067', '2024-01-11 16:56:24.228', NULL, '', 0, '', 0, 0, 0);
INSERT INTO `sys_dictionary_info` VALUES (29, '0000-00-00 00:00:00.000', '2024-01-11 17:39:10.413', NULL, '展示值', 2, '扩展值', 1, 1, 3);
INSERT INTO `sys_dictionary_info` VALUES (30, '2024-01-12 09:41:46.260', '2024-01-12 09:41:46.260', '0000-00-00 00:00:00.000', '展示值', 2, '扩展值', 1, 1, 1);
INSERT INTO `sys_dictionary_info` VALUES (31, '2024-01-12 09:42:25.022', '2024-01-12 09:42:25.022', NULL, '展示值', 2, '扩展值', 1, 1, 1);
INSERT INTO `sys_dictionary_info` VALUES (32, '2024-01-12 09:42:44.385', '2024-01-12 09:42:44.385', NULL, '展示值', 2, '', 1, 1, 1);
INSERT INTO `sys_dictionary_info` VALUES (33, '2024-01-12 09:43:19.700', '2024-01-12 09:43:19.700', '2024-01-12 09:54:18.779', '展示值', 2, '扩展值', 1, 1, 1);
INSERT INTO `sys_dictionary_info` VALUES (34, '2024-01-12 09:52:05.574', '2024-01-12 09:52:05.574', '2024-01-12 09:54:06.019', '展示值', 2, '扩展值', 1, 1, 1);

-- ----------------------------
-- Table structure for sys_user_authority
-- ----------------------------
DROP TABLE IF EXISTS `sys_user_authority`;
CREATE TABLE `sys_user_authority`  (
  `sys_user_id` bigint UNSIGNED NOT NULL,
  `sys_authority_authority_id` bigint UNSIGNED NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_user_id`, `sys_authority_authority_id`) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_user_authority
-- ----------------------------
INSERT INTO `sys_user_authority` VALUES (2, 11111);
INSERT INTO `sys_user_authority` VALUES (2, 100003);
INSERT INTO `sys_user_authority` VALUES (2, 100005);
INSERT INTO `sys_user_authority` VALUES (2, 100006);
INSERT INTO `sys_user_authority` VALUES (3, 11111);
INSERT INTO `sys_user_authority` VALUES (4, 11111);
INSERT INTO `sys_user_authority` VALUES (4, 100006);
INSERT INTO `sys_user_authority` VALUES (6, 1);
INSERT INTO `sys_user_authority` VALUES (6, 888);
INSERT INTO `sys_user_authority` VALUES (58, 1);
INSERT INTO `sys_user_authority` VALUES (58, 2);
INSERT INTO `sys_user_authority` VALUES (59, 1);
INSERT INTO `sys_user_authority` VALUES (59, 2);
INSERT INTO `sys_user_authority` VALUES (61, 11111);
INSERT INTO `sys_user_authority` VALUES (61, 100003);
INSERT INTO `sys_user_authority` VALUES (61, 100006);

-- ----------------------------
-- Table structure for sys_users
-- ----------------------------
DROP TABLE IF EXISTS `sys_users`;
CREATE TABLE `sys_users`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `uuid` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '用户UUID',
  `username` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '用户登录名',
  `password` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '用户登录密码',
  `nick_name` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT '系统用户' COMMENT '用户昵称',
  `side_mode` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT 'dark' COMMENT '用户侧边主题',
  `header_img` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT '' COMMENT '用户头像',
  `base_color` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT '#fff' COMMENT '基础颜色',
  `active_color` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT '#1890ff' COMMENT '活跃颜色',
  `authority_id` bigint UNSIGNED NULL DEFAULT 1000001 COMMENT '用户角色ID',
  `phone` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '用户手机号',
  `email` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '用户邮箱',
  `enable` bigint NULL DEFAULT 1 COMMENT '用户是否被冻结 1正常 2冻结',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_sys_users_deleted_at`(`deleted_at`) USING BTREE,
  INDEX `idx_sys_users_uuid`(`uuid`) USING BTREE,
  INDEX `idx_sys_users_username`(`username`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 61 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_users
-- ----------------------------
INSERT INTO `sys_users` VALUES (2, NULL, '2024-01-10 11:54:46.167', NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'admin', '$2a$10$V9AWTXw5miamRJTQXZuipe.zhPBW1LLv5NzlAg23rbqS7dersbZb6', 'Mr.root', 'dark', '', '#000', '#fff', 11111, '15888888888', 'TypeScript@111.com', 1);
INSERT INTO `sys_users` VALUES (3, NULL, '2024-01-10 09:27:22.905', NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'yanghao', '$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m', 'Mr.杨浩', 'dark', '', '#000', '#fff', 11111, '15888888888', 'TypeScript@111.com', 1);
INSERT INTO `sys_users` VALUES (4, NULL, '2024-01-10 09:26:44.047', NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'wenlong', '$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m', 'Mr.文龙', 'dark', '', '#000', '#1890ff', 100006, '15888888888', 'TypeScript@111.com', 1);
INSERT INTO `sys_users` VALUES (61, NULL, '2024-01-09 18:20:32.248', NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'yanghao-test', '$2a$10$/ulL9ffdRsLsxIu/eBPq8uuBeYgOcFPDmRFuv288j5GyiRrLFZiQ2', 'Mr.杨浩-test', 'dark', '', '#000', '#1890ff', 11111, '15888888888', 'TypeScript@111.com', 1);

SET FOREIGN_KEY_CHECKS = 1;
