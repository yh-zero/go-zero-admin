package appaccountservicelogic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/appauth"
)

type registrationLocation struct{ input string }

func (r *registrationLocation) Lookup(ip string) string {
	r.input = ip
	return "中国|广东省|深圳市"
}

func TestRegistrationPersistsServerIPAndRegionWithOptionalNickname(t *testing.T) {
	s, actor := accountFixture(t, true)
	resolver := &registrationLocation{}
	s.IPLocation = resolver
	ctx := context.Background()
	account := fmt.Sprintf("registration-ip-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = s.BizRedis.DelCtx(context.Background(), "app:credential:register:account:"+appauth.Digest(account), "app:credential:register:ip:"+appauth.Digest("::ffff:203.0.113.91"))
	})
	result, err := register(ctx, s, &pb.AppRegisterRequest{Account: account, Password: "Password123!", IP: "::ffff:203.0.113.91"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Nickname != "App用户" || result.RegistrationIP != "203.0.113.91" || result.RegistrationRegion != "中国|广东省|深圳市" || resolver.input != result.RegistrationIP {
		t.Fatalf("registration metadata not preserved: %+v", result)
	}
	var stored model.AppUser
	if err := s.DB.First(&stored, result.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.RegistrationIP != result.RegistrationIP || stored.RegistrationRegion != result.RegistrationRegion {
		t.Fatal("IP metadata not saved")
	}
	list, err := listUsers(ctx, s, &pb.AppAdminListRequest{Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.List) != 1 || list.List[0].RegistrationIP != result.RegistrationIP || list.List[0].RegistrationRegion != result.RegistrationRegion {
		t.Fatal("admin cannot see registration metadata")
	}
}
