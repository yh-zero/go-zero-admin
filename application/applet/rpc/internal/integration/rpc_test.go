package integration

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/model"
	auditserver "go-zero-admin/application/applet/rpc/internal/server/audit"
	fileserver "go-zero-admin/application/applet/rpc/internal/server/fileresourceservice"
	organizationserver "go-zero-admin/application/applet/rpc/internal/server/organization"
	sessionserver "go-zero-admin/application/applet/rpc/internal/server/sessionmanage"
	userserver "go-zero-admin/application/applet/rpc/internal/server/user"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	adminDevice  = "00000000-0000-4000-8000-000000000030"
	memberDevice = "00000000-0000-4000-8000-000000000010"
	otherDevice  = "00000000-0000-4000-8000-000000000020"
)

type rpcFixture struct {
	db                            *gorm.DB
	conn                          *grpc.ClientConn
	admin, member                 *pb.SessionRequest
	cancelStarted, cancelFinished chan struct{}
}

func fixture(t *testing.T) *rpcFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.SysAuthority{}, &model.SysUser{}, &model.SysUserAuthority{}, &model.SysDeviceSession{}, &model.SysDepartment{}, &model.SysPosition{}, &model.SysUserDepartment{}, &model.SysUserPosition{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}, &model.SysFileResource{}, &model.SysFileReference{}, &audit.Event{}); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SysPermissionVersion{}, &model.SysPermissionChange{}, &model.SysBaseMenu{}, &model.SysBaseMenuBtn{}, &model.SysAuthorityMenu{}, &model.SysAuthorityBtn{}, &model.SysApi{}, &model.SysBaseMenuParameter{}, &gormadapter.CasbinRule{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SysPermissionVersion{ID: 1, Revision: 1}).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{1, 2} {
		if err := db.Create(&model.SysAuthority{AuthorityId: id, AuthorityName: "role"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range []struct {
		id, role int64
		device   string
	}{{30, 1, adminDevice}, {10, 2, memberDevice}, {20, 2, otherDevice}} {
		user := model.SysUser{MODEL_BASE: base.MODEL_BASE{ID: entry.id}, Username: "user-" + entry.device, AuthorityId: entry.role, Enable: 1, SessionVersion: 1}
		if err := db.Omit("Authority", "Authorities").Create(&user).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.SysUserAuthority{SysUserId: entry.id, SysAuthorityAuthorityId: entry.role}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.SysDeviceSession{ID: entry.device, UserID: entry.id, AuthorityID: entry.role, SessionVersion: 1, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour), IP: "127.0.0.1", UserAgent: "integration"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	f := &rpcFixture{db: db, admin: &pb.SessionRequest{UserID: 30, AuthorityId: 1, SessionVersion: 1, SessionID: adminDevice}, member: &pb.SessionRequest{UserID: 10, AuthorityId: 2, SessionVersion: 1, SessionID: memberDevice}, cancelStarted: make(chan struct{}), cancelFinished: make(chan struct{})}
	service := &svc.ServiceContext{DB: &orm.DB{DB: db}}
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if in, ok := req.(*pb.DepartmentRequest); ok && in.Department != nil && in.Department.Code == "cancel_me" {
			close(f.cancelStarted)
			<-ctx.Done()
			defer close(f.cancelFinished)
		}
		return handler(ctx, req)
	}))
	pb.RegisterOrganizationServer(server, organizationserver.NewOrganizationServer(service))
	pb.RegisterAuditServer(server, auditserver.NewAuditServer(service))
	pb.RegisterSessionManageServer(server, sessionserver.NewSessionManageServer(service))
	pb.RegisterUserServer(server, userserver.NewUserServer(service))
	pb.RegisterFileResourceServiceServer(server, fileserver.NewFileResourceServiceServer(service))
	go func() { _ = server.Serve(listener) }()
	f.conn, err = grpc.NewClient("passthrough:///bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }), grpc.WithUnaryInterceptor(audit.ClientInterceptor()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.conn.Close(); server.Stop(); _ = listener.Close(); _ = sqlDB.Close() })
	return f
}
func operatorContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ctx = audit.WithRequest(ctx, audit.Request{Path: "/v1/sys/organization/departments", Method: "POST", IP: "127.0.0.1", TraceID: "integration-trace"})
	audit.SetActor(ctx, audit.Actor{ID: 30, Name: "管理员", AuthorityID: 1})
	return ctx, cancel
}

