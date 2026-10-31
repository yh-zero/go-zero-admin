-- 全模块联调补充数据：只用于 gozero-new-mysql 的 gozero_admin / gozero_platform。
-- 共 45 行：角色 1、字典 1、App 用户 8、会员 6、会话 4、消息 12、工作流 3、任务 7、租户 3。
-- 固定 ID 逐条声明，不认领整个区间；先双向检查身份，再只补缺失行，不覆盖已有可编辑内容。
-- App 密码哈希从现有管理员复制；租户随机凭证仅在数据库内生成，不写入源文件或输出。
-- 所有工作流禁用且无调度；租户仅用于展示，没有部署栈，禁止对其执行生命周期或重置密码操作。
-- 不改变表结构、旧账号、文件对象或缓存；任一断言失败由 mysql 退出并回滚本事务。

SET NAMES utf8mb4;
USE gozero_admin;
START TRANSACTION;
CREATE TEMPORARY TABLE demo_full_checks (check_name VARCHAR(64) PRIMARY KEY);
INSERT INTO demo_full_checks VALUES ('required_base');
INSERT INTO demo_full_checks SELECT 'required_base' WHERE
  NOT EXISTS (SELECT 1 FROM sys_user WHERE id=1 AND deleted_at IS NULL AND password_hash<>'')
  OR NOT EXISTS (SELECT 1 FROM sys_user WHERE id=9001100 AND username='demo_user_01' AND deleted_at IS NULL);

CREATE TEMPORARY TABLE demo_full_role LIKE sys_role;
CREATE TEMPORARY TABLE demo_full_dict LIKE sys_dict;
CREATE TEMPORARY TABLE demo_full_app_user LIKE app_user;
CREATE TEMPORARY TABLE demo_full_grant LIKE app_membership_grant;
CREATE TEMPORARY TABLE demo_full_conversation LIKE ai_conversation;
CREATE TEMPORARY TABLE demo_full_message LIKE ai_message;
CREATE TEMPORARY TABLE demo_full_workflow LIKE ai_workflow;
CREATE TEMPORARY TABLE demo_full_task LIKE ai_task;
CREATE TEMPORARY TABLE demo_full_tenant LIKE gozero_platform.tenant_registry;

INSERT INTO demo_full_role (id,name,code,description,data_scope,sort,status) VALUES
  (9010001,'demo_full_停用角色','demo_full_disabled','全模块联调：停用角色，无授权关联','SELF',99,2);
INSERT INTO demo_full_dict (id,name,code,items,description,status) VALUES
  (9010010,'demo_full_空字典','demo_full_empty',JSON_ARRAY(),'全模块联调：空字典项边界',1);

-- 使用保留示例邮箱和不可投递的 000 前缀号码；不发送短信或邮件。
INSERT INTO demo_full_app_user
  (id,account,identity_kind,password_hash,nickname,status,review_reason,email_verified_at,phone_verified_at,last_login_at)
SELECT 9010101,'demo_full_active@example.test','email',password_hash,'demo_full_邮箱已激活','active','测试样例：已审核',NOW(),NULL,NOW() FROM sys_user WHERE id=1
UNION ALL SELECT 9010102,'demo_full_pending@example.test','email',password_hash,'demo_full_邮箱待审核','pending','',NULL,NULL,NULL FROM sys_user WHERE id=1
UNION ALL SELECT 9010103,'demo_full_disabled@example.test','email',password_hash,'demo_full_邮箱已停用','disabled','测试样例：账号停用',NOW(),NULL,NULL FROM sys_user WHERE id=1
UNION ALL SELECT 9010104,'demo_full_rejected@example.test','email',password_hash,'demo_full_邮箱已驳回','rejected','测试样例：审核资料不完整',NULL,NULL,NULL FROM sys_user WHERE id=1
UNION ALL SELECT 9010105,'0009010105','phone',password_hash,'demo_full_手机已激活','active','测试样例：已审核',NULL,NOW(),NOW() FROM sys_user WHERE id=1
UNION ALL SELECT 9010106,'0009010106','phone',password_hash,'demo_full_手机待审核','pending','',NULL,NULL,NULL FROM sys_user WHERE id=1
UNION ALL SELECT 9010107,'0009010107','phone',password_hash,'demo_full_手机已停用','disabled','测试样例：账号停用',NULL,NOW(),NULL FROM sys_user WHERE id=1
UNION ALL SELECT 9010108,'0009010108','phone',password_hash,'demo_full_手机已驳回','rejected','测试样例：审核资料不完整',NULL,NULL,NULL FROM sys_user WHERE id=1;

