/*
 Navicat Premium Data Transfer

 Source Server         : 本地
 Source Server Type    : MySQL
 Source Server Version : 80034
 Source Host           : localhost:3306
 Source Schema         : gozero-admin

 Target Server Type    : MySQL
 Target Server Version : 80034
 File Encoding         : 65001

 Date: 18/01/2024 17:51:40
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
) ENGINE = InnoDB AUTO_INCREMENT = 84 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of casbin_rule
-- ----------------------------
INSERT INTO `casbin_rule` VALUES (22, 'p', '1', '/v1/sys/api/createApi', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (23, 'p', '1', '/v1/sys/api/deleteApi', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (25, 'p', '1', '/v1/sys/api/deleteApisByIds', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (24, 'p', '1', '/v1/sys/api/getAllApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (21, 'p', '1', '/v1/sys/api/getApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (26, 'p', '1', '/v1/sys/api/updateApi', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (17, 'p', '1', '/v1/sys/authority/addAuthorityMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (19, 'p', '1', '/v1/sys/authority/createAuthority', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (20, 'p', '1', '/v1/sys/authority/deleteAuthority', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (16, 'p', '1', '/v1/sys/authority/getAuthorityList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (18, 'p', '1', '/v1/sys/authority/updateAuthority', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (40, 'p', '1', '/v1/sys/base/uploadFileImg', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (37, 'p', '1', '/v1/sys/casbin/getPathByAuthorityId', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (38, 'p', '1', '/v1/sys/casbin/updateCasbinData', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (39, 'p', '1', '/v1/sys/casbin/updateCasbinDataByApiIds', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (7, 'p', '1', '/v1/sys/deleteUser', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (28, 'p', '1', '/v1/sys/dictionary/createSysDictionary', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (35, 'p', '1', '/v1/sys/dictionary/createSysDictionaryInfo', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (31, 'p', '1', '/v1/sys/dictionary/deleteSysDictionary', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (36, 'p', '1', '/v1/sys/dictionary/deleteSysDictionaryInfo', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (29, 'p', '1', '/v1/sys/dictionary/getSysDictionaryDetails', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (32, 'p', '1', '/v1/sys/dictionary/getSysDictionaryInfoList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (33, 'p', '1', '/v1/sys/dictionary/getSysDictionaryInfoListDetailsById', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (41, 'p', '1', '/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (27, 'p', '1', '/v1/sys/dictionary/getSysDictionaryList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (30, 'p', '1', '/v1/sys/dictionary/updateSysDictionary', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (34, 'p', '1', '/v1/sys/dictionary/updateSysDictionaryInfo', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (3, 'p', '1', '/v1/sys/getUserList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (1, 'p', '1', '/v1/sys/login', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (10, 'p', '1', '/v1/sys/menu/addBaseMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (15, 'p', '1', '/v1/sys/menu/deleteBaseMenu', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (13, 'p', '1', '/v1/sys/menu/getBaseMenuById', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (11, 'p', '1', '/v1/sys/menu/getBaseMenuTree', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (8, 'p', '1', '/v1/sys/menu/getMenu', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (12, 'p', '1', '/v1/sys/menu/getMenuAuthority', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (9, 'p', '1', '/v1/sys/menu/getMenuList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (14, 'p', '1', '/v1/sys/menu/updateBaseMenu', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (2, 'p', '1', '/v1/sys/randomImage', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (4, 'p', '1', '/v1/sys/register', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (6, 'p', '1', '/v1/sys/resetUserPassword', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (5, 'p', '1', '/v1/sys/updateUserInfo', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (63, 'p', '10', '/v1/sys/api/createApi', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (64, 'p', '10', '/v1/sys/api/deleteApi', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (66, 'p', '10', '/v1/sys/api/deleteApisByIds', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (65, 'p', '10', '/v1/sys/api/getAllApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (62, 'p', '10', '/v1/sys/api/getApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (67, 'p', '10', '/v1/sys/api/updateApi', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (58, 'p', '10', '/v1/sys/authority/addAuthorityMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (60, 'p', '10', '/v1/sys/authority/createAuthority', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (61, 'p', '10', '/v1/sys/authority/deleteAuthority', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (57, 'p', '10', '/v1/sys/authority/getAuthorityList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (59, 'p', '10', '/v1/sys/authority/updateAuthority', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (83, 'p', '10', '/v1/sys/base/sendEmailCode', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (81, 'p', '10', '/v1/sys/base/uploadFileImg', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (78, 'p', '10', '/v1/sys/casbin/getPathByAuthorityId', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (79, 'p', '10', '/v1/sys/casbin/updateCasbinData', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (80, 'p', '10', '/v1/sys/casbin/updateCasbinDataByApiIds', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (48, 'p', '10', '/v1/sys/deleteUser', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (69, 'p', '10', '/v1/sys/dictionary/createSysDictionary', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (76, 'p', '10', '/v1/sys/dictionary/createSysDictionaryInfo', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (72, 'p', '10', '/v1/sys/dictionary/deleteSysDictionary', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (77, 'p', '10', '/v1/sys/dictionary/deleteSysDictionaryInfo', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (70, 'p', '10', '/v1/sys/dictionary/getSysDictionaryDetails', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (73, 'p', '10', '/v1/sys/dictionary/getSysDictionaryInfoList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (74, 'p', '10', '/v1/sys/dictionary/getSysDictionaryInfoListDetailsById', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (82, 'p', '10', '/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (68, 'p', '10', '/v1/sys/dictionary/getSysDictionaryList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (71, 'p', '10', '/v1/sys/dictionary/updateSysDictionary', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (75, 'p', '10', '/v1/sys/dictionary/updateSysDictionaryInfo', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (44, 'p', '10', '/v1/sys/getUserList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (42, 'p', '10', '/v1/sys/login', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (51, 'p', '10', '/v1/sys/menu/addBaseMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (56, 'p', '10', '/v1/sys/menu/deleteBaseMenu', 'DELETE', '', '', '');
INSERT INTO `casbin_rule` VALUES (54, 'p', '10', '/v1/sys/menu/getBaseMenuById', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (52, 'p', '10', '/v1/sys/menu/getBaseMenuTree', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (49, 'p', '10', '/v1/sys/menu/getMenu', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (53, 'p', '10', '/v1/sys/menu/getMenuAuthority', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (50, 'p', '10', '/v1/sys/menu/getMenuList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (55, 'p', '10', '/v1/sys/menu/updateBaseMenu', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (43, 'p', '10', '/v1/sys/randomImage', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (45, 'p', '10', '/v1/sys/register', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (47, 'p', '10', '/v1/sys/resetUserPassword', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (46, 'p', '10', '/v1/sys/updateUserInfo', 'PUT', '', '', '');

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
) ENGINE = InnoDB AUTO_INCREMENT = 43 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_apis
-- ----------------------------
INSERT INTO `sys_apis` VALUES (1, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/login', '系统用户登录', '系统-无需权限', 'POST');
INSERT INTO `sys_apis` VALUES (2, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/randomImage', '获取验证码', '系统-无需权限', 'GET');
INSERT INTO `sys_apis` VALUES (3, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/getUserList', '分页获取用户列表', '用户', 'GET');
INSERT INTO `sys_apis` VALUES (4, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/register', '新增（注册）用户 - 管理员', '用户', 'POST');
INSERT INTO `sys_apis` VALUES (5, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/updateUserInfo', '修改用户信息', '用户', 'PUT');
INSERT INTO `sys_apis` VALUES (6, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/resetUserPassword', '重置用户密码 默认密码：goZero', '用户', 'PUT');
INSERT INTO `sys_apis` VALUES (7, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/deleteUser', '删除用户', '用户', 'DELETE');
INSERT INTO `sys_apis` VALUES (8, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/menu/getMenu', '获取菜单', '菜单', 'GET');
INSERT INTO `sys_apis` VALUES (9, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/menu/getMenuList', '分页获取base_menu列表', '菜单', 'GET');
INSERT INTO `sys_apis` VALUES (10, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/menu/addBaseMenu', '新增 base_menu', '菜单', 'POST');
INSERT INTO `sys_apis` VALUES (11, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/menu/getBaseMenuTree', '获取用户动态路由树  -- 用于角色管理的设置权限', '菜单', 'GET');
INSERT INTO `sys_apis` VALUES (12, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/menu/getMenuAuthority', '获取指定角色menu  -- 用于角色管理的设置权限', '菜单', 'GET');
INSERT INTO `sys_apis` VALUES (13, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/menu/getBaseMenuById', '根据id获取菜单', '菜单', 'GET');
INSERT INTO `sys_apis` VALUES (14, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/menu/updateBaseMenu', '更新系统菜单', '菜单', 'PUT');
INSERT INTO `sys_apis` VALUES (15, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/menu/deleteBaseMenu', '删除系统菜单', '菜单', 'DELETE');
INSERT INTO `sys_apis` VALUES (16, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/authority/getAuthorityList', '获取角色列表', '角色', 'GET');
INSERT INTO `sys_apis` VALUES (17, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/authority/addAuthorityMenu', '增加角色和base_menu关联关系', '角色', 'POST');
INSERT INTO `sys_apis` VALUES (18, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/authority/updateAuthority', '更新角色信息', '角色', 'PUT');
INSERT INTO `sys_apis` VALUES (19, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/authority/createAuthority', '创建角色', '角色', 'POST');
INSERT INTO `sys_apis` VALUES (20, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/authority/deleteAuthority', '删除角色', '角色', 'DELETE');
INSERT INTO `sys_apis` VALUES (21, '2024-01-15 13:14:52.000', '2024-01-15 10:35:23.438', NULL, '/v1/sys/api/getApiList', '获取api列表', 'api', 'GET');
INSERT INTO `sys_apis` VALUES (22, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/api/createApi', '创建/增加 api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (23, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/api/deleteApi', '删除 api列表', 'api', 'DELETE');
INSERT INTO `sys_apis` VALUES (24, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/api/getAllApiList', '获取 所有api', 'api', 'GET');
INSERT INTO `sys_apis` VALUES (25, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/api/deleteApisByIds', '删除多条api', 'api', 'DELETE');
INSERT INTO `sys_apis` VALUES (26, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/api/updateApi', '更新api', 'api', 'PUT');
INSERT INTO `sys_apis` VALUES (27, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/getSysDictionaryList', '获取SysDictionary列表', '字典', 'GET');
INSERT INTO `sys_apis` VALUES (28, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/createSysDictionary', '新建SysDictionary', '字典', 'POST');
INSERT INTO `sys_apis` VALUES (29, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/getSysDictionaryDetails', '根据ID或者type获取SysDictionary', '字典', 'GET');
INSERT INTO `sys_apis` VALUES (30, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/updateSysDictionary', '更新SysDictionary', '字典', 'PUT');
INSERT INTO `sys_apis` VALUES (31, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/deleteSysDictionary', '删除SysDictionary', '字典', 'DELETE');
INSERT INTO `sys_apis` VALUES (32, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/getSysDictionaryInfoList', '获取SysDictionaryInfo列表', '字典', 'GET');
INSERT INTO `sys_apis` VALUES (33, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/getSysDictionaryInfoListDetailsById', '根据id获取SysDictionaryInfo详情', '字典', 'GET');
INSERT INTO `sys_apis` VALUES (34, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/updateSysDictionaryInfo', '更新SysDictionaryInfo', '字典', 'PUT');
INSERT INTO `sys_apis` VALUES (35, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/createSysDictionaryInfo', '创建SysDictionaryInfo', '字典', 'POST');
INSERT INTO `sys_apis` VALUES (36, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/deleteSysDictionaryInfo', '删除SysDictionaryInfo', '字典', 'DELETE');
INSERT INTO `sys_apis` VALUES (37, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/casbin/getPathByAuthorityId', '根据角色id获取对应的casbin数据', 'casbin', 'GET');
INSERT INTO `sys_apis` VALUES (38, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/casbin/updateCasbinData', '更新一个角色的对应的casbin数据', 'casbin', 'PUT');
INSERT INTO `sys_apis` VALUES (39, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/casbin/updateCasbinDataByApiIds', '更新一个角色的对应的casbin数据 用api的ids 查数据', 'casbin', 'PUT');
INSERT INTO `sys_apis` VALUES (40, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/base/uploadFileImg', '上传图片', 'base', 'POST');
INSERT INTO `sys_apis` VALUES (41, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/dictionary/getSysDictionaryInfoListDetailsByType', '根据Type获取SysDictionaryInfo详情', '字典', 'GET');
INSERT INTO `sys_apis` VALUES (42, '2024-01-15 13:14:52.000', '2024-01-15 13:14:52.000', NULL, '/v1/sys/base/sendEmailCode', '邮箱发送验证码 - 暂时用于注册账号', 'base', 'POST');

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
) ENGINE = InnoDB AUTO_INCREMENT = 101 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_authorities
-- ----------------------------
INSERT INTO `sys_authorities` VALUES ('2024-01-15 13:14:52.000', '2024-01-16 17:49:10.662', NULL, 1, '超级用户', 0, 'authority');
INSERT INTO `sys_authorities` VALUES ('2024-01-16 16:56:06.053', '2024-01-16 17:27:37.537', NULL, 10, '测试角色', 0, 'index');

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
INSERT INTO `sys_authority_menus` VALUES (1, 1);
INSERT INTO `sys_authority_menus` VALUES (1, 10);
INSERT INTO `sys_authority_menus` VALUES (2, 1);
INSERT INTO `sys_authority_menus` VALUES (2, 10);
INSERT INTO `sys_authority_menus` VALUES (3, 1);
INSERT INTO `sys_authority_menus` VALUES (3, 10);
INSERT INTO `sys_authority_menus` VALUES (4, 1);
INSERT INTO `sys_authority_menus` VALUES (4, 10);
INSERT INTO `sys_authority_menus` VALUES (5, 1);
INSERT INTO `sys_authority_menus` VALUES (5, 10);
INSERT INTO `sys_authority_menus` VALUES (6, 1);
INSERT INTO `sys_authority_menus` VALUES (6, 10);
INSERT INTO `sys_authority_menus` VALUES (7, 1);
INSERT INTO `sys_authority_menus` VALUES (7, 10);
INSERT INTO `sys_authority_menus` VALUES (39, 1);
INSERT INTO `sys_authority_menus` VALUES (39, 10);
INSERT INTO `sys_authority_menus` VALUES (40, 1);
INSERT INTO `sys_authority_menus` VALUES (40, 10);
INSERT INTO `sys_authority_menus` VALUES (41, 1);
INSERT INTO `sys_authority_menus` VALUES (41, 10);
INSERT INTO `sys_authority_menus` VALUES (42, 1);
INSERT INTO `sys_authority_menus` VALUES (42, 10);
INSERT INTO `sys_authority_menus` VALUES (43, 1);
INSERT INTO `sys_authority_menus` VALUES (43, 10);

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
) ENGINE = InnoDB AUTO_INCREMENT = 23 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_base_menu_btns
-- ----------------------------

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
) ENGINE = InnoDB AUTO_INCREMENT = 21 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_base_menu_parameters
-- ----------------------------

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
) ENGINE = InnoDB AUTO_INCREMENT = 44 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = REDUNDANT;

-- ----------------------------
-- Records of sys_base_menus
-- ----------------------------
INSERT INTO `sys_base_menus` VALUES (1, '2023-11-17 15:00:25.000', '2024-01-18 09:54:30.123', NULL, 0, 0, 'admin', 'superAdmin', 0, 'views/superAdmin/index.vue', 1, '', 0, 0, '超级管理员', 'StopFilled', 0);
INSERT INTO `sys_base_menus` VALUES (2, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, 1, 'authority', 'authority', 0, 'views/superAdmin/authority/authority.vue', 1, '', 0, 0, '角色管理', 'CheckSquareOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (3, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, 1, 'menu', 'menu', 0, 'views/superAdmin/menu/menu.vue', 2, '', 1, 0, '菜单管理', 'BugOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (4, '2023-11-17 15:00:25.093', '2024-01-10 11:41:33.339', NULL, 0, 1, 'api', 'api', 0, 'views/superAdmin/api/api.vue', 3, '', 1, 0, 'api管理', 'ColumnHeightOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (5, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, 1, 'user', 'user', 0, 'views/superAdmin/user/user.vue', 4, '', 0, 0, '用户管理', 'BugOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (6, '2023-11-17 15:00:25.093', '2024-01-12 10:03:17.383', NULL, 0, 1, 'dictionary', 'dictionary', 0, 'views/superAdmin/dictionary/dictionary.vue', 5, '', 0, 0, '字典管理', 'BugOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (7, '2024-01-16 17:27:17.123', '2024-01-18 09:54:26.485', NULL, 0, 0, 'index', 'index', 0, 'views/index.vue', 0, '', 0, 0, '首页', 'AntDesignOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (39, '2024-01-17 14:12:08.123', '2024-01-18 10:11:35.493', NULL, 0, 0, '11', 'test1', 0, 'views/test/index.vue', 3, '', 0, 0, 'test1', '', 0);
INSERT INTO `sys_base_menus` VALUES (40, '2024-01-17 14:12:21.554', '2024-01-17 14:12:21.554', NULL, 0, 39, 'test2', 'test2', 0, 'views/routerHolder.vue', 0, '', 0, 0, 'test2', '', 0);
INSERT INTO `sys_base_menus` VALUES (41, '2024-01-17 15:38:46.677', '2024-01-17 15:38:46.677', NULL, 0, 39, 'test3', 'test3', 0, 'views/routerHolder.vue', 0, '', 0, 0, 'test3', 'AccountBookOutlined', 0);
INSERT INTO `sys_base_menus` VALUES (42, '2024-01-17 16:05:34.342', '2024-01-17 16:05:34.342', NULL, 0, 40, 'test4', 'test4', 0, 'views/routerHolder.vue', 0, '', 0, 0, 'test4', 'AccountBookTwoTone', 0);
INSERT INTO `sys_base_menus` VALUES (43, '2024-01-17 16:05:57.207', '2024-01-17 16:05:57.207', NULL, 0, 40, 'test5', 'test5', 0, 'views/routerHolder.vue', 0, '', 0, 0, 'test5', 'AccountBookOutlined', 0);

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
) ENGINE = InnoDB AUTO_INCREMENT = 34 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of sys_dictionaries
-- ----------------------------
INSERT INTO `sys_dictionaries` VALUES (1, '2023-11-17 15:00:24.877', '2023-11-17 15:00:24.896', NULL, '性别', 'gender', 1, '性别字典');
INSERT INTO `sys_dictionaries` VALUES (2, '2024-01-16 17:38:10.566', '2024-01-16 17:38:10.566', NULL, 'api请求', 'method', 1, 'api请求');

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
) ENGINE = InnoDB AUTO_INCREMENT = 49 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of sys_dictionary_info
-- ----------------------------
INSERT INTO `sys_dictionary_info` VALUES (1, '2024-01-16 17:37:18.206', '2024-01-16 17:37:18.206', NULL, '男', 1, '男-拓展值', 1, 0, 1);
INSERT INTO `sys_dictionary_info` VALUES (2, '2024-01-16 17:37:35.413', '2024-01-16 17:37:35.413', NULL, '女', 2, '女-拓展值', 1, 2, 1);
INSERT INTO `sys_dictionary_info` VALUES (3, '2024-01-16 17:38:19.939', '2024-01-18 17:26:35.798', NULL, 'POST', 2, 'POST', 1, 0, 2);
INSERT INTO `sys_dictionary_info` VALUES (4, '2024-01-16 17:38:36.624', '2024-01-16 17:38:36.624', NULL, 'GET', 2, 'GET', 1, 0, 2);
INSERT INTO `sys_dictionary_info` VALUES (5, '2024-01-16 17:38:45.246', '2024-01-16 17:38:45.246', NULL, 'PUT', 3, 'PUT', 1, 0, 2);
INSERT INTO `sys_dictionary_info` VALUES (6, '2024-01-16 17:38:56.280', '2024-01-16 17:38:56.280', NULL, 'DELETE', 4, 'DELETE', 1, 0, 2);

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
INSERT INTO `sys_user_authority` VALUES (1, 1);
INSERT INTO `sys_user_authority` VALUES (2, 10);
INSERT INTO `sys_user_authority` VALUES (3, 1);
INSERT INTO `sys_user_authority` VALUES (62, 10);
INSERT INTO `sys_user_authority` VALUES (63, 10);
INSERT INTO `sys_user_authority` VALUES (64, 10);
INSERT INTO `sys_user_authority` VALUES (65, 10);

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
  `active_color` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT '#fff' COMMENT '活跃颜色',
  `authority_id` bigint UNSIGNED NULL DEFAULT 10000001 COMMENT '用户角色ID',
  `phone` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '用户手机号',
  `email` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '用户邮箱',
  `enable` bigint NULL DEFAULT 1 COMMENT '用户是否被冻结 1正常 2冻结',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_sys_users_deleted_at`(`deleted_at`) USING BTREE,
  INDEX `idx_sys_users_uuid`(`uuid`) USING BTREE,
  INDEX `idx_sys_users_username`(`username`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 66 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_users
-- ----------------------------
INSERT INTO `sys_users` VALUES (1, '2024-01-10 11:54:46.167', '2024-01-10 11:54:46.167', NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'admin', '$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m', 'Mr.root', 'dark', '\r\nhttps://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg', '#000', '#fff', 1, '15888888888', 'TypeScript@111.com', 1);
INSERT INTO `sys_users` VALUES (2, '2024-01-10 09:27:22.905', '2024-01-16 17:48:41.643', NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'yanghao', '$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m', 'Mr.杨浩', 'dark', '\r\nhttps://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg', '#000', '#fff', 10, '15888888888', 'TypeScript@111.com', 1);
INSERT INTO `sys_users` VALUES (3, '2024-01-10 09:26:44.047', '2024-01-17 16:16:34.486', NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'wenlong', '$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m', 'Mr.文龙', 'dark', 'https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705479393275_49814642.jfif', '#000', '#fff', 1, '15888888888', 'TypeScript@111.com', 1);
INSERT INTO `sys_users` VALUES (62, '2024-01-17 10:08:17.097', '2024-01-17 10:08:17.097', NULL, 'f3149922-22b3-4866-ad1f-e3d58ae4d9a5', '用户001', '$2a$10$fpVOkDRMLD3V9l34VfIN6uSvzOhgARViFsKk6pUehoYt//n..2F9C', '用户001', 'dark', 'https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg', '#fff', '#fff', 888, '', '', 1);
INSERT INTO `sys_users` VALUES (63, '2024-01-17 11:05:43.707', '2024-01-17 11:05:43.707', NULL, 'db637c17-6461-4c93-b1ea-18b33d8b0b09', '11', '$2a$10$xH2J8jDZC0XpD.jefci4l.hgLtN7fNZbDevh6nIwuZeQFb17kS4US', '11', 'dark', 'https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg', '#fff', '#1890ff', 888, '', '', 1);
INSERT INTO `sys_users` VALUES (64, '2024-01-17 11:05:59.012', '2024-01-17 11:05:59.012', NULL, 'bfaba08a-4e9c-48b3-aaa5-fa8ebfe9d585', '22', '$2a$10$LnpbXmU51bWHhmRSVKusb.mXG7MHtlpYAfvNFQ2Dw.DxOSVSQManC', '22', 'dark', 'https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg', '#fff', '#1890ff', 888, '', '', 1);
INSERT INTO `sys_users` VALUES (65, '2024-01-17 11:07:41.376', '2024-01-17 11:07:41.376', NULL, 'd8f57f39-e366-4895-ad8e-d2af02321d80', '33', '$2a$10$woWz6ZlbewPKbNPKmNI/9.KhLhjpClPtPeLaSroDWCk6UQPdju8Om', '33', 'dark', 'https://go-zero-admin.oss-cn-beijing.aliyuncs.com/go-zero-admin/1705302257128_4d5477fec18994d0adb839d47fa30937_1.jpg', '#fff', '#1890ff', 888, '', '', 1);

SET FOREIGN_KEY_CHECKS = 1;