func TestOrganizationAuditRPCSerializationAndMetadata(t *testing.T) {
	f := fixture(t)
	ctx, cancel := operatorContext()
	defer cancel()
	org := pb.NewOrganizationClient(f.conn)
	root, err := org.CreateDepartment(ctx, &pb.DepartmentRequest{Department: &pb.Department{Name: "总部", Code: "headquarters", Status: 1}})
	if err != nil || root.ID <= 0 || root.Name != "总部" {
		t.Fatal("CreateDepartment RPC failed", root, err)
	}
	child, err := org.CreateDepartment(ctx, &pb.DepartmentRequest{Department: &pb.Department{ParentId: root.ID, Name: "研发部", Code: "engineering", Status: 1}})
	if err != nil {
		t.Fatal(err)
	}
	position, err := org.CreatePosition(ctx, &pb.PositionRequest{Position: &pb.Position{Name: "开发工程师", Code: "engineer", Status: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := org.UpdateMembership(ctx, &pb.MembershipRequest{UserID: 10, DepartmentId: child.ID, PositionIds: []int64{position.ID}}); err != nil {
		t.Fatal(err)
	}
	membership, err := org.GetMembership(ctx, &pb.OrganizationIDRequest{ID: 10})
	if err != nil || membership.DepartmentId != child.ID || len(membership.PositionIds) != 1 || membership.PositionIds[0] != position.ID {
		t.Fatal("membership protobuf mapping failed", membership, err)
	}
	tree, err := org.GetDepartmentTree(ctx, &pb.OrganizationListRequest{Keyword: "研发"})
	if err != nil || len(tree.List) != 1 || len(tree.List[0].Children) != 1 || tree.List[0].Children[0].ID != child.ID {
		t.Fatal("nested protobuf tree failed", tree, err)
	}
	initialScope, err := org.GetRoleDataScope(ctx, &pb.GetRoleDataScopeRequest{AuthorityId: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := org.UpdateRoleDataScope(ctx, &pb.RoleDataScopeRequest{DataScope: &pb.RoleDataScope{ExpectedRevision: initialScope.DataScope.Revision, AuthorityId: 2, Scope: "custom", DepartmentIds: []int64{child.ID}}}); err != nil {
		t.Fatal(err)
	}
	scope, err := org.GetRoleDataScope(ctx, &pb.GetRoleDataScopeRequest{AuthorityId: 2})
	if err != nil || scope.DataScope.Scope != "custom" || len(scope.DataScope.DepartmentIds) != 1 {
		t.Fatal("data-scope protobuf mapping failed", scope, err)
	}
	auditClient := pb.NewAuditClient(f.conn)
	logs, err := auditClient.GetAuditLogList(ctx, &pb.GetAuditLogListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}, Module: "organization"})
	if err != nil || logs.Total != 5 {
		t.Fatal("organization audit RPC failed", logs, err)
	}
	for _, event := range logs.List {
		if event.ActorID != 30 || event.ActorName != "管理员" || event.TraceID != "integration-trace" || event.CreatedAt == "" {
			t.Fatal("trusted outgoing metadata or event serialization failed", event)
		}
	}
	if _, err := auditClient.RecordAudit(ctx, &pb.RecordAuditRequest{Log: &pb.AuditLog{Module: "integration", Action: "sanitize", Params: `{"userId":10,"password":"secret","token":"private"}`}}); err != nil {
		t.Fatal(err)
	}
	logs, err = auditClient.GetAuditLogList(ctx, &pb.GetAuditLogListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}, Module: "integration"})
	if err != nil || logs.Total != 1 || strings.Contains(logs.List[0].Params, "secret") || strings.Contains(logs.List[0].Params, "private") {
		t.Fatal("audit RPC leaked credentials", logs, err)
	}
	valid, err := pb.NewUserClient(f.conn).CheckSession(ctx, f.member)
	if err != nil || valid.Valid {
		t.Fatal("membership update failed to revoke the prior session over RPC", valid, err)
	}
}

func TestDeviceSessionRPCPersonalAndAdminPermissions(t *testing.T) {
	f := fixture(t)
	ctx, cancel := operatorContext()
	defer cancel()
	sessions := pb.NewSessionManageClient(f.conn)
	page := &pb.PageRequest{PageNo: 1, PageSize: 10}
	self, err := sessions.GetDeviceSessions(ctx, &pb.DeviceSessionListRequest{Actor: f.member, PageRequest: page})
	if err != nil || self.Total != 1 || len(self.List) != 1 || !self.List[0].Current || self.List[0].ID != memberDevice {
		t.Fatal("personal session list escaped ownership", self, err)
	}
	all, err := sessions.GetDeviceSessions(ctx, &pb.DeviceSessionListRequest{Actor: f.admin, PageRequest: page})
	if err != nil || all.Total != 3 {
		t.Fatal("admin session list failed", all, err)
	}
	if _, err := sessions.GetDeviceSessions(ctx, &pb.DeviceSessionListRequest{Actor: f.member, UserID: 20, PageRequest: page}); err == nil {
		t.Fatal("member queried another user's sessions")
	}
	if _, err := sessions.RevokeDeviceSession(ctx, &pb.RevokeDeviceSessionRequest{Actor: f.member, ID: otherDevice}); err == nil {
		t.Fatal("member revoked another user's session")
	}
	if _, err := sessions.RevokeDeviceSession(ctx, &pb.RevokeDeviceSessionRequest{Actor: f.admin, ID: otherDevice, SelfOnly: true}); err == nil {
		t.Fatal("admin personal revoke escaped SelfOnly")
	}
	if _, err := sessions.RevokeDeviceSession(ctx, &pb.RevokeDeviceSessionRequest{Actor: f.admin, ID: otherDevice}); err != nil {
		t.Fatal(err)
	}
	valid, err := pb.NewUserClient(f.conn).CheckSession(ctx, &pb.SessionRequest{UserID: 20, AuthorityId: 2, SessionVersion: 1, SessionID: otherDevice})
	if err != nil || valid.Valid {
		t.Fatal("revoked device remains valid", valid, err)
	}
	valid, err = pb.NewUserClient(f.conn).CheckSession(ctx, f.member)
	if err != nil || !valid.Valid {
		t.Fatal("target revoke invalidated a different user device", valid, err)
	}
}

func TestRPCCancellationReachesTransactionAndAllNewServices(t *testing.T) {
	f := fixture(t)
	ctx, cancel := operatorContext()
	defer cancel()
	canceled, stop := context.WithCancel(ctx)
	org := pb.NewOrganizationClient(f.conn)
	done := make(chan error, 1)
	go func() {
		_, err := org.CreateDepartment(canceled, &pb.DepartmentRequest{Department: &pb.Department{Name: "Canceled", Code: "cancel_me", Status: 1}})
		done <- err
	}()
	select {
	case <-f.cancelStarted:
	case <-ctx.Done():
		t.Fatal("RPC did not reach the server")
	}
	stop()
	if err := <-done; status.Code(err) != codes.Canceled {
		t.Fatal("RPC did not propagate cancellation", err)
	}
	select {
	case <-f.cancelFinished:
	case <-ctx.Done():
		t.Fatal("server handler ignored canceled context")
	}
	var count int64
	if err := f.db.Model(&model.SysDepartment{}).Where("code = ?", "cancel_me").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("canceled RPC committed a department", count, err)
	}
	dead, cancelDead := context.WithCancel(ctx)
	cancelDead()
	if _, err := org.GetDepartmentTree(dead, &pb.OrganizationListRequest{}); status.Code(err) != codes.Canceled {
		t.Fatal("organization cancellation failed", err)
	}
	if _, err := pb.NewAuditClient(f.conn).GetAuditLogList(dead, &pb.GetAuditLogListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}}); status.Code(err) != codes.Canceled {
		t.Fatal("audit cancellation failed", err)
	}
	if _, err := pb.NewSessionManageClient(f.conn).GetDeviceSessions(dead, &pb.DeviceSessionListRequest{Actor: f.admin, PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}}); status.Code(err) != codes.Canceled {
		t.Fatal("device-session cancellation failed", err)
	}
}