INSERT INTO demo_full_grant
  (id,user_id,plan_code,benefits,request_id,granted_by,starts_at,expires_at,revoked,revoked_at) VALUES
  (9010201,9010101,'vip','profile.export','demo_full_grant_active',1,DATE_SUB(NOW(),INTERVAL 1 DAY),DATE_ADD(NOW(),INTERVAL 30 DAY),0,NULL),
  (9010202,9010105,'vip','profile.export','demo_full_grant_expired',1,DATE_SUB(NOW(),INTERVAL 60 DAY),DATE_SUB(NOW(),INTERVAL 30 DAY),0,NULL),
  (9010203,9010105,'vip','profile.export','demo_full_grant_future',1,DATE_ADD(NOW(),INTERVAL 30 DAY),DATE_ADD(NOW(),INTERVAL 60 DAY),0,NULL),
  (9010204,9010101,'vip','profile.export','demo_full_grant_revoked',1,DATE_SUB(NOW(),INTERVAL 7 DAY),DATE_ADD(NOW(),INTERVAL 7 DAY),1,NOW()),
  (9010205,9010101,'vip','profile.export','demo_full_grant_overlap',1,DATE_SUB(NOW(),INTERVAL 1 DAY),DATE_ADD(NOW(),INTERVAL 7 DAY),0,NULL),
  (9010206,9010105,'vip','profile.export','demo_full_grant_self',NULL,DATE_SUB(NOW(),INTERVAL 1 DAY),DATE_ADD(NOW(),INTERVAL 7 DAY),0,NULL);

INSERT INTO demo_full_conversation (id,user_id,title) VALUES
  (9010301,1,'demo_full_四种消息角色'),
  (9010302,1,'demo_full_多轮与工具消息'),
  (9010303,9001100,'demo_full_其他用户会话'),
  (9010304,1,'demo_full_空会话');
INSERT INTO demo_full_message (id,conversation_id,role,content,tool_calls,tokens,created_at) VALUES
  (9010401,9010301,'system','demo_full_模拟系统说明，不连接模型。',NULL,0,'2026-10-31 09:00:01'),
  (9010402,9010301,'user','demo_full_请统计示例文本。',NULL,6,'2026-10-31 09:00:02'),
  (9010403,9010301,'assistant','demo_full_模拟工具调用。',JSON_ARRAY(JSON_OBJECT('id','demo_full_call_01','type','function','function',JSON_OBJECT('name','text_stats','arguments','{"text":"demo sample"}'))),12,'2026-10-31 09:00:03'),
  (9010404,9010301,'tool','demo_full_模拟结果：字符数 11，单词数 2，行数 1。',NULL,0,'2026-10-31 09:00:04'),
  (9010405,9010302,'user','demo_full_第一轮：你好。',NULL,4,'2026-10-31 09:01:01'),
  (9010406,9010302,'assistant','demo_full_第一轮：这是保存的示例回复。',NULL,8,'2026-10-31 09:01:02'),
  (9010407,9010302,'user','demo_full_第二轮：中文、English 与多行\n第二行。',NULL,10,'2026-10-31 09:01:03'),
  (9010408,9010302,'assistant','demo_full_第二轮：模拟工具调用。',JSON_ARRAY(JSON_OBJECT('id','demo_full_call_02','type','function','function',JSON_OBJECT('name','text_stats','arguments','{"text":"example"}'))),9,'2026-10-31 09:01:04'),
  (9010409,9010302,'tool','demo_full_模拟结果：字符数 7，单词数 1，行数 1。',NULL,0,'2026-10-31 09:01:05'),
  (9010410,9010302,'assistant','demo_full_最终回复：统计完成。',NULL,6,'2026-10-31 09:01:06'),
  (9010411,9010303,'user','demo_full_其他用户的示例提问。',NULL,5,'2026-10-31 09:02:01'),
  (9010412,9010303,'assistant','demo_full_其他用户的模拟回复。',NULL,7,'2026-10-31 09:02:02');

