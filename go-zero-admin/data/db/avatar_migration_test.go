package db_test

import "testing"

func TestRustFSAvatarMigrationIsIdempotentAndPreservesCustomAvatars(t *testing.T) {
	db, name := appMySQL(t, `
CREATE TABLE sys_users (
  id BIGINT NOT NULL PRIMARY KEY,
  header_img VARCHAR(1024) NOT NULL DEFAULT 'https://legacy.oss-cn-beijing.aliyuncs.com/go-zero-admin/avatar.jpg'
);
INSERT INTO sys_users(id,header_img) VALUES
 (1,'https://legacy.oss-cn-beijing.aliyuncs.com/go-zero-admin/avatar.jpg'),
 (2,'https://custom.example.com/avatar.png'),
 (3,'/uploads/custom-avatar.png'),
 (4,'/default-avatar.svg');
`)
	migration := appSQL(t, "migrations/20261020_rustfs_avatar.sql")
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(migration); err != nil {
			t.Fatalf("avatar migration attempt %d failed: %v", i+1, err)
		}
	}
	appCount(t, db, "SELECT COUNT(*) FROM sys_users", 4)
	for id, expected := range map[int]string{
		1: "/default-avatar.svg",
		2: "https://custom.example.com/avatar.png",
		3: "/uploads/custom-avatar.png",
		4: "/default-avatar.svg",
	} {
		var avatar string
		if err := db.QueryRow("SELECT header_img FROM sys_users WHERE id=?", id).Scan(&avatar); err != nil {
			t.Fatal(err)
		}
		if avatar != expected {
			t.Fatalf("user %d avatar=%q, want %q", id, avatar, expected)
		}
	}
	appCount(t, db, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=? AND table_name='sys_users' AND column_name='header_img' AND column_default='/default-avatar.svg'", 1, name)
	if _, err := db.Exec("INSERT INTO sys_users(id) VALUES(5)"); err != nil {
		t.Fatal(err)
	}
	appCount(t, db, "SELECT COUNT(*) FROM sys_users WHERE id=5 AND header_img='/default-avatar.svg'", 1)
}
