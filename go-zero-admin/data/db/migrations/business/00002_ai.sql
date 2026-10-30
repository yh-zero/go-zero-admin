-- +goose Up
-- 阶段 1+2：AI 助手平台表（会话/消息/工作流/任务）

CREATE TABLE ai_conversation (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  user_id     BIGINT UNSIGNED NOT NULL COMMENT '所属用户',
  title       VARCHAR(128) NOT NULL DEFAULT '' COMMENT '会话标题',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at  DATETIME NULL DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (id),
  KEY idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 会话';

CREATE TABLE ai_message (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  conversation_id BIGINT UNSIGNED NOT NULL COMMENT '会话ID',
  role            VARCHAR(16) NOT NULL COMMENT '角色：user/assistant/tool/system',
  content         TEXT NOT NULL COMMENT '消息内容',
  tool_calls      JSON NULL COMMENT '工具调用记录（JSON 数组）',
  tokens          INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '消耗 token 数',
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_conversation (conversation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 消息';

CREATE TABLE ai_workflow (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  name             VARCHAR(128) NOT NULL COMMENT '工作流名称',
  definition       JSON NOT NULL COMMENT 'JSON DSL：{"steps":[{"name","tool","input","approval"}]}',
  interval_seconds INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '调度间隔秒数，0=仅手动触发',
  enabled          TINYINT NOT NULL DEFAULT 1 COMMENT '是否启用',
  next_run_at      DATETIME NULL DEFAULT NULL COMMENT '下次执行时间',
  created_by       BIGINT UNSIGNED NOT NULL COMMENT '创建者用户ID',
  created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at       DATETIME NULL DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (id),
  KEY idx_next_run (enabled, next_run_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 工作流';

CREATE TABLE ai_task (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  workflow_id  BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '工作流ID（对话触发的任务为空）',
  conversation_id BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '会话ID',
  status       VARCHAR(24) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/running/pending_approval/succeeded/failed/rejected',
  input        TEXT NULL COMMENT '输入（JSON）',
  result       TEXT NULL COMMENT '执行结果',
  error        VARCHAR(512) NOT NULL DEFAULT '' COMMENT '错误信息',
  approved_by  BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '审批人ID',
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  completed_at DATETIME NULL DEFAULT NULL COMMENT '完成时间',
  PRIMARY KEY (id),
  KEY idx_workflow (workflow_id),
  KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 任务';

-- +goose Down
DROP TABLE IF EXISTS ai_task;
DROP TABLE IF EXISTS ai_workflow;
DROP TABLE IF EXISTS ai_message;
DROP TABLE IF EXISTS ai_conversation;
