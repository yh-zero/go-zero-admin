-- Durable permission synchronization; the version changes atomically with policies.
CREATE TABLE IF NOT EXISTS sys_policy_versions (
  id BIGINT UNSIGNED NOT NULL,
  version BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
INSERT IGNORE INTO sys_policy_versions (id, version) VALUES (1, 0);
