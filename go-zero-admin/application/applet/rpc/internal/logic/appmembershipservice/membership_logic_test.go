package appmembershipservicelogic

import (
	"context"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

// Missing RPC authentication must never expose membership or accept an admin mutation.
func TestMembershipRPCRejectsMissingCredentials(t *testing.T) {
	s := &svc.ServiceContext{}
	ctx := context.Background()
	tests := []struct {
		name string
		call func() error
	}{
		{"membership", func() error {
			_, err := NewGetAppMembershipLogic(ctx, s).GetAppMembership(&pb.AppAccessRequest{})
			return err
		}},
		{"export", func() error {
			_, err := NewGetAppMemberExampleLogic(ctx, s).GetAppMemberExample(&pb.AppExampleRequest{})
			return err
		}},
		{"plans", func() error { _, err := NewListAppPlansLogic(ctx, s).ListAppPlans(&pb.AppAdminRequest{}); return err }},
		{"grants", func() error {
			_, err := NewListAppGrantsLogic(ctx, s).ListAppGrants(&pb.AppGrantListRequest{})
			return err
		}},
		{"grant", func() error {
			_, err := NewGrantAppMembershipLogic(ctx, s).GrantAppMembership(&pb.AppGrantRequest{})
			return err
		}},
		{"revoke", func() error {
			_, err := NewRevokeAppMembershipLogic(ctx, s).RevokeAppMembership(&pb.AppGrantRevokeRequest{})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Fatal("unauthenticated RPC accepted")
			}
		})
	}
}
