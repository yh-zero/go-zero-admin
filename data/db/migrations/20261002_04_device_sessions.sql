CREATE TABLE IF NOT EXISTS `sys_device_sessions` (
  `id` varchar(36) NOT NULL,
  `user_id` bigint NOT NULL,
  `authority_id` bigint NOT NULL,
  `session_version` bigint NOT NULL,
  `ip` varchar(64) NOT NULL DEFAULT '',
  `user_agent` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `revoked_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_device_user` (`user_id`, `expires_at`),
  KEY `idx_device_expiry` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
