-- Incremental frontend menus/buttons. Existing custom names, paths and
-- components are preserved; conflicting entries are reported as SKIPPED.
-- Requires the 20261002 module/audit migrations. HTTP resources are unchanged.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;

CREATE TEMPORARY TABLE frontend_module_seed (
  name VARCHAR(191) NOT NULL,
  path VARCHAR(191) NOT NULL,
  component VARCHAR(191) NOT NULL,
  title VARCHAR(191) NOT NULL,
  sort BIGINT NOT NULL,
  PRIMARY KEY (name)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
INSERT INTO frontend_module_seed VALUES
('audit','audit','views/business/system/audit/index.vue','操作审计',60),
('files','files','views/business/system/files/index.vue','文件资源',61),
('sessions','sessions','views/business/system/sessions/index.vue','在线会话',62),
('organization-departments','organization/departments','views/business/system/organization/departments.vue','部门管理',63),
('organization-positions','organization/positions','views/business/system/organization/positions.vue','岗位管理',64);

-- Resolve complete routes before insertion so an absolute customized path under
-- another parent cannot collide with a new route. The database also enforces
-- active raw-path uniqueness, which is checked separately below.
CREATE TEMPORARY TABLE frontend_existing_paths AS
WITH RECURSIVE resolved AS (
  SELECT id,parent_id,
    CAST(CONCAT('/',TRIM(BOTH '/' FROM COALESCE(path,''))) AS CHAR(2048)) COLLATE utf8mb4_general_ci AS full_path,
    CAST(id AS CHAR(2048)) AS visited,0 AS depth
  FROM sys_base_menus WHERE deleted_at IS NULL AND COALESCE(parent_id,0)=0
  UNION ALL
  SELECT child.id,child.parent_id,
    CAST(CASE WHEN LEFT(child.path,1)='/' THEN CONCAT('/',TRIM(BOTH '/' FROM child.path))
      ELSE CONCAT(TRIM(TRAILING '/' FROM parent.full_path),'/',TRIM(BOTH '/' FROM COALESCE(child.path,''))) END AS CHAR(2048)) COLLATE utf8mb4_general_ci,
    CONCAT(parent.visited,',',child.id),parent.depth+1
  FROM sys_base_menus child JOIN resolved parent ON child.parent_id=parent.id
  WHERE child.deleted_at IS NULL AND parent.depth<64 AND FIND_IN_SET(child.id,parent.visited)=0
)
SELECT id,parent_id,LOWER(full_path) AS full_path FROM resolved;

START TRANSACTION;
SET @frontend_parent = (SELECT id FROM sys_base_menus WHERE name='superAdmin' AND deleted_at IS NULL ORDER BY id LIMIT 1);
SET @frontend_parent_path = (SELECT full_path FROM frontend_existing_paths WHERE id=@frontend_parent LIMIT 1);

INSERT INTO sys_base_menus (created_at,updated_at,menu_level,parent_id,path,name,hidden,component,sort,active_name,keep_alive,default_menu,title,icon,close_tab)
SELECT NOW(3),NOW(3),0,@frontend_parent,seed.path,seed.name,0,seed.component,seed.sort,'',0,0,seed.title,'',0
FROM frontend_module_seed seed
WHERE @frontend_parent IS NOT NULL AND @frontend_parent_path IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM sys_base_menus existing WHERE existing.name=seed.name AND existing.deleted_at IS NULL)
  AND NOT EXISTS (SELECT 1 FROM sys_base_menus existing WHERE existing.path=seed.path AND existing.deleted_at IS NULL)
  AND NOT EXISTS (SELECT 1 FROM frontend_existing_paths existing WHERE existing.full_path=LOWER(CONCAT(TRIM(TRAILING '/' FROM @frontend_parent_path),'/',seed.path)));

CREATE TEMPORARY TABLE frontend_granted_menu_ids (id BIGINT NOT NULL PRIMARY KEY);
INSERT IGNORE INTO frontend_granted_menu_ids
SELECT menu.id FROM sys_base_menus menu JOIN frontend_module_seed seed ON menu.name=seed.name AND menu.component=seed.component
WHERE menu.deleted_at IS NULL;
INSERT IGNORE INTO frontend_granted_menu_ids
SELECT id FROM sys_base_menus WHERE name IN ('user','authority') AND deleted_at IS NULL;

