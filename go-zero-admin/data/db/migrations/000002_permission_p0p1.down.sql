DELETE FROM `sys_permission`;
DROP TABLE IF EXISTS `sys_permission`;
DROP TABLE IF EXISTS `sys_policy_state`;
DELETE FROM `casbin_rule` WHERE `ptype`='p' AND `v2`='/v1/sys/dept/directory' AND `v3`='GET' AND `v0` IN ('scope_all','scope_dept_children','scope_dept','scope_self');
DELETE FROM `sys_role_menu` WHERE `menu_id` IN (9001460,9001461,9001462,9001463);
DELETE FROM `sys_menu` WHERE `id` IN (9001460,9001461,9001462,9001463);
DELETE FROM `sys_api` WHERE `id` IN (135,136,137,143);
