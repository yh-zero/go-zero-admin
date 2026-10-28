-- Replace legacy object-storage avatar addresses with the application's bundled avatar.
-- Existing files and user accounts are retained. Safe to retry before recording the checksum.
ALTER TABLE sys_users ALTER COLUMN header_img SET DEFAULT '/default-avatar.svg';
UPDATE sys_users
SET header_img='/default-avatar.svg'
WHERE header_img LIKE '%oss%.aliyuncs.com/%';
