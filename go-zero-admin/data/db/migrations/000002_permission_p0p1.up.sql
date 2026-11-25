-- 002：权限映射、策略版本与权限诊断（兼容已有开发库）
CREATE TABLE IF NOT EXISTS `sys_permission` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `menu_id` bigint unsigned NOT NULL COMMENT '菜单或按钮 ID',
  `api_id` bigint unsigned NOT NULL COMMENT 'API 资源 ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_menu_api` (`menu_id`,`api_id`),
  KEY `idx_api` (`api_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单与API授权映射';

CREATE TABLE IF NOT EXISTS `sys_policy_state` (
  `id` bigint unsigned NOT NULL,
  `revision` bigint unsigned NOT NULL DEFAULT 1,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='权限策略版本状态';
INSERT IGNORE INTO `sys_policy_state` (`id`,`revision`) VALUES (1,1);

INSERT INTO `sys_menu` (`id`,`parent_id`,`title`,`name`,`type`,`path`,`component`,`perm_code`,`icon`,`sort`,`visible`,`status`) VALUES
(9001460,9001302,'查看用户权限','user_permissions',3,'','','sys:user:permissions','',4,0,1),
(9001461,9001302,'批量导入用户','user_import',3,'','','sys:user:import','',5,0,1),
(9001462,0,'组织通讯录','directory',2,'/sys/directory','directory','','dept',5,1,1),
(9001463,9001300,'权限诊断','permission_check',2,'/sys/permission-check','permission_check','','safety',29,1,1)
ON DUPLICATE KEY UPDATE `title`=VALUES(`title`);
INSERT IGNORE INTO `sys_role_menu` (`role_id`,`menu_id`) VALUES
(1,9001460),(1,9001461),(1,9001462),(2,9001462),(3,9001462),(4,9001462),(5,9001462),(1,9001463);

INSERT INTO `sys_api` (`id`,`group_name`,`method`,`path`,`title`) VALUES
(135,'sys/user','GET','/v1/sys/user/:id/permissions','查看用户权限'),
(136,'sys/user','POST','/v1/sys/user/import','批量导入用户'),
(137,'sys/dept','GET','/v1/sys/dept/directory','查看组织通讯录'),
(143,'sys/permission','POST','/v1/sys/permission/check','权限诊断')
ON DUPLICATE KEY UPDATE `title`=VALUES(`title`);
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
('p','scope_all','*','/v1/sys/dept/directory','GET','',''),
('p','scope_dept_children','*','/v1/sys/dept/directory','GET','',''),
('p','scope_dept','*','/v1/sys/dept/directory','GET','',''),
('p','scope_self','*','/v1/sys/dept/directory','GET','','');

INSERT IGNORE INTO `sys_permission` (`menu_id`,`api_id`) VALUES
(9001302,93),(9001303,80),(9001304,69),(9001305,55),(9001306,60),(9001307,29),(9001308,63),(9001308,62),(9001308,140),(9001308,141),(9001309,44),(9001309,139),(9001310,73),(9001310,74),(9001320,92),(9001321,90),(9001322,88),(9001323,91),
(9001324,79),(9001325,77),(9001326,75),(9001327,78),(9001328,68),(9001329,67),(9001330,65),(9001331,53),(9001332,52),(9001333,51),(9001334,59),(9001335,57),(9001336,56),(9001337,28),(9001338,61),(9001339,73),(9001339,74),
(9001361,37),(9001362,34),(9001363,41),(9001364,40),(9001365,42),(9001366,43),(9001367,35),(9001368,36),(9001369,64),(9001369,142),(9001370,39),(9001371,31),(9001372,32),(9001374,87),(9001375,86),(9001376,84),(9001377,83),(9001378,81),
(9001460,135),(9001461,136),(9001462,137),(9001463,143);
