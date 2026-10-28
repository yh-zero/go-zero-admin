package db_test

import (
	"strings"
	"testing"
)

func TestUserAvatarMigrationBackfillsAndPreservesBothDomains(t *testing.T) {
	before, _, found := strings.Cut(appSQL(t, "gozero-admin.sql"), "-- Automatic user avatar references; run with API/RPC stopped.")
	if !found {
		t.Fatal("missing avatar migration in fresh SQL")
	}
	db, name := appMySQL(t, before)
	seed := `
INSERT INTO sys_file_resources(id,object_key,name,mime,size,owner_id,department_id,visibility,status,created_at,updated_at) VALUES
 (901,'go-zero-admin/10000000-0000-4000-8000-000000000901.png','public.png','image/png',100,1,0,'public','active',NOW(3),NOW(3)),
 (902,'go-zero-admin/10000000-0000-4000-8000-000000000902.png','private.png','image/png',100,1,0,'private','active',NOW(3),NOW(3));
INSERT INTO sys_users(id,username,password,header_img) VALUES
 (901,'avatar-901','unused','http://127.0.0.1:9000/gozero-public/go-zero-admin/10000000-0000-4000-8000-000000000901.png'),
 (902,'avatar-902','unused','http://localhost:9000/gozero-public/go-zero-admin/10000000-0000-4000-8000-000000000901.png?v=2'),
 (903,'avatar-903','unused','https://external.example.test/avatar.png'),
 (904,'avatar-904','unused','http://127.0.0.1:9000/gozero-private/go-zero-admin/10000000-0000-4000-8000-000000000902.png'),
 (905,'avatar-905','unused',REPLACE('http://127.0.0.1:9000/gozero-public/go-zero-admin/10000000-0000-4000-8000-000000000901.png','/',CHAR(92))),
 (906,'avatar-906','unused','https://external.example.test/static/go-zero-admin/10000000-0000-4000-8000-000000000901.png'),
 (907,'avatar-907','unused','http://127.0.0.1:9000/gozero-private/go-zero-admin/10000000-0000-4000-8000-000000000901.png');
INSERT INTO app_users(id,nickname,status,created_at,updated_at) VALUES(901,'app avatar','disabled',NOW(3),NOW(3));
`
	if _, err := db.Exec(seed); err != nil {
		t.Fatal(err)
	}
	migration := appSQL(t, "migrations/20261020_user_avatar_references.sql")
	if _, err := db.Exec(migration); err != nil {
		t.Fatal(err)
	}
	appCount(t, db, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=? AND table_name='app_users' AND column_name='avatar_url' AND character_maximum_length=1024 AND is_nullable='NO' AND column_default=''", 1, name)
	appCount(t, db, "SELECT COUNT(*) FROM sys_file_references WHERE object_type='sys_user_avatar'", 3)
	appCount(t, db, "SELECT COUNT(*) FROM sys_file_references WHERE file_id=902", 0)
	if _, err := db.Exec("UPDATE app_users SET avatar_url='http://127.0.0.1:9000/gozero-public/go-zero-admin/10000000-0000-4000-8000-000000000901.png' WHERE id=901"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(migration); err != nil {
			t.Fatal(err)
		}
	}
	appCount(t, db, "SELECT COUNT(*) FROM sys_file_references WHERE file_id=901", 4)
	appCount(t, db, "SELECT COUNT(*) FROM sys_file_references WHERE object_type='app_user_avatar' AND object_id='901'", 1)
	appCount(t, db, "SELECT COUNT(*) FROM sys_base_menu_btns WHERE permission_key='app-members-users:avatar'", 1)
	appCount(t, db, "SELECT COUNT(*) FROM sys_apis WHERE path='/v1/sys/app-members/avatar' AND method='PUT' AND deleted_at IS NULL", 1)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v1='/v1/sys/app-members/avatar'", 0)
	bootstrap := appSQL(t, "operator/20261020_avatar_admin_bootstrap.sql")
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(bootstrap); err != nil {
			t.Fatal(err)
		}
	}
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1='/v1/sys/app-members/avatar' AND v2='PUT'", 1)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0<>'1' AND v1='/v1/sys/app-members/avatar'", 0)
	appCount(t, db, "SELECT COUNT(*) FROM sys_users WHERE id BETWEEN 901 AND 904", 4)
	appCount(t, db, "SELECT COUNT(*) FROM app_users WHERE id=901 AND status='disabled'", 1)
}
