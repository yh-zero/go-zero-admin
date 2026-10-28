package appmembershipservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

const (
	plansPath     = "/v1/sys/app-members/plans"
	grantsPath    = "/v1/sys/app-members/grants"
	grantPath     = "/v1/sys/app-members/grant"
	profileExport = "profile.export"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,63}$`)

func membershipError(message string) error {
	return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, message)
}
func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func membershipNow() time.Time     { return time.Now().UTC().Truncate(time.Millisecond) }

func validateReason(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > 500 || strings.ContainsAny(value, "\x00\r\n") {
		return "", membershipError("操作原因须为1至500个字符")
	}
	return value, nil
}

func entitlements(snapshot string) ([]string, error) {
	var values []string
	if err := json.Unmarshal([]byte(snapshot), &values); err != nil || values == nil {
		return nil, errors.New("invalid membership entitlement snapshot")
	}
	return values, nil
}

func grantStatus(g *model.AppMembershipGrant, now time.Time) string {
	if g.RevokedAt != nil {
		return "revoked"
	}
	if !g.EndsAt.After(now) {
		return "expired"
	}
	if g.StartsAt.After(now) {
		return "scheduled"
	}
	return "active"
}

func grantProto(g *model.AppMembershipGrant, account string, now time.Time) (*pb.AppGrantInfo, error) {
	rights, err := entitlements(g.Entitlements)
	if err != nil {
		return nil, err
	}
	result := &pb.AppGrantInfo{ID: g.ID, UserID: g.UserID, Account: account, PlanID: g.PlanID, PlanCode: g.PlanCode, PlanVersion: g.PlanVersion, Entitlements: rights, StartsAt: timestamp(g.StartsAt), EndsAt: timestamp(g.EndsAt), Reason: g.Reason, RevokeReason: g.RevokeReason, CreatedAt: timestamp(g.CreatedAt), Status: grantStatus(g, now)}
	if g.RevokedAt != nil {
		result.RevokedAt = timestamp(*g.RevokedAt)
	}
	return result, nil
}

func userAccount(db *gorm.DB, userID int64) (string, error) {
	var identity model.AppUserIdentity
	if err := db.Select("identifier").Where("user_id = ?", userID).First(&identity).Error; err != nil {
		return "", err
	}
	return identity.Identifier, nil
}

// Active rights are derived from grant snapshots at request time. Scheduled
// grants may extend a continuous expiry, but never contribute rights early.
func membershipAt(db *gorm.DB, userID int64, now time.Time) (*pb.AppMembershipInfo, error) {
	var grants []model.AppMembershipGrant
	if err := db.Where("user_id = ? AND revoked_at IS NULL AND ends_at > ?", userID, now).Order("starts_at ASC, ends_at ASC").Find(&grants).Error; err != nil {
		return nil, err
	}
	result := &pb.AppMembershipInfo{Entitlements: []string{}}
	var end time.Time
	rights := map[string]bool{}
	for i := range grants {
		g := &grants[i]
		if g.StartsAt.After(now) {
			continue
		}
		values, err := entitlements(g.Entitlements)
		if err != nil {
			return nil, err
		}
		result.Active = true
		if g.EndsAt.After(end) {
			end = g.EndsAt
		}
		for _, value := range values {
			rights[value] = true
		}
	}
	if !result.Active {
		return result, nil
	}
	for i := range grants {
		g := &grants[i]
		if !g.StartsAt.After(end) && g.EndsAt.After(end) {
			end = g.EndsAt
		}
	}
	for value := range rights {
		result.Entitlements = append(result.Entitlements, value)
	}
	sort.Strings(result.Entitlements)
	result.ExpiresAt = timestamp(end)
	return result, nil
}

func auditMembership(ctx context.Context, tx *gorm.DB, actor *pb.SessionRequest, action, path, method string, g *model.AppMembershipGrant) error {
	var admin model.SysUser
	if err := tx.Select("username").First(&admin, "id = ?", actor.UserID).Error; err != nil {
		return err
	}
	return audit.Record(ctx, tx, audit.Event{ActorID: actor.UserID, ActorName: admin.Username, AuthorityID: actor.AuthorityId, Module: "app-memberships", Action: action, Object: fmt.Sprintf("app-grant:%d", g.ID), Path: path, Method: method, Params: fmt.Sprintf(`{"id":%d,"userId":%d}`, g.ID, g.UserID)})
}