-- Also grant all parent directories without depending on their numeric IDs.
INSERT INTO sys_authority_menus (sys_base_menu_id,sys_authority_authority_id)
WITH RECURSIVE ancestors AS (
  SELECT menu.id,menu.parent_id,CAST(menu.id AS CHAR(2048)) AS visited,0 AS depth
  FROM sys_base_menus menu JOIN frontend_granted_menu_ids ids ON ids.id=menu.id WHERE menu.deleted_at IS NULL
  UNION ALL
  SELECT parent.id,parent.parent_id,CONCAT(child.visited,',',parent.id),child.depth+1
  FROM sys_base_menus parent JOIN ancestors child ON child.parent_id=parent.id
  WHERE parent.deleted_at IS NULL AND child.depth<64 AND FIND_IN_SET(parent.id,child.visited)=0
)
SELECT DISTINCT ancestors.id,1 FROM ancestors
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
  AND NOT EXISTS (SELECT 1 FROM sys_authority_menus granted WHERE granted.sys_base_menu_id=ancestors.id AND granted.sys_authority_authority_id=1);

CREATE TEMPORARY TABLE frontend_button_seed (menu_name VARCHAR(191),name VARCHAR(191),description VARCHAR(191))
DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
INSERT INTO frontend_button_seed VALUES
('files','upload','上传文件'),('files','delete','删除文件'),('files','reference','管理文件引用'),
('sessions','revoke','撤销设备会话'),
('organization-departments','create','新增部门'),('organization-departments','update','编辑部门'),('organization-departments','delete','删除部门'),
('organization-positions','create','新增岗位'),('organization-positions','update','编辑岗位'),('organization-positions','delete','删除岗位'),
('user','membership','分配部门岗位'),('authority','dataScope','设置数据权限');

INSERT INTO sys_base_menu_btns (created_at,updated_at,name,`desc`,sys_base_menu_id)
SELECT NOW(3),NOW(3),seed.name,seed.description,menu.id
FROM frontend_button_seed seed JOIN sys_base_menus menu ON menu.name=seed.menu_name AND menu.deleted_at IS NULL
JOIN frontend_granted_menu_ids allowed ON allowed.id=menu.id
WHERE NOT EXISTS (SELECT 1 FROM sys_base_menu_btns existing WHERE existing.sys_base_menu_id=menu.id AND existing.name=seed.name AND existing.deleted_at IS NULL);

INSERT INTO sys_authority_btns (authority_id,sys_menu_id,sys_base_menu_btn_id)
SELECT 1,menu.id,button.id
FROM frontend_button_seed seed JOIN sys_base_menus menu ON menu.name=seed.menu_name AND menu.deleted_at IS NULL
JOIN frontend_granted_menu_ids allowed ON allowed.id=menu.id
JOIN sys_base_menu_btns button ON button.sys_base_menu_id=menu.id AND button.name=seed.name AND button.deleted_at IS NULL
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
  AND NOT EXISTS (SELECT 1 FROM sys_authority_btns granted WHERE granted.authority_id=1 AND granted.sys_menu_id=menu.id AND granted.sys_base_menu_btn_id=button.id);

INSERT INTO sys_audit_logs (event_type,module,action,object,result,params)
SELECT 'operation','menu','seedFrontendModules','20261003_frontend_modules.sql','success','{}'
WHERE NOT EXISTS (SELECT 1 FROM sys_audit_logs WHERE action='seedFrontendModules' AND object='20261003_frontend_modules.sql');
COMMIT;

SELECT seed.name,CASE WHEN menu.id IS NOT NULL AND menu.component=seed.component THEN 'READY'
  WHEN menu.id IS NOT NULL THEN 'SKIPPED: existing customized component preserved'
  ELSE 'SKIPPED: parent missing or route/name conflict preserved' END AS migration_status
FROM frontend_module_seed seed LEFT JOIN sys_base_menus menu ON menu.name=seed.name AND menu.deleted_at IS NULL ORDER BY seed.sort;
DROP TEMPORARY TABLE frontend_module_seed;
DROP TEMPORARY TABLE frontend_existing_paths;
DROP TEMPORARY TABLE frontend_granted_menu_ids;
DROP TEMPORARY TABLE frontend_button_seed;