INSERT INTO demo_full_workflow (id,name,definition,interval_seconds,enabled,next_run_at,created_by) VALUES
  (9010501,'demo_full_空步骤边界',JSON_OBJECT('steps',JSON_ARRAY()),0,0,NULL,1),
  (9010502,'demo_full_本地文本统计',JSON_OBJECT('steps',JSON_ARRAY(JSON_OBJECT('name','示例统计','tool','text_stats','input',JSON_OBJECT('text','demo full fixture'),'approval',FALSE))),0,0,NULL,1),
  (9010503,'demo_full_审批展示',JSON_OBJECT('steps',JSON_ARRAY(JSON_OBJECT('name','审批前统计','tool','text_stats','input',JSON_OBJECT('text','demo full approval'),'approval',TRUE),JSON_OBJECT('name','后续统计','tool','text_stats','input',JSON_OBJECT('text','demo full after approval'),'approval',FALSE))),0,0,NULL,1);
INSERT INTO demo_full_task
  (id,workflow_id,conversation_id,status,input,result,error,approved_by,created_at,completed_at) VALUES
  (9010601,9010502,NULL,'pending','{"fixture":"demo_full","case":"pending"}',NULL,'',NULL,'2026-10-31 09:10:01',NULL),
  (9010602,9010502,NULL,'running','{"fixture":"demo_full","case":"running"}',NULL,'',NULL,'2026-10-31 09:10:02',NULL),
  (9010603,9010503,NULL,'pending_approval','{"fixture":"demo_full","case":"approve"}','demo_full_模拟待审批结果','',NULL,'2026-10-31 09:10:03',NULL),
  (9010604,9010502,9010301,'succeeded','{"fixture":"demo_full","case":"succeeded"}','demo_full_模拟成功结果','',1,'2026-10-31 09:10:04','2026-10-31 09:11:04'),
  (9010605,9010502,9010302,'failed','{"fixture":"demo_full","case":"failed"}',NULL,'demo_full_模拟工具失败，不代表真实调用',NULL,'2026-10-31 09:10:05','2026-10-31 09:11:05'),
  (9010606,9010503,NULL,'rejected','{"fixture":"demo_full","case":"rejected"}','demo_full_模拟拒绝结果','',1,'2026-10-31 09:10:06','2026-10-31 09:11:06'),
  (9010607,9010503,9010303,'pending_approval','{"fixture":"demo_full","case":"reject"}','demo_full_第二条模拟待审批结果','',NULL,'2026-10-31 09:10:07',NULL);

INSERT INTO demo_full_tenant
  (id,slug,domain,compose_project,status,mysql_root_password,jwt_access_secret,app_jwt_access_secret,error_msg,created_by) VALUES
  (9010701,'demo-full-stopped','demo-full-stopped.localhost','gozero-demo-full-stopped','stopped',HEX(RANDOM_BYTES(32)),HEX(RANDOM_BYTES(32)),HEX(RANDOM_BYTES(32)),'demo_full_纯测试展示，未部署容器；禁止启停、删除栈或重置管理员密码。',1),
  (9010702,'demo-full-error','demo-full-error.localhost','gozero-demo-full-error','error',HEX(RANDOM_BYTES(32)),HEX(RANDOM_BYTES(32)),HEX(RANDOM_BYTES(32)),'demo_full_模拟部署失败；纯测试展示，未部署容器，禁止生命周期操作。',1),
  (9010703,'demo-full-idle','demo-full-idle.localhost','gozero-demo-full-idle','stopped',HEX(RANDOM_BYTES(32)),HEX(RANDOM_BYTES(32)),HEX(RANDOM_BYTES(32)),'demo_full_空闲展示样例，未部署容器；禁止生命周期操作。',NULL);

-- 双向身份校验：既防固定 ID 被占用，也防同一自然身份存在于其他 ID。
INSERT INTO demo_full_checks VALUES
  ('identity_role'),('identity_dict'),('identity_app_user'),('identity_grant'),
  ('identity_conversation'),('identity_message'),('identity_workflow'),('identity_task'),('identity_tenant'),
  ('verify_admin'),('verify_app'),('verify_conversation'),('verify_workflow_task'),('verify_tenant');
