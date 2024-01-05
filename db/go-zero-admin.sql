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
) ENGINE = InnoDB AUTO_INCREMENT = 34 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of casbin_rule
-- ----------------------------
INSERT INTO `casbin_rule` VALUES (14, 'p', '0', '/base/login1111', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (15, 'p', '0', '/jwt/jsonInBlacklist1222', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (32, 'p', '0', '/v1/sys/getUserList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (31, 'p', '0', '/v1/sys/login', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (33, 'p', '0', '/v1/sys/register', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (34, 'p', '0', '/v1/sys/updateUserInfo', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (6, 'p', '100001', '/v1/sys/api/createApi', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (21, 'p', '100001', '/v1/sys/api/getAllApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (7, 'p', '100001', '/v1/sys/api/getAllApiList', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (5, 'p', '100001', '/v1/sys/api/getApiList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (12, 'p', '100001', '/v1/sys/authority/addAuthorityMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (22, 'p', '100001', '/v1/sys/authority/createAuthority', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (4, 'p', '100001', '/v1/sys/authority/getAuthorityList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (18, 'p', '100001', '/v1/sys/authority/updateAuthority', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (8, 'p', '100001', '/v1/sys/casbin/getPathByAuthorityId', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (13, 'p', '100001', '/v1/sys/casbin/updateCasbinData', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (3, 'p', '100001', '/v1/sys/menu/addBaseMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (19, 'p', '100001', '/v1/sys/menu/getBaseMenuById', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (17, 'p', '100001', '/v1/sys/menu/getBaseMenuTree', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (9, 'p', '100001', '/v1/sys/menu/getBaseMenuTree', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (1, 'p', '100001', '/v1/sys/menu/getMenu', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (16, 'p', '100001', '/v1/sys/menu/getMenuAuthority', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (10, 'p', '100001', '/v1/sys/menu/getMenuAuthority', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (11, 'p', '100001', '/v1/sys/menu/getMenuList', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (2, 'p', '100001', '/v1/sys/menu/getMenuList1111', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (20, 'p', '100001', '/v1/sys/menu/updateBaseMenu', 'PUT', '', '', '');
INSERT INTO `casbin_rule` VALUES (25, 'p', '11111', '/base/login', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (24, 'p', '11111', '/jwt/jsonInBlacklist', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (23, 'p', '11111', '/menu/getMenu', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (26, 'p', '11111', '/user/admin_register', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (27, 'p', '11111', '/user/changePassword', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (30, 'p', '11111', '/user/getUserInfo', 'GET', '', '', '');
INSERT INTO `casbin_rule` VALUES (28, 'p', '11111', '/user/setUserAuthority', 'POST', '', '', '');
INSERT INTO `casbin_rule` VALUES (29, 'p', '11111', '/user/setUserInfo', 'PUT', '', '', '');

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
) ENGINE = InnoDB AUTO_INCREMENT = 730 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

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
INSERT INTO `sys_apis` VALUES (704, '2023-12-26 14:58:48.822', '2023-12-26 14:58:48.822', NULL, 'test', '1', 'test', 'GET');
INSERT INTO `sys_apis` VALUES (705, '2024-01-03 16:31:43.688', '2024-01-03 16:31:43.688', NULL, 'test1', '1', 'test', 'GET');
INSERT INTO `sys_apis` VALUES (706, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (707, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (708, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (709, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (710, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (711, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (712, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (713, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
INSERT INTO `sys_apis` VALUES (714, '2023-11-17 15:00:24.717', '2023-11-17 15:00:24.717', NULL, 'v1/sys/api/createApi', '添加api列表', 'api', 'POST');
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
) ENGINE = InnoDB AUTO_INCREMENT = 100004 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_authorities
-- ----------------------------
INSERT INTO `sys_authorities` VALUES ('2024-01-05 09:44:13.630', '2024-01-05 09:44:13.630', NULL, 1, '', NULL, 'dashboard');
INSERT INTO `sys_authorities` VALUES ('2024-01-05 09:44:13.630', '2024-01-05 09:44:13.630', NULL, 2, '', NULL, 'dashboard');
INSERT INTO `sys_authorities` VALUES ('2024-01-05 09:47:15.602', '2024-01-05 09:47:15.602', NULL, 10, '', NULL, 'dashboard');
INSERT INTO `sys_authorities` VALUES ('2024-01-05 09:47:15.602', '2024-01-05 09:47:15.602', NULL, 20, '', NULL, 'dashboard');
INSERT INTO `sys_authorities` VALUES ('2024-01-04 10:05:36.725', '2024-01-04 10:05:36.783', '0000-00-00 00:00:00.000', 11111, 'yanghao', 0, 'dashboard');
INSERT INTO `sys_authorities` VALUES (NULL, '2024-01-03 20:45:22.096', NULL, 100001, '超级用户', 0, 'superAdmin');
INSERT INTO `sys_authorities` VALUES (NULL, NULL, NULL, 100002, 'admin-test', 100001, 'defaultRouter');

-- ----------------------------
-- Table structure for sys_authority_btns
-- ----------------------------
DROP TABLE IF EXISTS `sys_authority_btns`;
CREATE TABLE `sys_authority_btns`  (
  `authority_id` bigint UNSIGNED NULL DEFAULT NULL COMMENT '角色ID',
  `sys_menu_id` bigint UNSIGNED NULL DEFAULT NULL COMMENT '菜单ID',
  `sys_base_menu_btn_id` bigint UNSIGNED NULL DEFAULT NULL COMMENT '菜单按钮ID'
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

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
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of sys_authority_menus
-- ----------------------------
INSERT INTO `sys_authority_menus` VALUES (1, 11111);
INSERT INTO `sys_authority_menus` VALUES (1, 100001);
INSERT INTO `sys_authority_menus` VALUES (2, 100001);
INSERT INTO `sys_authority_menus` VALUES (3, 100001);
INSERT INTO `sys_authority_menus` VALUES (4, 100001);
INSERT INTO `sys_authority_menus` VALUES (5, 100001);
INSERT INTO `sys_authority_menus` VALUES (6, 100001);
INSERT INTO `sys_authority_menus` VALUES (6, 100002);
INSERT INTO `sys_authority_menus` VALUES (7, 100001);
INSERT INTO `sys_authority_menus` VALUES (7, 100002);
INSERT INTO `sys_authority_menus` VALUES (25, 100001);
INSERT INTO `sys_authority_menus` VALUES (25, 100002);
INSERT INTO `sys_authority_menus` VALUES (28, 100001);
INSERT INTO `sys_authority_menus` VALUES (28, 100002);
INSERT INTO `sys_authority_menus` VALUES (29, 100001);
INSERT INTO `sys_authority_menus` VALUES (29, 100002);

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
) ENGINE = InnoDB AUTO_INCREMENT = 18 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

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
INSERT INTO `sys_base_menu_btns` VALUES (16, '2023-12-27 16:02:29.820', '2023-12-27 16:02:29.820', NULL, '22', '22', 28);

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
) ENGINE = InnoDB AUTO_INCREMENT = 16 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

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
INSERT INTO `sys_base_menu_parameters` VALUES (14, '2023-12-27 16:02:29.819', '2023-12-27 16:02:29.819', NULL, 28, 'query', '11', '11');

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
  `parent_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '父菜单ID',
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
) ENGINE = InnoDB AUTO_INCREMENT = 30 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = REDUNDANT;

-- ----------------------------
-- Records of sys_base_menus
-- ----------------------------
INSERT INTO `sys_base_menus` VALUES (1, '2023-11-17 15:00:25.000', '2023-11-17 15:00:25.000', NULL, 0, '0', 'admin', 'superAdmin', 0, 'views/superAdmin/index.vue', 0, '', 0, 0, '超级管理员', 'user', 0);
INSERT INTO `sys_base_menus` VALUES (2, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, '1', 'authority', 'authority', 0, 'views/superAdmin/authority/authority.vue', 1, '', 0, 0, '角色管理', 'avatar', 0);
INSERT INTO `sys_base_menus` VALUES (3, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, '1', 'menu', 'menu', 0, 'views/superAdmin/menu/menu.vue', 2, '', 1, 0, '菜单管理', 'tickets', 0);
INSERT INTO `sys_base_menus` VALUES (4, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, '1', 'api', 'api', 0, 'views/superAdmin/api/api.vue', 3, '', 1, 0, 'api管理', 'platform', 0);
INSERT INTO `sys_base_menus` VALUES (5, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, '1', 'user', 'user', 0, 'views/superAdmin/user/user.vue', 4, '', 0, 0, '用户管理', 'coordinate', 0);
INSERT INTO `sys_base_menus` VALUES (6, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, '1', 'dictionary', 'dictionary', 0, 'views/superAdmin/dictionary/sysDictionary.vue', 5, '', 0, 0, '字典管理', 'notebook', 0);
INSERT INTO `sys_base_menus` VALUES (7, '2023-11-17 15:00:25.093', '2023-11-17 15:00:25.093', NULL, 0, '1', 'operation', 'operation', 0, 'views/superAdmin/operation/sysOperationRecord.vue', 6, '', 0, 0, '操作历史', 'pie-chart', 0);
INSERT INTO `sys_base_menus` VALUES (25, '2023-12-25 06:58:23.563', '2023-12-25 06:58:23.563', NULL, 0, '0', 'route-name', 'route-name6', 0, 'views/routerHolder.vue', 1, '', 0, 0, '展示名称', 'aim', 0);
INSERT INTO `sys_base_menus` VALUES (28, '2023-12-27 16:02:29.818', '2023-12-27 16:02:29.818', NULL, 0, '0', 'test', 'test', 0, 'views/test/index.vue', 12, '123', 0, 0, 'TEST', 'jack', 0);
INSERT INTO `sys_base_menus` VALUES (29, '2023-12-27 16:22:45.316', '2024-01-03 11:06:18.363', NULL, 0, '0', 'admin', 'superAdmin-test', 0, 'view/superAdmin/index.vue', 3, '', 0, 0, '超级管理员-test', 'user', 0);

-- ----------------------------
-- Table structure for sys_user_authority
-- ----------------------------
DROP TABLE IF EXISTS `sys_user_authority`;
CREATE TABLE `sys_user_authority`  (
  `sys_user_id` bigint UNSIGNED NOT NULL,
  `sys_authority_authority_id` bigint UNSIGNED NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`sys_user_id`, `sys_authority_authority_id`) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of sys_user_authority
-- ----------------------------
INSERT INTO `sys_user_authority` VALUES (2, 100001);
INSERT INTO `sys_user_authority` VALUES (3, 100001);
INSERT INTO `sys_user_authority` VALUES (4, 100001);
INSERT INTO `sys_user_authority` VALUES (6, 1);
INSERT INTO `sys_user_authority` VALUES (6, 888);
INSERT INTO `sys_user_authority` VALUES (58, 1);
INSERT INTO `sys_user_authority` VALUES (58, 2);
INSERT INTO `sys_user_authority` VALUES (59, 1);
INSERT INTO `sys_user_authority` VALUES (59, 2);
INSERT INTO `sys_user_authority` VALUES (60, 1);
INSERT INTO `sys_user_authority` VALUES (60, 888);

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
) ENGINE = InnoDB AUTO_INCREMENT = 59 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sys_users
-- ----------------------------
INSERT INTO `sys_users` VALUES (2, NULL, NULL, NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'admin', '$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m', 'Mr.root', 'dark', '', '#fff', '#1890ff', 100001, '15888888888', 'TypeScript@111.com', 1);
INSERT INTO `sys_users` VALUES (3, NULL, NULL, NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'yanghao', '$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m', 'Mr.杨浩', 'dark', '', '#fff', '#1890ff', 100001, '15888888888', 'TypeScript@111.com', 1);
INSERT INTO `sys_users` VALUES (4, NULL, NULL, NULL, 'e2ab86b1-8e3d-4864-80ad-e83131353686', 'wenlong', '$2a$10$0tL51xscR4JV7RQCz34Wc.uYDLuGQdovS9XCXAderJEG7TmkDVi2m', 'Mr.文龙', 'dark', '', '#fff', '#1890ff', 100001, '15888888888', 'TypeScript@111.com', 1);
INSERT INTO `sys_users` VALUES (5, '2024-01-04 15:55:28.837', '2024-01-04 15:55:28.837', '0000-00-00 00:00:00.000', '341d5849-529d-4a4f-afe1-021fa839665c', 'test001', '$2a$10$4BHVIx9lBViG2Ydc2jrEg.buwhvXE0DFR8lYG4Um3arvy.xsa/nGG', 'test001', 'dark', 'https://qmplusimg.henrongyi.top/gva_header.jpg', '#fff', '#1890ff', 1, '', '', 1);
INSERT INTO `sys_users` VALUES (6, '2024-01-04 15:57:12.446', '2024-01-04 15:57:12.446', '0000-00-00 00:00:00.000', 'f3dafc2e-b6fa-4d5c-89b2-984d358d3a3e', 'test001', '$2a$10$tGj.5Xuicy0EEC6fmOFz6uukMWHw64kM7yTUSG0bHiVTW/3/9R.2G', 'test001', 'dark', 'https://qmplusimg.henrongyi.top/gva_header.jpg', '#fff', '#1890ff', 1, '', '', 1);
INSERT INTO `sys_users` VALUES (56, '2024-01-04 17:50:12.256', '2024-01-04 17:50:12.256', NULL, '84a01713-0252-4f2f-82b2-3add4d6cc98b', 'test013', '$2a$10$QRgiwHH/ADumuRqHiUpeaeu6MhBWDMuXDyZgOwW28g3LQtr5iF09.', 'test001', 'dark', 'https://qmplusimg.henrongyi.top/gva_header.jpg', '#fff', '#1890ff', 1, '', '', 1);
INSERT INTO `sys_users` VALUES (57, '2024-01-04 17:52:25.220', '2024-01-04 17:52:25.220', NULL, 'ffb717ac-26d2-4a2a-a01c-c3f96c7d9f07', 'test014', '$2a$10$r.4wtHrKJ/Lb.LDoTzCL6.86qgL/IbYm1BabTGmPhZyJAJlbS70W6', 'test001', 'dark', 'https://qmplusimg.henrongyi.top/gva_header.jpg', '#fff', '#1890ff', 1, '', '', 1);
INSERT INTO `sys_users` VALUES (58, '2024-01-05 09:44:13.575', '2024-01-05 09:44:13.575', NULL, 'e35a4de1-07b3-4e0f-a988-09a9b40da050', 'test002', '$2a$10$XKrMTmbUYBT13qpxK.rbTOl6Wc0Y9cvGAOldFk4zruRla8Y7ArbQS', 'test001', 'dark', 'https://qmplusimg.henrongyi.top/gva_header.jpg', '#fff', '#1890ff', 1, '', '', 1);
INSERT INTO `sys_users` VALUES (59, '2024-01-05 09:44:34.287', '2024-01-05 09:44:34.287', NULL, '16370fe5-61fc-4e42-a052-bcc200f0a313', 'test003', '$2a$10$3MxCOLfuStYleNYk/EX9buyahwt7z51EHUhbH7SDPl/XO70BZMTK2', 'test001', 'dark', 'https://qmplusimg.henrongyi.top/gva_header.jpg', '#fff', '#1890ff', 1, '', '', 1);
INSERT INTO `sys_users` VALUES (60, '2024-01-05 09:47:15.546', '2024-01-05 15:44:20.066', NULL, 'd4b450cd-3729-4dd7-b866-acc53fbf35b3', 'test004', '$2a$10$GPs7ZMVyIzcox39K./5mLu4/.UKATwVBNG.k2E.GSuE11TEYMMgVy', 'test004-update1', 'dark-update1', 'img-update1', '#fff', '#1890ff', 1, '155888888881', 'uniapp@111.com', 1);

SET FOREIGN_KEY_CHECKS = 1;
