package fileservicelogic

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/model"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/objectstorage"
	"go-zero-admin/pkg/xerr"
)

type cleanupTestFiles struct {
	model.SysFileModel
	file          model.SysFile
	marks         int
	confirmations int
	failReceipt   bool
}

func (f *cleanupTestFiles) MarkDeletedIfUnused(_ context.Context, id, actorID int64) (*model.SysFile, error) {
	if id != int64(f.file.Id) || actorID != f.file.UploaderId.Int64 {
		return nil, model.ErrFileForbidden
	}
	f.marks++
	f.file.DeletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	return &f.file, nil
}

func (f *cleanupTestFiles) ConfirmObjectCleanup(_ context.Context, id int64) error {
	f.confirmations++
	if id != int64(f.file.Id) || !f.file.DeletedAt.Valid {
		return errors.New("缺少对应文件的软删除记录")
	}
	if f.failReceipt {
		return errors.New("模拟回执写入失败")
	}
	f.file.ObjectCleanedAt = sql.NullTime{Time: time.Now(), Valid: true}
	return nil
}

func TestDeleteFileRetriesUntilCleanupConfirmed(t *testing.T) {
	for _, tc := range []struct {
		name            string
		failStorage     bool
		wantCode        uint32
		initialReceipts int
	}{
		{"对象清理失败", true, xerr.SERVER_COMMON_ERROR, 0},
		{"回执写入失败", false, xerr.DB_ERROR, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var deletes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := deletes.Add(1)
				if r.Method != http.MethodDelete || r.URL.Path != "/archive-private/docs/sample.txt" {
					t.Errorf("应删除原有私有对象，得到 %s %s", r.Method, r.URL.Path)
				}
				if tc.failStorage && n == 1 {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(server.Close)
			store, err := objectstorage.New(objectstorage.Config{
				Endpoint: server.URL, ExternalEndpoint: server.URL, Region: "us-east-1",
				AccessKeyID: "test", SecretAccessKey: "test", PublicBucket: "public-test", PrivateBucket: "private-test",
			})
			if err != nil {
				t.Fatal(err)
			}
			files := &cleanupTestFiles{
				file:        model.SysFile{Id: 3, UploaderId: sql.NullInt64{Int64: 7, Valid: true}, Bucket: "archive-private", ObjectKey: "docs/sample.txt", Visibility: "private"},
				failReceipt: !tc.failStorage,
			}
			logic := NewDeleteFileLogic(t.Context(), &svc.ServiceContext{SysFileModel: files, Store: store})
			request := &filepb.DeleteFileRequest{FileID: 3, Actor: &filepb.Actor{UserID: 7}}
			resp, err := logic.DeleteFile(request)
			code, ok := xerr.AsCodeError(err)
			if resp != nil || !ok || code.GetErrCode() != tc.wantCode {
				t.Fatalf("首次删除应报告 %d，得到 %v", tc.wantCode, err)
			}
			if deletes.Load() != 1 || !files.file.DeletedAt.Valid || files.file.ObjectCleanedAt.Valid || files.confirmations != tc.initialReceipts {
				t.Fatalf("失败后应保留待清理状态，已删除=%t、已确认=%t、存储请求=%d、回执写入次数=%d",
					files.file.DeletedAt.Valid, files.file.ObjectCleanedAt.Valid, deletes.Load(), files.confirmations)
			}
			files.failReceipt = false
			if _, err := logic.DeleteFile(request); err != nil || !files.file.ObjectCleanedAt.Valid {
				t.Fatalf("重试应完成并记回执: %v", err)
			}
			if _, err := logic.DeleteFile(request); err != nil {
				t.Fatalf("已确认的删除再次提交应成功: %v", err)
			}
			if deletes.Load() != 2 || files.confirmations != tc.initialReceipts+1 || files.marks != 3 {
				t.Fatalf("已确认后重复提交只应核对原记录，实际存储请求=%d、回执写入次数=%d、归属检查=%d",
					deletes.Load(), files.confirmations, files.marks)
			}
		})
	}
}

func TestDeleteFileInvalidRecordedBucketKeepsPending(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("无效的历史桶名不应触发存储请求")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	store, err := objectstorage.New(objectstorage.Config{
		Endpoint: server.URL, ExternalEndpoint: server.URL, Region: "us-east-1",
		AccessKeyID: "test", SecretAccessKey: "test", PublicBucket: "public-test", PrivateBucket: "private-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range []string{"", "../private-test", "127.0.0.1"} {
		files := &cleanupTestFiles{file: model.SysFile{
			Id: 3, Bucket: bucket, ObjectKey: "docs/sample.txt", Visibility: "private", UploaderId: sql.NullInt64{Int64: 7, Valid: true},
		}}
		logic := NewDeleteFileLogic(t.Context(), &svc.ServiceContext{SysFileModel: files, Store: store})
		_, err := logic.DeleteFile(&filepb.DeleteFileRequest{FileID: 3, Actor: &filepb.Actor{UserID: 7}})
		if err == nil || !files.file.DeletedAt.Valid || files.file.ObjectCleanedAt.Valid || files.confirmations != 0 {
			t.Fatalf("无效历史桶 %q 不应确认清理", bucket)
		}
	}
}
