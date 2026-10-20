package avatarutil_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/logic/avatarutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/objectstorage"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var storage = objectstorage.Config{ExternalEndpoint: "http://127.0.0.1:9000", PublicBucket: "gozero-public", PrivateBucket: "gozero-private"}

func fixture(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if err := db.AutoMigrate(&model.SysFileResource{}, &model.SysFileReference{}); err != nil {
		t.Fatal(err)
	}
	for id := int64(1); id <= 4; id++ {
		file := model.SysFileResource{ID: id, ObjectKey: key(id), Name: "avatar.png", Mime: "image/png", Size: 100, Visibility: "public", Status: "active"}
		if id == 3 {
			file.Visibility = "private"
		}
		if id == 4 {
			file.Status = "deleting"
		}
		if err := db.Create(&file).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}
func key(id int64) string { return fmt.Sprintf("go-zero-admin/10000000-0000-4000-8000-%012d.png", id) }
func avatar(id int64) string {
	return storage.ExternalEndpoint + "/" + storage.PublicBucket + "/" + key(id)
}
func sync(t *testing.T, db *gorm.DB, kind string, id int64, previous, next string) {
	t.Helper()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return avatarutil.Sync(context.Background(), tx, storage, kind, id, previous, next)
	}); err != nil {
		t.Fatal(err)
	}
}
func count(t *testing.T, db *gorm.DB, id, want int64) {
	t.Helper()
	var got int64
	if err := db.Model(&model.SysFileReference{}).Where("file_id = ?", id).Count(&got).Error; err != nil || got != want {
		t.Fatalf("file %d references=%d want=%d err=%v", id, got, want, err)
	}
}

func TestAvatarReplacementSeparatesUserDomainsAndKeepsSharedFiles(t *testing.T) {
	db := fixture(t)
	sync(t, db, avatarutil.SysUser, 7, "", avatar(1))
	sync(t, db, avatarutil.AppUser, 7, "", avatar(1))
	count(t, db, 1, 2)
	sync(t, db, avatarutil.SysUser, 7, avatar(1), avatar(1))
	count(t, db, 1, 2)
	sync(t, db, avatarutil.SysUser, 7, avatar(1), avatar(2))
	count(t, db, 1, 1)
	count(t, db, 2, 1)
	sync(t, db, avatarutil.AppUser, 7, avatar(1), "")
	count(t, db, 1, 0)
	sync(t, db, avatarutil.SysUser, 7, avatar(2), "/default-avatar.svg")
	count(t, db, 2, 0)
	var objects int64
	db.Model(&model.SysFileResource{}).Where("status = ?", "active").Count(&objects)
	if objects != 3 {
		t.Fatal("replacing an avatar must not delete its file")
	}
}

func TestAvatarRejectsUnknownPrivateDeletingAndWrongBucketFiles(t *testing.T) {
	for _, next := range []string{avatar(3), avatar(4), avatar(99), storage.ExternalEndpoint + "/" + storage.PrivateBucket + "/" + key(1), storage.ExternalEndpoint + "/not-an-image", storage.ExternalEndpoint + "/gozero-public/go-zero-admin/not-a-key.png"} {
		t.Run(fmt.Sprint(len(next))+next[len(next)-5:], func(t *testing.T) {
			db := fixture(t)
			sync(t, db, avatarutil.SysUser, 7, "", avatar(1))
			err := db.Transaction(func(tx *gorm.DB) error {
				return avatarutil.Sync(context.Background(), tx, storage, avatarutil.SysUser, 7, avatar(1), next)
			})
			if err == nil {
				t.Fatal("invalid avatar accepted")
			}
			count(t, db, 1, 1)
		})
	}
}

func TestAvatarAliasesAreTrackedAndTransactionRollbackKeepsOldReference(t *testing.T) {
	db := fixture(t)
	sync(t, db, avatarutil.SysUser, 7, "", "http://localhost:9000/gozero-public/"+key(1)+"?v=2")
	count(t, db, 1, 1)
	sentinel := errors.New("user write failed")
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := avatarutil.Sync(context.Background(), tx, storage, avatarutil.SysUser, 7, avatar(1), avatar(2)); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	count(t, db, 1, 1)
	count(t, db, 2, 0)
	sync(t, db, avatarutil.SysUser, 7, avatar(1), "https://images.example.test/avatar.png")
	count(t, db, 1, 0)
}

func TestBrowserURLSpellingsCannotBypassManagedAvatarReferences(t *testing.T) {
	for name, prefix := range map[string]string{"scheme-relative": "//127.0.0.1:9000/gozero-public/", "root-relative": "/gozero-public/", "backslashes": `http:\\127.0.0.1:9000\gozero-public\`, "uppercase-scheme": "HTTP://127.0.0.1:9000/gozero-public/"} {
		t.Run(name, func(t *testing.T) {
			db := fixture(t)
			sync(t, db, avatarutil.SysUser, 7, "", prefix+key(1))
			count(t, db, 1, 1)
		})
	}
}
