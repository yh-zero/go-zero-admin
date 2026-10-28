package fileresourceservicelogic

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/objectstorage"
	"gorm.io/gorm"
)

func attachTestStorage(t *testing.T, service *svc.ServiceContext, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := objectstorage.New(objectstorage.Config{Endpoint: server.URL, ExternalEndpoint: "http://127.0.0.1:9000", Region: "us-east-1", AccessKeyID: "test-access", SecretAccessKey: "test-secret", PublicBucket: "gozero-public", PrivateBucket: "gozero-private"})
	if err != nil {
		t.Fatal(err)
	}
	service.Storage = client
}

func uploadInput(actor *pb.SessionRequest, visibility string) *pb.UploadFileRequest {
	return &pb.UploadFileRequest{Actor: actor, Filename: "test.png", Visibility: visibility, Data: []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 50))}
}

func TestRustFSUploadAndURLsRetainMetadataAuditAndScope(t *testing.T) {
	service, member, admin := fileDB(t)
	var writes atomic.Int32
	attachTestStorage(t, service, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || (!strings.HasPrefix(r.URL.Path, "/gozero-public/go-zero-admin/") && !strings.HasPrefix(r.URL.Path, "/gozero-private/go-zero-admin/")) {
			t.Errorf("unexpected upload request %s %s", r.Method, r.URL.Path)
		}
		writes.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	ctx := context.Background()
	for _, visibility := range []string{"public", "private"} {
		response, err := NewUploadFileLogic(ctx, service).UploadFile(uploadInput(member, visibility))
		if err != nil {
			t.Fatal(err)
		}
		if response.File.ID <= 0 || response.File.OwnerID != member.UserID || response.File.DepartmentId != 2 || response.File.Status != "active" || response.File.Visibility != visibility {
			t.Fatalf("wrong registered file: %+v", response.File)
		}
		if !strings.HasPrefix(response.Url, "http://127.0.0.1:9000/gozero-"+visibility+"/go-zero-admin/") || strings.Contains(response.Url, "test-secret") {
			t.Fatal("wrong upload URL or leaked secret")
		}
		wantExpiry := int64(0)
		if visibility == "private" {
			wantExpiry = 300
		}
		if response.ExpiresIn != wantExpiry {
			t.Fatalf("wrong upload expiry: %d", response.ExpiresIn)
		}
		urlResponse, err := NewGetFileURLLogic(ctx, service).GetFileURL(&pb.FileIDRequest{Actor: member, ID: response.File.ID})
		if err != nil || urlResponse.ExpiresIn != wantExpiry || !strings.HasPrefix(urlResponse.Url, "http://127.0.0.1:9000/gozero-"+visibility+"/") {
			t.Fatalf("wrong access URL: %+v %v", urlResponse, err)
		}
	}
	if writes.Load() != 2 {
		t.Fatalf("unexpected upload count %d", writes.Load())
	}
	var count int64
	if err := service.DB.Model(&model.SysAuditLog{}).Where("module = 'files' AND action = 'registerFile'").Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("missing registration audit: %d %v", count, err)
	}
	other, err := NewRegisterFileLogic(ctx, service).RegisterFile(fileInput(admin, "3"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewGetFileURLLogic(ctx, service).GetFileURL(&pb.FileIDRequest{Actor: member, ID: other.ID}); err == nil {
		t.Fatal("signed another owner's file")
	}
}

func TestRustFSUploadFailureDoesNotRegisterAndAuditFailureCompensates(t *testing.T) {
	for _, failUpload := range []bool{true, false} {
		t.Run(map[bool]string{true: "storage failure", false: "audit rollback cleanup"}[failUpload], func(t *testing.T) {
			service, member, _ := fileDB(t)
			var uploads, deletes atomic.Int32
			attachTestStorage(t, service, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "PUT" {
					uploads.Add(1)
					if failUpload {
						w.WriteHeader(http.StatusForbidden)
						_, _ = io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>test-secret</Message></Error>`)
						return
					}
				} else if r.Method == "DELETE" {
					deletes.Add(1)
				} else {
					t.Errorf("unexpected operation %s", r.Method)
				}
				w.WriteHeader(http.StatusOK)
			})
			if !failUpload {
				if err := service.DB.Exec("CREATE TRIGGER reject_upload_audit BEFORE INSERT ON sys_audit_logs WHEN NEW.module = 'files' BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END").Error; err != nil {
					t.Fatal(err)
				}
			}
			if _, err := NewUploadFileLogic(context.Background(), service).UploadFile(uploadInput(member, "private")); err == nil || strings.Contains(err.Error(), "test-secret") {
				t.Fatalf("failed upload must be safe error: %v", err)
			}
			var count int64
			if err := service.DB.Model(&model.SysFileResource{}).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("failed upload registered metadata: %d %v", count, err)
			}
			wantDeletes := int32(0)
			if !failUpload {
				wantDeletes = 1
			}
			if uploads.Load() != 1 || deletes.Load() != wantDeletes {
				t.Fatalf("wrong compensation uploads=%d deletes=%d", uploads.Load(), deletes.Load())
			}
		})
	}
}

func TestRustFSDeleteFailureRetainsRetryStateAndReferencesBlockStorageCall(t *testing.T) {
	service, member, admin := fileDB(t)
	var deletes atomic.Int32
	var fail atomic.Bool
	fail.Store(true)
	attachTestStorage(t, service, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || !strings.HasPrefix(r.URL.Path, "/gozero-private/") {
			t.Errorf("unexpected delete %s %s", r.Method, r.URL.Path)
		}
		deletes.Add(1)
		if fail.Load() {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	ctx := context.Background()
	file, err := NewRegisterFileLogic(ctx, service).RegisterFile(fileInput(member, "1"))
	if err != nil {
		t.Fatal(err)
	}
	reference := &pb.FileReferenceRequest{Actor: admin, FileID: file.ID, ObjectType: "article", ObjectID: "7"}
	if _, err := NewAddFileReferenceLogic(ctx, service).AddFileReference(reference); err != nil {
		t.Fatal(err)
	}
	request := &pb.FileIDRequest{Actor: member, ID: file.ID}
	if _, err := NewDeleteFileLogic(ctx, service).DeleteFile(request); err == nil || deletes.Load() != 0 {
		t.Fatal("referenced object reached storage delete")
	}
	if _, err := NewRemoveFileReferenceLogic(ctx, service).RemoveFileReference(reference); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDeleteFileLogic(ctx, service).DeleteFile(request); err == nil {
		t.Fatal("storage delete failure reported success")
	}
	var row model.SysFileResource
	if err := service.DB.First(&row, file.ID).Error; err != nil || row.Status != "deleting" {
		t.Fatalf("lost retry state %+v %v", row, err)
	}
	if _, err := NewGetFileURLLogic(ctx, service).GetFileURL(request); err == nil {
		t.Fatal("deleting file still issued an access URL")
	}
	fail.Store(false)
	for i := 0; i < 2; i++ {
		if _, err := NewDeleteFileLogic(ctx, service).DeleteFile(request); err != nil {
			t.Fatal(err)
		}
	}
	if deletes.Load() != 2 {
		t.Fatalf("finished retry repeated storage deletion: %d", deletes.Load())
	}
	if err := service.DB.First(&row, file.ID).Error; err != nil || row.Status != "deleted" {
		t.Fatalf("delete not finished %+v %v", row, err)
	}
}

func TestRustFSUnconfiguredStorageAndInvalidUploadsFailWithoutWrites(t *testing.T) {
	service, member, _ := fileDB(t)
	ctx := context.Background()
	if _, err := NewUploadFileLogic(ctx, service).UploadFile(uploadInput(member, "public")); err == nil || !strings.Contains(err.Error(), "未配置") {
		t.Fatalf("missing configuration not reported: %v", err)
	}
	file, err := NewRegisterFileLogic(ctx, service).RegisterFile(fileInput(member, "1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Model(&model.SysFileResource{}).Where("id = ?", file.ID).Update("visibility", "public").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewGetFileURLLogic(ctx, service).GetFileURL(&pb.FileIDRequest{Actor: member, ID: file.ID}); err == nil || !strings.Contains(err.Error(), "未配置") {
		t.Fatalf("public URL accepted unconfigured storage: %v", err)
	}
	var requests atomic.Int32
	attachTestStorage(t, service, func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(http.StatusOK) })
	for _, mutate := range []func(*pb.UploadFileRequest){
		func(in *pb.UploadFileRequest) { in.Data = make([]byte, (10<<20)+1) },
		func(in *pb.UploadFileRequest) { in.Data = []byte("<script>unsafe</script>") },
		func(in *pb.UploadFileRequest) { in.Visibility = "unknown" },
		func(in *pb.UploadFileRequest) { in.Mime = "image/jpeg" },
		func(in *pb.UploadFileRequest) { in.Actor.SessionVersion = 99 },
	} {
		input := uploadInput(&pb.SessionRequest{UserID: member.UserID, AuthorityId: member.AuthorityId, SessionVersion: 1}, "public")
		mutate(input)
		if _, err := NewUploadFileLogic(ctx, service).UploadFile(input); err == nil {
			t.Fatal("invalid upload accepted")
		}
	}
	if requests.Load() != 0 {
		t.Fatal("invalid upload reached storage")
	}
}

func TestRustFSUploadRegistrationFailureCleansStoredObject(t *testing.T) {
	for _, failure := range []string{"revoked session", "file lookup unavailable", "membership table unavailable"} {
		t.Run(failure, func(t *testing.T) {
			service, member, _ := fileDB(t)
			if failure == "file lookup unavailable" {
				// SQLite cannot grant INSERT while denying SELECT. Inject only the
				// failed lookup at the ORM boundary; transactions and writes stay real.
				if err := service.DB.Callback().Query().Before("gorm:query").Register("fail_file_lookup", func(tx *gorm.DB) {
					if _, ok := tx.Statement.Dest.(*model.SysFileResource); ok {
						tx.AddError(errors.New("file lookup unavailable"))
					}
				}); err != nil {
					t.Fatal(err)
				}
				defer service.DB.Callback().Query().Remove("fail_file_lookup")
			}
			var uploads, deletes atomic.Int32
			var uploadedPath atomic.Value
			attachTestStorage(t, service, func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPut:
					uploads.Add(1)
					uploadedPath.Store(r.URL.Path)
					var err error
					if failure == "revoked session" {
						err = service.DB.Model(&model.SysUser{}).Where("id = ?", member.UserID).Update("session_version", 2).Error
					} else if failure == "membership table unavailable" {
						err = service.DB.Exec("DROP TABLE sys_user_departments").Error
					}
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				case http.MethodDelete:
					deletes.Add(1)
					if r.URL.Path != uploadedPath.Load() {
						t.Errorf("cleanup targeted %q, uploaded %q", r.URL.Path, uploadedPath.Load())
					}
				default:
					t.Errorf("unexpected storage operation %s", r.Method)
				}
				w.WriteHeader(http.StatusOK)
			})
			if _, err := NewUploadFileLogic(context.Background(), service).UploadFile(uploadInput(member, "private")); err == nil {
				t.Fatal("upload registered despite invalid session or failed database lookup")
			}
			var files, audits int64
			if err := service.DB.Model(&model.SysFileResource{}).Count(&files).Error; err != nil {
				t.Fatal(err)
			}
			if err := service.DB.Model(&model.SysAuditLog{}).Count(&audits).Error; err != nil {
				t.Fatal(err)
			}
			if files != 0 || audits != 0 || uploads.Load() != 1 || deletes.Load() != 1 {
				t.Fatalf("failed registration left files=%d audits=%d uploads=%d deletes=%d", files, audits, uploads.Load(), deletes.Load())
			}
		})
	}
}

func TestRustFSUploadResponseFailureKeepsCommittedObject(t *testing.T) {
	service, member, _ := fileDB(t)
	if err := service.DB.Callback().Query().Before("gorm:query").Register("fail_file_response", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_file_references" {
			tx.AddError(errors.New("reference count unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer service.DB.Callback().Query().Remove("fail_file_response")
	var uploads, deletes atomic.Int32
	attachTestStorage(t, service, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			uploads.Add(1)
		} else if r.Method == http.MethodDelete {
			deletes.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	})
	if _, err := NewUploadFileLogic(context.Background(), service).UploadFile(uploadInput(member, "private")); err == nil {
		t.Fatal("response query failure reported success")
	}
	var files, audits int64
	if err := service.DB.Model(&model.SysFileResource{}).Count(&files).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Model(&model.SysAuditLog{}).Count(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if files != 1 || audits != 1 || uploads.Load() != 1 || deletes.Load() != 0 {
		t.Fatalf("response failure destroyed committed upload: files=%d audits=%d uploads=%d deletes=%d", files, audits, uploads.Load(), deletes.Load())
	}
}

func TestRustFSUploadCancellationDoesNotCancelCompensation(t *testing.T) {
	service, member, _ := fileDB(t)
	sqlDB, err := service.DB.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(2)
	// Keep this shared in-memory SQLite database alive if database/sql discards
	// the transaction connection after cancellation.
	anchor, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer anchor.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := service.DB.Callback().Query().Before("gorm:query").Register("cancel_file_registration", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*model.SysFileResource); ok {
			cancel()
			tx.AddError(context.Canceled)
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer service.DB.Callback().Query().Remove("cancel_file_registration")
	var uploads, deletes atomic.Int32
	attachTestStorage(t, service, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			uploads.Add(1)
		} else if r.Method == http.MethodDelete {
			deletes.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	})
	if _, err := NewUploadFileLogic(ctx, service).UploadFile(uploadInput(member, "private")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	var files, audits int64
	if err := service.DB.Model(&model.SysFileResource{}).Count(&files).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Model(&model.SysAuditLog{}).Count(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if files != 0 || audits != 0 || uploads.Load() != 1 || deletes.Load() != 1 {
		t.Fatalf("canceled registration cleanup failed: files=%d audits=%d uploads=%d deletes=%d", files, audits, uploads.Load(), deletes.Load())
	}
}
