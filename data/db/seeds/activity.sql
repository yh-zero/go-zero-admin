-- Fresh-install activity examples only; apply after seeds/demo.sql.
-- All records are explicitly labelled 内置演示, not evidence of real actions.
-- No login credentials, online sessions, queued jobs or provider requests are created.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
START TRANSACTION;
SET @activity_now = UTC_TIMESTAMP(3);

INSERT INTO sys_audit_logs(
 actor_id,actor_name,authority_id,event_type,module,action,object,path,method,
 result,status_code,ip,trace_id,duration_ms,params,created_at
) VALUES
 (1,'admin',1,'operation','user','内置演示：账号管理记录','内置演示：后台运营账号',
  '/v1/sys/updateUserInfo','PUT','success',200,'192.0.2.10','',0,
  JSON_OBJECT('userid',2,'enable',1),DATE_SUB(@activity_now,INTERVAL 3 MINUTE)),
 (1,'admin',1,'operation','files','内置演示：文件管理记录','内置演示：无真实文件操作',
  '/v1/sys/base/uploadFileImg','POST','success',200,'192.0.2.10','',0,
  JSON_OBJECT(),DATE_SUB(@activity_now,INTERVAL 2 MINUTE)),
 (2,'operator',2,'operation','authority','内置演示：越权操作被拒绝','内置演示：系统角色配置',
  '/v1/sys/authority/updateAuthority','PUT','failure',403,'192.0.2.20','',0,
  JSON_OBJECT('authorityid',1),DATE_SUB(@activity_now,INTERVAL 1 MINUTE));

-- The normal history reader joins messages to their run and filters by owner.
-- A terminal run is never claimed by the worker, even when AI is later enabled.
SET @activity_conversation = 'd0000000-0000-4000-8000-000000000001';
SET @activity_run = 'd0000000-0000-4000-8000-000000000002';
SET @activity_request = 'd0000000-0000-4000-8000-000000000003';
SET @activity_question = '【内置演示】第一次使用这个后台，可以从哪些页面开始体验？';
SET @activity_answer = CONCAT(
 '## 内置演示：开始体验',CHAR(10),CHAR(10),
 '这段对话是随初始化数据库提供的静态示例，未调用云模型、未执行工具，也未产生模型费用。',CHAR(10),CHAR(10),
 '| 页面 | 可以体验的内容 |',CHAR(10),
 '| --- | --- |',CHAR(10),
 '| 用户管理、角色管理 | 比较管理员、运营人员和只读观察员的菜单、按钮及接口权限。 |',CHAR(10),
 '| App 用户、App 会员 | 查看普通、会员和待审核示例账号，以及会员权益和到期时间。 |',CHAR(10),
 '| 文件资源 | 上传公开或私有图片，预览、下载，并体验头像引用与删除保护。 |',CHAR(10),
 '| 操作审计 | 查看标注为“内置演示”的成功和拒绝记录，尝试筛选与详情。 |',CHAR(10),CHAR(10),
 '配置并启用模型后，只有你主动发送新问题才会创建新的 AI 任务；当前历史示例不会自动运行。'
);

INSERT INTO sys_ai_conversations(id,owner_id,title,next_sequence,created_at,updated_at)
VALUES (@activity_conversation,1,'【内置演示】后台功能导览',1,
 DATE_SUB(@activity_now,INTERVAL 1 MINUTE),@activity_now);

INSERT INTO sys_ai_runs(
 id,owner_id,authority_id,session_id,session_version,request_id,request_hash,
 conversation_id,question,sequence,status,cancel_requested,text,provider,model,
 input_tokens,output_tokens,extra_json,error_code,error,lease_owner,lease_until,
 queue_expires_at,deadline_at,started_at,finished_at,created_at,updated_at
) VALUES (
 @activity_run,1,1,'',1,@activity_request,
 SHA2(CONCAT('{"ConversationID":"","Question":',JSON_QUOTE(@activity_question),'}'),256),
 @activity_conversation,@activity_question,1,'succeeded',0,@activity_answer,'内置演示','static-demo',
 0,0,'[]','','','',NULL,
 @activity_now,NULL,DATE_SUB(@activity_now,INTERVAL 1 MINUTE),@activity_now,
 DATE_SUB(@activity_now,INTERVAL 1 MINUTE),@activity_now
);

INSERT INTO sys_ai_messages(id,owner_id,conversation_id,run_id,role,content,created_at)
VALUES
 ('d0000000-0000-4000-8000-000000000004',1,@activity_conversation,@activity_run,
  'user',@activity_question,DATE_SUB(@activity_now,INTERVAL 1 MINUTE)),
 ('d0000000-0000-4000-8000-000000000005',1,@activity_conversation,@activity_run,
  'assistant',@activity_answer,@activity_now);

COMMIT;
SET @activity_answer = NULL;
SET @activity_question = NULL;
SET @activity_request = NULL;
SET @activity_run = NULL;
SET @activity_conversation = NULL;
SET @activity_now = NULL;
