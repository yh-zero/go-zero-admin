-- Department/position membership and current-role data scopes.
-- Deletion is allowed only after references have been removed.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sys_departments (
  id BIGINT NOT NULL AUTO_INCREMENT, created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), deleted_at DATETIME(3) NULL,
  parent_id BIGINT NOT NULL DEFAULT 0, name VARCHAR(64) NOT NULL, code VARCHAR(64) NOT NULL,
  sort BIGINT NOT NULL DEFAULT 0, status BIGINT NOT NULL DEFAULT 1, leader VARCHAR(64) NOT NULL DEFAULT '',
  PRIMARY KEY(id), UNIQUE KEY uk_department_code(code), KEY idx_department_parent(parent_id), KEY idx_department_deleted(deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sys_positions (
  id BIGINT NOT NULL AUTO_INCREMENT, created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), deleted_at DATETIME(3) NULL,
  name VARCHAR(64) NOT NULL, code VARCHAR(64) NOT NULL, sort BIGINT NOT NULL DEFAULT 0, status BIGINT NOT NULL DEFAULT 1,
  PRIMARY KEY(id), UNIQUE KEY uk_position_code(code), KEY idx_position_deleted(deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sys_user_departments (
  user_id BIGINT NOT NULL, department_id BIGINT NOT NULL, updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY(user_id), KEY idx_user_department(department_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sys_user_positions (
  user_id BIGINT NOT NULL, position_id BIGINT NOT NULL,
  PRIMARY KEY(user_id,position_id), KEY idx_user_position(position_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sys_role_data_scopes (
  authority_id BIGINT NOT NULL, scope VARCHAR(32) NOT NULL DEFAULT 'self',
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), PRIMARY KEY(authority_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sys_role_scope_departments (
  authority_id BIGINT NOT NULL, department_id BIGINT NOT NULL,
  PRIMARY KEY(authority_id,department_id), KEY idx_scope_department(department_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
