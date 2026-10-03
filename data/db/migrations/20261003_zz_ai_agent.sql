-- Append-only asynchronous AI jobs. Never execute provider calls in migrations.
CREATE TABLE IF NOT EXISTS `sys_ai_conversations` (
  `id` varchar(36) COLLATE utf8mb4_bin NOT NULL,
  `owner_id` bigint NOT NULL,
  `title` varchar(160) NOT NULL,
  `next_sequence` bigint NOT NULL DEFAULT 0,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`), KEY `idx_agent_conversation_owner` (`owner_id`,`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS `sys_ai_messages` (
  `id` varchar(36) COLLATE utf8mb4_bin NOT NULL,
  `owner_id` bigint NOT NULL,
  `conversation_id` varchar(36) COLLATE utf8mb4_bin NOT NULL,
  `run_id` varchar(36) COLLATE utf8mb4_bin NOT NULL,
  `role` varchar(16) NOT NULL,
  `content` longtext NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`), UNIQUE KEY `uk_agent_message_run_role` (`run_id`,`role`),
  KEY `idx_agent_message_history` (`conversation_id`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS `sys_ai_runs` (
  `id` varchar(36) COLLATE utf8mb4_bin NOT NULL,
  `owner_id` bigint NOT NULL,
  `authority_id` bigint NOT NULL,
  `session_id` varchar(36) COLLATE utf8mb4_bin NOT NULL DEFAULT '',
  `session_version` bigint NOT NULL,
  `request_id` varchar(36) COLLATE utf8mb4_bin NOT NULL,
  `request_hash` char(64) COLLATE utf8mb4_bin NOT NULL,
  `conversation_id` varchar(36) COLLATE utf8mb4_bin NOT NULL,
  `question` longtext NOT NULL,
  `sequence` bigint NOT NULL,
  `status` varchar(16) NOT NULL,
  `cancel_requested` tinyint(1) NOT NULL DEFAULT 0,
  `text` longtext NOT NULL,
  `provider` varchar(64) NOT NULL DEFAULT '',
  `model` varchar(128) NOT NULL DEFAULT '',
  `input_tokens` bigint NOT NULL DEFAULT 0,
  `output_tokens` bigint NOT NULL DEFAULT 0,
  `extra_json` longtext NOT NULL,
  `error_code` varchar(32) NOT NULL DEFAULT '',
  `error` varchar(256) NOT NULL DEFAULT '',
  `lease_owner` varchar(36) COLLATE utf8mb4_bin NOT NULL DEFAULT '',
  `lease_until` datetime(3) DEFAULT NULL,
  `queue_expires_at` datetime(3) NOT NULL,
  `deadline_at` datetime(3) DEFAULT NULL,
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`), UNIQUE KEY `uk_agent_run_request` (`owner_id`,`request_id`),
  UNIQUE KEY `uk_agent_run_sequence` (`conversation_id`,`sequence`),
  KEY `idx_agent_run_owner_status` (`owner_id`,`status`),
  KEY `idx_agent_run_conversation_status` (`conversation_id`,`status`),
  KEY `idx_agent_run_claim` (`status`,`queue_expires_at`), KEY `idx_agent_run_lease` (`lease_until`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
CREATE TEMPORARY TABLE ai_api_seed (path varchar(191),method varchar(16),description varchar(191)) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
INSERT INTO ai_api_seed VALUES
('/v1/ai/info','GET','查询AI助手配置'),('/v1/ai/runs','POST','提交AI任务'),
('/v1/ai/runs/:id','GET','查询本人AI任务'),('/v1/ai/runs/:id/cancel','POST','取消本人AI任务'),
('/v1/ai/conversations','GET','查询本人AI会话'),('/v1/ai/conversations/:id/messages','GET','查询本人AI会话消息');
CREATE TEMPORARY TABLE ai_existing_paths AS
WITH RECURSIVE resolved AS (
  SELECT id,parent_id,CAST(REGEXP_REPLACE(CONCAT('/',TRIM(BOTH '/' FROM COALESCE(path,''))), '/{2,}', '/') AS CHAR(16384)) COLLATE utf8mb4_general_ci AS full_path,
    CAST(id AS CHAR(8192)) AS visited,0 AS depth
  FROM sys_base_menus WHERE deleted_at IS NULL AND (COALESCE(parent_id,0)=0 OR LEFT(path,1)='/')
  UNION ALL
  SELECT child.id,child.parent_id,CAST(REGEXP_REPLACE(CONCAT(TRIM(TRAILING '/' FROM parent.full_path),'/',COALESCE(child.path,'')), '/{2,}', '/') AS CHAR(16384)) COLLATE utf8mb4_general_ci,
    CONCAT(parent.visited,',',child.id),parent.depth+1
  FROM sys_base_menus child JOIN resolved parent ON child.parent_id=parent.id
  WHERE child.deleted_at IS NULL AND LEFT(COALESCE(child.path,''),1)<>'/' AND parent.depth<128 AND FIND_IN_SET(child.id,parent.visited)=0
)
SELECT id,parent_id,LOWER(TRIM(TRAILING '/' FROM full_path)) AS full_path FROM resolved;

START TRANSACTION;
SELECT authority_id FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL FOR UPDATE;
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),seed.path,seed.method,'ai-agent',seed.description FROM ai_api_seed seed
WHERE NOT EXISTS(SELECT 1 FROM sys_apis existing WHERE existing.path=seed.path AND existing.method=seed.method AND existing.deleted_at IS NULL);
-- Casbin keyMatch2 recognizes the :id paths. Keep all other role grants.
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1',seed.path,seed.method FROM ai_api_seed seed
WHERE EXISTS(SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
  AND NOT EXISTS(SELECT 1 FROM casbin_rule existing WHERE existing.ptype='p' AND existing.v0='1' AND existing.v1 COLLATE utf8mb4_general_ci=seed.path AND existing.v2 COLLATE utf8mb4_general_ci=seed.method);
SET @ai_policy_changes=ROW_COUNT();
UPDATE sys_policy_versions SET version=version+1,updated_at=NOW(3) WHERE id=1 AND @ai_policy_changes>0;

SET @ai_parent=(SELECT id FROM sys_base_menus WHERE name='superAdmin' AND deleted_at IS NULL ORDER BY id LIMIT 1);
SET @ai_parent_path=(SELECT full_path FROM ai_existing_paths WHERE id=@ai_parent LIMIT 1);
INSERT INTO sys_base_menus (created_at,updated_at,menu_level,parent_id,path,name,hidden,component,sort,active_name,keep_alive,default_menu,title,icon,close_tab)
SELECT NOW(3),NOW(3),0,@ai_parent,'ai-agent','ai-agent',0,'views/business/ai-agent/index.vue',65,'',0,0,'AI助手','',0
WHERE @ai_parent IS NOT NULL AND @ai_parent_path IS NOT NULL
  AND NOT EXISTS(SELECT 1 FROM sys_base_menus WHERE deleted_at IS NULL AND (name='ai-agent' OR path='ai-agent'))
  AND NOT EXISTS(SELECT 1 FROM ai_existing_paths WHERE full_path=LOWER(CONCAT(@ai_parent_path,'/ai-agent')));
CREATE TEMPORARY TABLE ai_granted_menus (id bigint PRIMARY KEY);
INSERT INTO ai_granted_menus SELECT id FROM sys_base_menus WHERE name='ai-agent' AND component='views/business/ai-agent/index.vue' AND deleted_at IS NULL;
INSERT INTO sys_authority_menus (sys_base_menu_id,sys_authority_authority_id)
WITH RECURSIVE ancestors AS (
  SELECT menu.id,menu.parent_id,CAST(menu.id AS CHAR(8192)) AS visited,0 AS depth FROM sys_base_menus menu JOIN ai_granted_menus granted ON granted.id=menu.id
  UNION ALL
  SELECT parent.id,parent.parent_id,CONCAT(child.visited,',',parent.id),child.depth+1 FROM sys_base_menus parent JOIN ancestors child ON child.parent_id=parent.id
  WHERE parent.deleted_at IS NULL AND child.depth<128 AND FIND_IN_SET(parent.id,child.visited)=0
)
SELECT DISTINCT ancestors.id,1 FROM ancestors WHERE EXISTS(SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
  AND NOT EXISTS(SELECT 1 FROM sys_authority_menus existing WHERE existing.sys_base_menu_id=ancestors.id AND existing.sys_authority_authority_id=1);
CREATE TEMPORARY TABLE ai_button_seed (name varchar(191),description varchar(191)) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
INSERT INTO ai_button_seed VALUES('run','提交AI任务'),('cancel','取消AI任务');
INSERT INTO sys_base_menu_btns (created_at,updated_at,name,`desc`,sys_base_menu_id)
SELECT NOW(3),NOW(3),seed.name,seed.description,menu.id FROM ai_button_seed seed CROSS JOIN ai_granted_menus menu
WHERE NOT EXISTS(SELECT 1 FROM sys_base_menu_btns existing WHERE existing.sys_base_menu_id=menu.id AND existing.name=seed.name AND existing.deleted_at IS NULL);
INSERT INTO sys_authority_btns (authority_id,sys_menu_id,sys_base_menu_btn_id)
SELECT 1,menu.id,button.id FROM ai_granted_menus menu JOIN sys_base_menu_btns button ON button.sys_base_menu_id=menu.id AND button.name IN ('run','cancel') AND button.deleted_at IS NULL
WHERE EXISTS(SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
  AND NOT EXISTS(SELECT 1 FROM sys_authority_btns existing WHERE existing.authority_id=1 AND existing.sys_menu_id=menu.id AND existing.sys_base_menu_btn_id=button.id);
INSERT INTO sys_audit_logs (event_type,module,action,object,result,params)
SELECT 'operation','ai-agent','seedAgentModule','20261003_zz_ai_agent.sql','success','{}'
WHERE NOT EXISTS(SELECT 1 FROM sys_audit_logs WHERE action='seedAgentModule' AND object='20261003_zz_ai_agent.sql');
COMMIT;
SELECT 'ai-agent' AS name,CASE WHEN menu.id IS NOT NULL AND menu.component='views/business/ai-agent/index.vue' THEN 'READY'
  WHEN menu.id IS NOT NULL THEN 'SKIPPED: existing customized component preserved'
  ELSE 'SKIPPED: parent missing or canonical route/name conflict preserved' END AS migration_status
FROM (SELECT 1) marker LEFT JOIN sys_base_menus menu ON menu.name='ai-agent' AND menu.deleted_at IS NULL;
DROP TEMPORARY TABLE ai_api_seed;
DROP TEMPORARY TABLE ai_existing_paths;
DROP TEMPORARY TABLE ai_granted_menus;
DROP TEMPORARY TABLE ai_button_seed;