INSERT INTO demo_full_checks
SELECT 'identity_role' FROM demo_full_role e JOIN sys_role a ON a.id=e.id OR a.code=e.code
WHERE a.id<>e.id OR BINARY a.code<>BINARY e.code LIMIT 1;
INSERT INTO demo_full_checks
SELECT 'identity_dict' FROM demo_full_dict e JOIN sys_dict a ON a.id=e.id OR a.code=e.code
WHERE a.id<>e.id OR BINARY a.code<>BINARY e.code LIMIT 1;
INSERT INTO demo_full_checks
SELECT 'identity_app_user' FROM demo_full_app_user e JOIN app_user a ON a.id=e.id OR a.account=e.account
WHERE a.id<>e.id OR BINARY a.account<>BINARY e.account OR BINARY a.identity_kind<>BINARY e.identity_kind LIMIT 1;
INSERT INTO demo_full_checks
SELECT 'identity_grant' FROM demo_full_grant e JOIN app_membership_grant a ON a.id=e.id OR a.request_id=e.request_id
WHERE a.id<>e.id OR BINARY a.request_id<>BINARY e.request_id OR a.user_id<>e.user_id OR BINARY a.plan_code<>BINARY e.plan_code LIMIT 1;
INSERT INTO demo_full_checks
SELECT 'identity_conversation' FROM demo_full_conversation e JOIN ai_conversation a ON a.id=e.id OR (a.user_id=e.user_id AND a.title=e.title)
WHERE a.id<>e.id OR a.user_id<>e.user_id OR BINARY a.title<>BINARY e.title LIMIT 1;
INSERT INTO demo_full_checks
SELECT 'identity_message' FROM demo_full_message e JOIN ai_message a ON a.id=e.id OR (a.conversation_id=e.conversation_id AND a.role=e.role AND a.content=e.content)
WHERE a.id<>e.id OR a.conversation_id<>e.conversation_id OR BINARY a.role<>BINARY e.role OR BINARY a.content<>BINARY e.content LIMIT 1;
INSERT INTO demo_full_checks
SELECT 'identity_workflow' FROM demo_full_workflow e JOIN ai_workflow a ON a.id=e.id OR a.name=e.name
WHERE a.id<>e.id OR BINARY a.name<>BINARY e.name OR a.created_by<>e.created_by LIMIT 1;
INSERT INTO demo_full_checks
SELECT 'identity_task' FROM demo_full_task e JOIN ai_task a ON a.id=e.id OR a.input=e.input
WHERE a.id<>e.id OR NOT (a.workflow_id<=>e.workflow_id) OR NOT (a.conversation_id<=>e.conversation_id)
  OR NOT (BINARY a.input<=>BINARY e.input) LIMIT 1;
INSERT INTO demo_full_checks
SELECT 'identity_tenant' FROM demo_full_tenant e JOIN gozero_platform.tenant_registry a
  ON a.id=e.id OR a.slug=e.slug OR a.domain=e.domain OR a.compose_project=e.compose_project
WHERE a.id<>e.id OR BINARY a.slug<>BINARY e.slug OR BINARY a.domain<>BINARY e.domain
  OR BINARY a.compose_project<>BINARY e.compose_project LIMIT 1;

-- 管理补充：插入后立即核对，再推进 App 模块。
INSERT INTO sys_role SELECT e.* FROM demo_full_role e LEFT JOIN sys_role a ON a.id=e.id WHERE a.id IS NULL;
INSERT INTO sys_dict SELECT e.* FROM demo_full_dict e LEFT JOIN sys_dict a ON a.id=e.id WHERE a.id IS NULL;
INSERT INTO demo_full_checks SELECT 'verify_admin' WHERE
  (SELECT COUNT(*) FROM sys_role a JOIN demo_full_role e ON a.id=e.id)<>1
  OR (SELECT COUNT(*) FROM sys_dict a JOIN demo_full_dict e ON a.id=e.id)<>1;
SELECT 'admin_verified' AS module_check,
  (SELECT COUNT(*) FROM sys_role a JOIN demo_full_role e ON a.id=e.id) AS role_rows,
  (SELECT COUNT(*) FROM sys_dict a JOIN demo_full_dict e ON a.id=e.id) AS dict_rows,
  (SELECT COUNT(*) FROM sys_role WHERE id=9010001 AND status=2) AS disabled_roles,
  (SELECT COUNT(*) FROM sys_dict WHERE id=9010010 AND JSON_LENGTH(items)=0) AS empty_dicts;