func TestFileRPCOwnershipScopeReferencesAndDeviceRevocation(t *testing.T) {
	f := fixture(t)
	ctx, cancel := operatorContext()
	defer cancel()
	files := pb.NewFileResourceServiceClient(f.conn)
	if err := f.db.Create(&model.SysDepartment{MODEL_BASE: base.MODEL_BASE{ID: 1}, Name: "root", Code: "root", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Create(&model.SysUserDepartment{UserID: 10, DepartmentID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	file, err := files.RegisterFile(ctx, &pb.RegisterFileRequest{Actor: f.member, File: &pb.FileResource{ObjectKey: "go-zero-admin/00000000-0000-4000-8000-000000000041.png", Name: "头像.png", Mime: "image/png", Size: 128, Visibility: "private", OwnerID: 30, DepartmentId: 999}})
	if err != nil || file.OwnerID != 10 || file.DepartmentId != 1 || file.Status != "active" || file.CreatedAt == "" {
		t.Fatal("file RPC trusted caller ownership or lost metadata", file, err)
	}
	other := &pb.SessionRequest{UserID: 20, AuthorityId: 2, SessionVersion: 1, SessionID: otherDevice}
	foreign, err := files.RegisterFile(ctx, &pb.RegisterFileRequest{Actor: other, File: &pb.FileResource{ObjectKey: "go-zero-admin/00000000-0000-4000-8000-000000000042.png", Name: "other.png", Mime: "image/png", Size: 128, Visibility: "private"}})
	if err != nil {
		t.Fatal(err)
	}
	list, err := files.GetFileList(ctx, &pb.FileListRequest{Actor: f.member, PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}})
	if err != nil || list.Total != 1 || list.List[0].ID != file.ID {
		t.Fatal("file RPC scope escaped owner", list, err)
	}
	if _, err := files.GetFile(ctx, &pb.FileIDRequest{Actor: f.member, ID: foreign.ID}); err == nil {
		t.Fatal("foreign file read over RPC")
	}
	ref := &pb.FileReferenceRequest{Actor: f.admin, FileID: file.ID, ObjectType: "profile", ObjectID: "10"}
	if _, err := files.AddFileReference(ctx, ref); err != nil {
		t.Fatal(err)
	}
	if _, err := files.BeginDeleteFile(ctx, &pb.FileIDRequest{Actor: f.member, ID: file.ID}); err == nil {
		t.Fatal("referenced file began deletion")
	}
	if _, err := files.RemoveFileReference(ctx, ref); err != nil {
		t.Fatal(err)
	}
	deleting, err := files.BeginDeleteFile(ctx, &pb.FileIDRequest{Actor: f.member, ID: file.ID})
	if err != nil || deleting.Status != "deleting" {
		t.Fatal("file delete transition failed", deleting, err)
	}
	if _, err := files.FinishDeleteFile(ctx, &pb.FileIDRequest{Actor: f.member, ID: file.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := files.GetFile(ctx, &pb.FileIDRequest{Actor: f.member, ID: file.ID}); err == nil {
		t.Fatal("deleted file still accessible")
	}
	if err := f.db.Model(&model.SysDeviceSession{}).Where("id = ?", memberDevice).Update("revoked_at", time.Now().UTC()).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := files.GetFileList(ctx, &pb.FileListRequest{Actor: f.member, PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}}); err == nil {
		t.Fatal("revoked device read files over RPC")
	}
}
