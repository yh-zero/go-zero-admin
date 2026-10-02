-- Repair canonical route collisions left by the already-applied module seed.
-- Preserve custom rows and grants; only untouched original seed rows can move.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
CREATE TEMPORARY TABLE frontend_route_seed (
  name VARCHAR(191) PRIMARY KEY,path VARCHAR(191),component VARCHAR(191),title VARCHAR(191),sort BIGINT
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
INSERT INTO frontend_route_seed VALUES
('audit','audit','views/business/system/audit/index.vue','操作审计',60),
('files','files','views/business/system/files/index.vue','文件资源',61),
('sessions','sessions','views/business/system/sessions/index.vue','在线会话',62),
('organization-departments','organization/departments','views/business/system/organization/departments.vue','部门管理',63),
('organization-positions','organization/positions','views/business/system/organization/positions.vue','岗位管理',64);

START TRANSACTION;
-- Same serialization guard used by application menu writes.
SELECT authority_id FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL FOR UPDATE;
CREATE TEMPORARY TABLE frontend_canonical_paths AS
WITH RECURSIVE resolved AS (
  SELECT id,parent_id,
    CAST(REGEXP_REPLACE(CONCAT('/',TRIM(BOTH '/' FROM COALESCE(path,''))), '/{2,}', '/') AS CHAR(16384)) COLLATE utf8mb4_general_ci AS full_path,
    CAST(id AS CHAR(8192)) AS visited,0 AS depth
  FROM sys_base_menus WHERE deleted_at IS NULL AND (COALESCE(parent_id,0)=0 OR LEFT(path,1)='/')
  UNION ALL
  SELECT child.id,child.parent_id,
    CAST(REGEXP_REPLACE(CONCAT(TRIM(TRAILING '/' FROM parent.full_path),'/',COALESCE(child.path,'')), '/{2,}', '/') AS CHAR(16384)) COLLATE utf8mb4_general_ci,
    CONCAT(parent.visited,',',child.id),parent.depth+1
  FROM sys_base_menus child JOIN resolved parent ON child.parent_id=parent.id
  WHERE child.deleted_at IS NULL AND LEFT(COALESCE(child.path,''),1)<>'/'
    AND parent.depth<128 AND FIND_IN_SET(child.id,parent.visited)=0
)
SELECT id,parent_id,LOWER(TRIM(TRAILING '/' FROM full_path)) AS full_path FROM resolved;
-- MySQL cannot reopen one temporary table with multiple aliases in a query.
CREATE TEMPORARY TABLE frontend_canonical_parents AS SELECT * FROM frontend_canonical_paths;
CREATE TEMPORARY TABLE frontend_canonical_others AS SELECT * FROM frontend_canonical_paths;

CREATE TEMPORARY TABLE frontend_route_conflicts AS
SELECT menu.id,menu.name,menu.parent_id,seed.path AS original_path,parent.full_path AS parent_path,
  (BINARY menu.path=BINARY seed.path AND BINARY menu.component=BINARY seed.component
    AND BINARY menu.name=BINARY seed.name AND BINARY menu.title=BINARY seed.title
    AND menu.sort=seed.sort AND menu.menu_level=0 AND menu.hidden=0
    AND COALESCE(menu.active_name,'')='' AND COALESCE(menu.keep_alive,0)=0
    AND COALESCE(menu.default_menu,0)=0 AND COALESCE(menu.icon,'')='' AND COALESCE(menu.close_tab,0)=0
    AND menu.created_at=menu.updated_at AND menu.parent_id=directory.id
    AND NOT EXISTS(SELECT 1 FROM sys_base_menus child WHERE child.parent_id=menu.id AND child.deleted_at IS NULL)
    AND NOT EXISTS(SELECT 1 FROM sys_base_menu_parameters parameter WHERE parameter.sys_base_menu_id=menu.id AND parameter.deleted_at IS NULL)
  ) AS untouched
FROM sys_base_menus menu JOIN frontend_route_seed seed ON menu.name=seed.name
JOIN frontend_canonical_paths route ON route.id=menu.id
LEFT JOIN frontend_canonical_parents parent ON parent.id=menu.parent_id
LEFT JOIN sys_base_menus directory ON directory.id=menu.parent_id AND directory.name='superAdmin' AND directory.deleted_at IS NULL
WHERE menu.deleted_at IS NULL AND EXISTS(
  SELECT 1 FROM frontend_canonical_others other WHERE other.id<>menu.id AND other.full_path=route.full_path
);

CREATE TEMPORARY TABLE frontend_route_candidates AS
WITH RECURSIVE attempts AS (SELECT 0 AS n UNION ALL SELECT n+1 FROM attempts WHERE n<99)
SELECT conflict.id,attempts.n,
  CONCAT(conflict.original_path,'-module-',conflict.id,IF(attempts.n=0,'',CONCAT('-',attempts.n))) AS new_path,
  LOWER(CONCAT(conflict.parent_path,'/',conflict.original_path,'-module-',conflict.id,IF(attempts.n=0,'',CONCAT('-',attempts.n)))) AS full_path
FROM frontend_route_conflicts conflict CROSS JOIN attempts
WHERE conflict.untouched=1 AND conflict.parent_path IS NOT NULL;

CREATE TEMPORARY TABLE frontend_route_available AS
SELECT candidate.* FROM frontend_route_candidates candidate
WHERE CHAR_LENGTH(candidate.new_path)<=191
  AND NOT EXISTS(SELECT 1 FROM sys_base_menus menu WHERE menu.deleted_at IS NULL AND LOWER(menu.path)=LOWER(candidate.new_path))
  AND NOT EXISTS(SELECT 1 FROM frontend_canonical_paths existing WHERE existing.full_path=candidate.full_path);
CREATE TEMPORARY TABLE frontend_route_first_free AS SELECT id,MIN(n) AS n FROM frontend_route_available GROUP BY id;
CREATE TEMPORARY TABLE frontend_route_updates AS
SELECT available.id,available.new_path FROM frontend_route_available available
JOIN frontend_route_first_free first_free ON first_free.id=available.id AND first_free.n=available.n;

UPDATE sys_base_menus menu JOIN frontend_route_updates repair ON repair.id=menu.id
SET menu.path=repair.new_path,menu.updated_at=NOW(3);
INSERT INTO sys_audit_logs (event_type,module,action,object,result,params)
SELECT 'operation','menu','repairFrontendRoute',CONCAT('20261003_frontend_routes_compat.sql:',repair.id),'success','{}'
FROM frontend_route_updates repair
WHERE NOT EXISTS(SELECT 1 FROM sys_audit_logs event WHERE event.action='repairFrontendRoute' AND event.object=CONCAT('20261003_frontend_routes_compat.sql:',repair.id));
COMMIT;

SELECT seed.name,CASE WHEN menu.id IS NULL THEN 'MISSING: original seed unavailable'
  WHEN repair.id IS NOT NULL THEN CONCAT('REPAIRED: ',repair.new_path)
  WHEN conflict.id IS NOT NULL THEN 'SKIPPED: customized seed or no safe candidate; resolve manually'
  ELSE 'UNCHANGED' END AS migration_status
FROM frontend_route_seed seed LEFT JOIN sys_base_menus menu ON menu.name=seed.name AND menu.deleted_at IS NULL
LEFT JOIN frontend_route_conflicts conflict ON conflict.id=menu.id
LEFT JOIN frontend_route_updates repair ON repair.id=menu.id ORDER BY seed.sort;
DROP TEMPORARY TABLE frontend_route_seed;
DROP TEMPORARY TABLE frontend_canonical_paths;
DROP TEMPORARY TABLE frontend_canonical_parents;
DROP TEMPORARY TABLE frontend_canonical_others;
DROP TEMPORARY TABLE frontend_route_conflicts;
DROP TEMPORARY TABLE frontend_route_candidates;
DROP TEMPORARY TABLE frontend_route_available;
DROP TEMPORARY TABLE frontend_route_first_free;
DROP TEMPORARY TABLE frontend_route_updates;