-- App：会员须关联真实 App 用户，管理员关联仅指向已有后台用户。
INSERT INTO app_user SELECT e.* FROM demo_full_app_user e LEFT JOIN app_user a ON a.id=e.id WHERE a.id IS NULL;
INSERT INTO app_membership_grant SELECT e.* FROM demo_full_grant e LEFT JOIN app_membership_grant a ON a.id=e.id WHERE a.id IS NULL;
INSERT INTO demo_full_checks SELECT 'verify_app' WHERE
  (SELECT COUNT(*) FROM app_user a JOIN demo_full_app_user e ON a.id=e.id)<>8
  OR (SELECT COUNT(*) FROM app_membership_grant a JOIN demo_full_grant e ON a.id=e.id)<>6
  OR EXISTS (SELECT 1 FROM app_membership_grant a LEFT JOIN app_user u ON u.id=a.user_id WHERE a.id IN (9010201,9010202,9010203,9010204,9010205,9010206) AND u.id IS NULL)
  OR EXISTS (SELECT 1 FROM app_membership_grant a LEFT JOIN sys_user u ON u.id=a.granted_by WHERE a.id IN (9010201,9010202,9010203,9010204,9010205,9010206) AND a.granted_by IS NOT NULL AND u.id IS NULL);
SELECT 'app_verified' AS module_check,
  (SELECT COUNT(*) FROM app_user a JOIN demo_full_app_user e ON a.id=e.id) AS user_rows,
  (SELECT COUNT(*) FROM app_membership_grant a JOIN demo_full_grant e ON a.id=e.id) AS membership_rows,
  (SELECT COUNT(DISTINCT status) FROM app_user WHERE id IN (9010101,9010102,9010103,9010104,9010105,9010106,9010107,9010108)) AS user_states,
  (SELECT COUNT(DISTINCT identity_kind) FROM app_user WHERE id IN (9010101,9010102,9010103,9010104,9010105,9010106,9010107,9010108)) AS identity_kinds,
  (SELECT SUM(revoked) FROM app_membership_grant WHERE id IN (9010201,9010202,9010203,9010204,9010205,9010206)) AS revoked_grants;

-- AI 会话：消息仅关联本批会话，会话归属必须存在。
INSERT INTO ai_conversation SELECT e.* FROM demo_full_conversation e LEFT JOIN ai_conversation a ON a.id=e.id WHERE a.id IS NULL;
INSERT INTO ai_message SELECT e.* FROM demo_full_message e LEFT JOIN ai_message a ON a.id=e.id WHERE a.id IS NULL;
INSERT INTO demo_full_checks SELECT 'verify_conversation' WHERE
  (SELECT COUNT(*) FROM ai_conversation a JOIN demo_full_conversation e ON a.id=e.id)<>4
  OR (SELECT COUNT(*) FROM ai_message a JOIN demo_full_message e ON a.id=e.id)<>12
  OR EXISTS (SELECT 1 FROM ai_conversation a LEFT JOIN sys_user u ON u.id=a.user_id WHERE a.id IN (9010301,9010302,9010303,9010304) AND u.id IS NULL)
  OR EXISTS (SELECT 1 FROM ai_message a LEFT JOIN ai_conversation c ON c.id=a.conversation_id WHERE a.id IN (9010401,9010402,9010403,9010404,9010405,9010406,9010407,9010408,9010409,9010410,9010411,9010412) AND c.id IS NULL);
SELECT 'ai_conversation_verified' AS module_check,
  (SELECT COUNT(*) FROM ai_conversation a JOIN demo_full_conversation e ON a.id=e.id) AS conversation_rows,
  (SELECT COUNT(*) FROM ai_message a JOIN demo_full_message e ON a.id=e.id) AS message_rows,
  (SELECT COUNT(DISTINCT role) FROM ai_message WHERE id IN (9010401,9010402,9010403,9010404,9010405,9010406,9010407,9010408,9010409,9010410,9010411,9010412)) AS message_roles,
  (SELECT COUNT(*) FROM ai_message WHERE id IN (9010401,9010402,9010403,9010404,9010405,9010406,9010407,9010408,9010409,9010410,9010411,9010412) AND tool_calls IS NOT NULL) AS tool_call_messages;

