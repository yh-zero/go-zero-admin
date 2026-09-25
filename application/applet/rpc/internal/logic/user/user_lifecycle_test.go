package userlogic

import (
	"context"
	"fmt"
	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/orm"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
)

func testUserDB(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&model.SysAuthority{}, &model.SysUser{}, &model.SysUserAuthority{}, &model.SysAuthorityMenu{}, &model.SysBaseMenu{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{801, 802} {
		if err := db.Create(&model.SysAuthority{AuthorityId: id, AuthorityName: fmt.Sprint(id)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return &svc.ServiceContext{DB: &orm.DB{DB: db}}
}
func TestUserLifecycleAndAtomicRoles(t *testing.T) {
	svcCtx := testUserDB(t)
	ctx := context.Background()
	register := NewRegisterLogic(ctx, svcCtx)
	input := &pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: "test_account", Password: "test123456", NickName: "SearchAlias", AuthorityId: 801, Enable: 1, Phone: "123", Email: "before@example.test"}, AuthorityIds: []int64{801, 802}}
	if _, err := register.Register(input); err != nil {
		t.Fatal(err)
	}
	if _, err := register.Register(input); err == nil {
		t.Fatal("duplicate username accepted")
	}
	var user model.SysUser
	if err := svcCtx.DB.Where("username = ?", "test_account").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	update := NewUpdateUserInfoLogic(ctx, svcCtx)
	// Removing the default role is rejected and no profile change is committed.
	if _, err := update.UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: user.ID, NickName: "should rollback"}, UpdateFields: []string{"nickName"}, UpdateAuthorities: true, AuthorityIds: []int64{802}}); err == nil {
		t.Fatal("invalid default role accepted")
	}
	var after model.SysUser
	if err := svcCtx.DB.First(&after, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.NickName != "SearchAlias" {
		t.Fatal("profile changed despite rejected roles")
	}
	// 模拟资料写入失败：先替换角色的操作也必须回滚。
	if err := svcCtx.DB.Exec("CREATE TRIGGER reject_user_update BEFORE UPDATE OF nick_name ON sys_users WHEN NEW.nick_name = 'reject' BEGIN SELECT RAISE(FAIL, 'rejected update'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := update.UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: user.ID, NickName: "reject", AuthorityId: 802}, UpdateFields: []string{"nickName", "authorityId"}, UpdateAuthorities: true, AuthorityIds: []int64{802}}); err == nil {
		t.Fatal("expected database failure")
	}
	var preservedRoles int64
	if err := svcCtx.DB.Model(&model.SysUserAuthority{}).Where("sys_user_id = ?", user.ID).Count(&preservedRoles).Error; err != nil || preservedRoles != 2 {
		t.Fatal("role replacement did not roll back")
	}
	// Explicit empty fields are saved atomically with the new default and role set.
	if _, err := update.UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: user.ID, Phone: "", Email: "", AuthorityId: 802}, UpdateFields: []string{"phone", "email", "authorityId"}, UpdateAuthorities: true, AuthorityIds: []int64{802}}); err != nil {
		t.Fatal(err)
	}
	if err := svcCtx.DB.Preload("Authorities").First(&after, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Phone != "" || after.Email != "" || after.NickName != "SearchAlias" || after.AuthorityId != 802 || len(after.Authorities) != 1 {
		t.Fatalf("unexpected stored user: %+v", after)
	}
	list := NewGetUserListLogic(ctx, svcCtx)
	filtered, err := list.GetUserList(&pb.GetUserListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10, Keyword: "SearchAlias"}})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 1 || filtered.UserInfoList[0].Password != "" {
		t.Fatal("search or password filtering failed")
	}
	empty, err := list.GetUserList(&pb.GetUserListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10, Keyword: "missing"}})
	if err != nil || empty.Total != 0 {
		t.Fatal("keyword must filter")
	}
	login, err := NewGetUserInfoLogic(ctx, svcCtx).GetUserInfo(&pb.GetUserInfoRequest{UserName: "test_account", Password: "test123456"})
	if err != nil || login.UserInfo.AuthorityId != 802 || login.UserInfo.Password != "" {
		t.Fatal("active user login or default role invalid", err)
	}
	if _, err := update.UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: user.ID, Enable: 2}, UpdateFields: []string{"enable"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewGetUserInfoLogic(ctx, svcCtx).GetUserInfo(&pb.GetUserInfoRequest{UserName: "test_account", Password: "test123456"}); err == nil {
		t.Fatal("frozen user logged in")
	}
	if _, err := NewDeleteUserLogic(ctx, svcCtx).DeleteUser(&pb.DeleteUserRequest{UserID: user.ID}); err != nil {
		t.Fatal(err)
	}
	var relationships int64
	if err := svcCtx.DB.Model(&model.SysUserAuthority{}).Where("sys_user_id = ?", user.ID).Count(&relationships).Error; err != nil || relationships != 0 {
		t.Fatal("user role links left behind")
	}
	if _, err := NewResetUserPasswordLogic(ctx, svcCtx).ResetUserPassword(&pb.ResetUserPasswordRequest{UserID: user.ID}); err == nil {
		t.Fatal("nonexistent user reset reported success")
	}
}
