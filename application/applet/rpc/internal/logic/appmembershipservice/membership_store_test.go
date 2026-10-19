package appmembershipservicelogic

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	cm "github.com/casbin/casbin/v2/model"
	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/appauth"
	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func membershipFixture(t *testing.T, db *gorm.DB) (*svc.ServiceContext, *pb.SessionRequest, string) {
	t.Helper()
	if db == nil {
		var err error
		db, err = gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		sqlDB.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	if err := db.AutoMigrate(&model.AppUser{}, &model.AppUserIdentity{}, &model.AppSession{}, &model.AppMembershipPlan{}, &model.AppMembershipGrant{}, &model.SysUser{}, &model.SysAuthority{}, &model.SysUserAuthority{}, &model.SysDeviceSession{}, &model.SysAuditLog{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{
		&model.SysAuthority{AuthorityId: 1, AuthorityName: "admin"},
		&model.SysUser{MODEL_BASE: base.MODEL_BASE{ID: 30}, Username: "admin", AuthorityId: 1, Enable: 1, SessionVersion: 1},
		&model.SysUserAuthority{SysUserId: 30, SysAuthorityAuthorityId: 1},
		&model.SysDeviceSession{ID: "admin-device", UserID: 30, AuthorityID: 1, SessionVersion: 1, ExpiresAt: time.Now().Add(time.Hour)},
		&model.AppUser{ID: 7, Nickname: "vip member", Status: "active", SessionVersion: 1},
		&model.AppUser{ID: 8, Nickname: "other member", Status: "active", SessionVersion: 1},
		&model.AppUserIdentity{UserID: 7, Kind: "email", Identifier: "member@example.com", PasswordHash: "private hash"},
		&model.AppUserIdentity{UserID: 8, Kind: "email", Identifier: "other@example.com", PasswordHash: "other private hash"},
		&model.AppSession{ID: "app-session", UserID: 7, SessionVersion: 1, RefreshHash: "private refresh hash", ExpiresAt: time.Now().Add(time.Hour)},
		&model.AppMembershipPlan{ID: 1, Code: "vip", Name: "VIP", Version: 1, Entitlements: `["profile.export"]`, Enabled: true},
	} {
		if err := db.Omit("Authority", "Authorities").Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	m, err := cm.NewModelFromString("[request_definition]\nr = sub, obj, act\n[policy_definition]\np = sub, obj, act\n[policy_effect]\ne = some(where (p.eft == allow))\n[matchers]\nm = r.sub == p.sub && r.obj == p.obj && r.act == p.act")
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewSyncedCachedEnforcer(m)
	if err != nil {
		t.Fatal(err)
	}
	enforcer.EnableCache(false)
	for _, p := range [][]string{{"1", "/v1/sys/app-members/plans", "GET"}, {"1", "/v1/sys/app-members/grants", "GET"}, {"1", "/v1/sys/app-members/grants", "POST"}, {"1", "/v1/sys/app-members/grant", "DELETE"}} {
		if _, err := enforcer.AddPolicy(p); err != nil {
			t.Fatal(err)
		}
	}
	s := &svc.ServiceContext{DB: &orm.DB{DB: db}, Casbin: enforcer}
	s.Config.AppJwtAuth = appauth.Config{AccessSecret: "app-membership-test-01234567890123456789", AccessExpire: 900, RefreshExpire: 2592000}
	token, err := appauth.Issue(s.Config.AppJwtAuth, 7, "app-session", 1, 1, "access", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return s, &pb.SessionRequest{UserID: 30, AuthorityId: 1, SessionVersion: 1, SessionID: "admin-device"}, token
}

func grantInput(actor *pb.SessionRequest, key string) *pb.AppGrantRequest {
	return &pb.AppGrantRequest{Actor: actor, UserID: 7, PlanID: 1, Days: 30, RequestID: key, Reason: "人工开通"}
}

// Catches lost renewal time, request replays extending twice, and mutable plan-derived rights.
func TestGrantRenewalReplayAndSnapshot(t *testing.T) {
	s, actor, token := membershipFixture(t, nil)
	ctx := context.Background()
	grant := NewGrantAppMembershipLogic(ctx, s)
	first, err := grant.GrantAppMembership(grantInput(actor, "first-request"))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == 0 || first.Status != "active" || first.Account != "member@example.com" {
		t.Fatalf("incomplete grant: %+v", first)
	}
	start, _ := time.Parse(time.RFC3339Nano, first.StartsAt)
	end, _ := time.Parse(time.RFC3339Nano, first.EndsAt)
	if end.Sub(start) != 30*24*time.Hour {
		t.Fatal("grant duration is not 30 days")
	}
	replay, err := grant.GrantAppMembership(grantInput(actor, "first-request"))
	if err != nil || replay.ID != first.ID || replay.EndsAt != first.EndsAt {
		t.Fatalf("retry extended membership: %+v %v", replay, err)
	}
	changed := grantInput(actor, "first-request")
	changed.Days = 31
	if _, err = grant.GrantAppMembership(changed); err == nil {
		t.Fatal("request ID accepted different payload")
	}
	second, err := grant.GrantAppMembership(grantInput(actor, "second-request"))
	if err != nil {
		t.Fatal(err)
	}
	if second.StartsAt != first.EndsAt || second.Status != "scheduled" {
		t.Fatalf("renewal overlaps active grant: %+v", second)
	}
	if err = s.DB.Model(&model.AppMembershipPlan{}).Where("id = 1").Update("entitlements", `[]`).Error; err != nil {
		t.Fatal(err)
	}
	current, err := NewGetAppMembershipLogic(ctx, s).GetAppMembership(&pb.AppAccessRequest{AccessToken: token})
	if err != nil || !current.Active || len(current.Entitlements) != 1 || current.Entitlements[0] != "profile.export" || current.ExpiresAt != second.EndsAt {
		t.Fatalf("snapshot/renewed expiry lost: %+v %v", current, err)
	}
	var logs []model.SysAuditLog
	if err = s.DB.Order("id").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 || logs[0].ActorID != 30 || !strings.HasPrefix(logs[0].Object, "app-grant:") {
		t.Fatalf("wrong transaction audit: %+v", logs)
	}
}

// Catches future/expired/revoked grants granting immediate access and exporting another user's record.
func TestMembershipTimeWindowsAndOwnProfileOnly(t *testing.T) {
	for _, state := range []string{"none", "future", "expired", "revoked", "active"} {
		t.Run(state, func(t *testing.T) {
			s, _, token := membershipFixture(t, nil)
			now := time.Now().UTC()
			if state != "none" {
				g := model.AppMembershipGrant{UserID: 7, PlanID: 1, PlanCode: "vip", PlanVersion: 1, Entitlements: `["profile.export"]`, RequestID: "fixture", Days: 1, StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour), GrantedBy: 30, Reason: "fixture"}
				if state == "future" {
					g.StartsAt = now.Add(time.Hour)
					g.EndsAt = now.Add(2 * time.Hour)
				}
				if state == "expired" {
					g.EndsAt = now.Add(-time.Second)
				}
				if state == "revoked" {
					g.RevokedAt = &now
				}
				if err := s.DB.Create(&g).Error; err != nil {
					t.Fatal(err)
				}
			}
			info, err := NewGetAppMembershipLogic(context.Background(), s).GetAppMembership(&pb.AppAccessRequest{AccessToken: token})
			if err != nil || info.Active != (state == "active") {
				t.Fatalf("wrong active state %+v %v", info, err)
			}
			export := NewGetAppMemberExampleLogic(context.Background(), s)
			own, err := export.GetAppMemberExample(&pb.AppExampleRequest{AccessToken: token, UserID: 7})
			if state == "active" {
				if err != nil || own.UserID != 7 || own.Account != "member@example.com" || own.Nickname != "vip member" {
					t.Fatalf("own export invalid %+v %v", own, err)
				}
			} else if err == nil {
				t.Fatal("inactive membership permitted export")
			}
			if _, err = export.GetAppMemberExample(&pb.AppExampleRequest{AccessToken: token, UserID: 8}); err == nil {
				t.Fatal("cross-user export permitted")
			}
		})
	}
}

func TestGrantRevokeOnlyTargetAndAuditRollback(t *testing.T) {
	s, actor, token := membershipFixture(t, nil)
	ctx := context.Background()
	first, err := NewGrantAppMembershipLogic(ctx, s).GrantAppMembership(grantInput(actor, "first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewGrantAppMembershipLogic(ctx, s).GrantAppMembership(grantInput(actor, "second"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DB.Exec("CREATE TRIGGER reject_membership_audit BEFORE INSERT ON sys_audit_logs BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END").Error; err != nil {
		t.Fatal(err)
	}
	revoke := &pb.AppGrantRevokeRequest{Actor: actor, ID: first.ID, Reason: "退回人工开通"}
	if _, err = NewRevokeAppMembershipLogic(ctx, s).RevokeAppMembership(revoke); err == nil {
		t.Fatal("revocation survived audit failure")
	}
	if _, err = NewGrantAppMembershipLogic(ctx, s).GrantAppMembership(grantInput(actor, "audit-failed")); err == nil {
		t.Fatal("grant survived audit failure")
	}
	var row model.AppMembershipGrant
	if err = s.DB.First(&row, first.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.RevokedAt != nil {
		t.Fatal("failed audit left grant revoked")
	}
	var count int64
	s.DB.Model(&model.AppMembershipGrant{}).Count(&count)
	if count != 2 {
		t.Fatal("failed audit left a grant")
	}
	if err = s.DB.Exec("DROP TRIGGER reject_membership_audit").Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = NewRevokeAppMembershipLogic(ctx, s).RevokeAppMembership(revoke); err != nil {
			t.Fatal(err)
		}
	}
	var later model.AppMembershipGrant
	if err = s.DB.First(&later, second.ID).Error; err != nil {
		t.Fatal(err)
	}
	if later.RevokedAt != nil || later.StartsAt.UTC().Format(time.RFC3339Nano) != second.StartsAt {
		t.Fatal("revocation changed a later grant")
	}
	info, err := NewGetAppMembershipLogic(ctx, s).GetAppMembership(&pb.AppAccessRequest{AccessToken: token})
	if err != nil || info.Active {
		t.Fatalf("scheduled renewal became active after revoke: %+v %v", info, err)
	}
	s.DB.Model(&model.SysAuditLog{}).Count(&count)
	if count != 3 {
		t.Fatalf("expected one revoke audit, got %d total", count)
	}
}

func TestGrantValidationAuthorizationAndList(t *testing.T) {
	s, actor, _ := membershipFixture(t, nil)
	ctx := context.Background()
	grant := NewGrantAppMembershipLogic(ctx, s)
	for _, days := range []int64{0, -1, 3651} {
		input := grantInput(actor, "bad-days")
		input.Days = days
		if _, err := grant.GrantAppMembership(input); err == nil {
			t.Fatal("invalid grant duration accepted")
		}
	}
	for _, status := range []string{"pending", "disabled", "rejected"} {
		s.DB.Model(&model.AppUser{}).Where("id = 7").Update("status", status)
		if _, err := grant.GrantAppMembership(grantInput(actor, "bad-state")); err == nil {
			t.Fatalf("granted %s account", status)
		}
	}
	s.DB.Model(&model.AppUser{}).Where("id = 7").Update("status", "active")
	if _, err := s.Casbin.RemovePolicy("1", "/v1/sys/app-members/grants", "POST"); err != nil {
		t.Fatal(err)
	}
	if _, err := grant.GrantAppMembership(grantInput(actor, "forbidden")); err == nil {
		t.Fatal("grant without permission accepted")
	}
	if _, err := s.Casbin.AddPolicy("1", "/v1/sys/app-members/grants", "POST"); err != nil {
		t.Fatal(err)
	}
	if _, err := grant.GrantAppMembership(grantInput(actor, "valid")); err != nil {
		t.Fatal(err)
	}
	plans, err := NewListAppPlansLogic(ctx, s).ListAppPlans(&pb.AppAdminRequest{Actor: actor})
	if err != nil || len(plans.List) != 1 || plans.List[0].Code != "vip" {
		t.Fatalf("plan list %+v %v", plans, err)
	}
	list, err := NewListAppGrantsLogic(ctx, s).ListAppGrants(&pb.AppGrantListRequest{Actor: actor, Page: &pb.PageRequest{PageNo: 1, PageSize: 1}, UserID: 7})
	if err != nil || list.Total != 1 || len(list.List) != 1 || list.List[0].Account != "member@example.com" {
		t.Fatalf("grant list %+v %v", list, err)
	}
	list, err = NewListAppGrantsLogic(ctx, s).ListAppGrants(&pb.AppGrantListRequest{Actor: actor, Page: &pb.PageRequest{PageNo: 1, PageSize: 1}, UserID: 8})
	if err != nil || list.Total != 0 {
		t.Fatalf("user filter %+v %v", list, err)
	}
}