-- AI 工作流/任务：只落库展示，不执行工作流或调用模型。
INSERT INTO ai_workflow SELECT e.* FROM demo_full_workflow e LEFT JOIN ai_workflow a ON a.id=e.id WHERE a.id IS NULL;
INSERT INTO ai_task SELECT e.* FROM demo_full_task e LEFT JOIN ai_task a ON a.id=e.id WHERE a.id IS NULL;
INSERT INTO demo_full_checks SELECT 'verify_workflow_task' WHERE
  (SELECT COUNT(*) FROM ai_workflow a JOIN demo_full_workflow e ON a.id=e.id)<>3
  OR (SELECT COUNT(*) FROM ai_task a JOIN demo_full_task e ON a.id=e.id)<>7
  OR EXISTS (SELECT 1 FROM ai_workflow a LEFT JOIN sys_user u ON u.id=a.created_by WHERE a.id IN (9010501,9010502,9010503) AND u.id IS NULL)
  OR EXISTS (SELECT 1 FROM ai_task a LEFT JOIN ai_workflow w ON w.id=a.workflow_id WHERE a.id IN (9010601,9010602,9010603,9010604,9010605,9010606,9010607) AND a.workflow_id IS NOT NULL AND w.id IS NULL)
  OR EXISTS (SELECT 1 FROM ai_task a LEFT JOIN ai_conversation c ON c.id=a.conversation_id WHERE a.id IN (9010601,9010602,9010603,9010604,9010605,9010606,9010607) AND a.conversation_id IS NOT NULL AND c.id IS NULL)
  OR EXISTS (SELECT 1 FROM ai_task a LEFT JOIN sys_user u ON u.id=a.approved_by WHERE a.id IN (9010601,9010602,9010603,9010604,9010605,9010606,9010607) AND a.approved_by IS NOT NULL AND u.id IS NULL);
SELECT 'ai_workflow_task_verified' AS module_check,
  (SELECT COUNT(*) FROM ai_workflow a JOIN demo_full_workflow e ON a.id=e.id) AS workflow_rows,
  (SELECT COUNT(*) FROM ai_task a JOIN demo_full_task e ON a.id=e.id) AS task_rows,
  (SELECT COUNT(*) FROM ai_workflow WHERE id IN (9010501,9010502,9010503) AND enabled=0 AND interval_seconds=0 AND next_run_at IS NULL) AS disabled_unscheduled,
  (SELECT COUNT(DISTINCT status) FROM ai_task WHERE id IN (9010601,9010602,9010603,9010604,9010605,9010606,9010607)) AS task_states;

-- 租户跨库插入仍属于同一事务；现有表没有 admin_account_ready，不增列迁移。
INSERT INTO gozero_platform.tenant_registry
SELECT e.* FROM demo_full_tenant e LEFT JOIN gozero_platform.tenant_registry a ON a.id=e.id WHERE a.id IS NULL;
INSERT INTO demo_full_checks SELECT 'verify_tenant' WHERE
  (SELECT COUNT(*) FROM gozero_platform.tenant_registry a JOIN demo_full_tenant e ON a.id=e.id)<>3
  OR EXISTS (SELECT 1 FROM gozero_platform.tenant_registry a LEFT JOIN sys_user u ON u.id=a.created_by WHERE a.id IN (9010701,9010702,9010703) AND a.created_by IS NOT NULL AND u.id IS NULL);
SELECT 'tenant_verified' AS module_check,
  (SELECT COUNT(*) FROM gozero_platform.tenant_registry a JOIN demo_full_tenant e ON a.id=e.id) AS tenant_rows,
  (SELECT COUNT(*) FROM gozero_platform.tenant_registry WHERE id IN (9010701,9010702,9010703) AND status='stopped') AS stopped_rows,
  (SELECT COUNT(*) FROM gozero_platform.tenant_registry WHERE id IN (9010701,9010702,9010703) AND status='error') AS error_rows;

DROP TEMPORARY TABLE demo_full_role,demo_full_dict,demo_full_app_user,demo_full_grant,
  demo_full_conversation,demo_full_message,demo_full_workflow,demo_full_task,demo_full_tenant,demo_full_checks;
COMMIT;
SELECT 'full_fixture_committed' AS result,45 AS declared_fixture_rows;
