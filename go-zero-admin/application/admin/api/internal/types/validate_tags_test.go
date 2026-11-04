package types_test

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"

	"go-zero-admin/application/admin/api/internal/types"
)

// 校验 .api 来源的 validate 标签已正确生成到类型上，覆盖各规则类别的合法/缺失/畸形用例。
// types.go 为生成物，重新生成丢失标签时本测试应失败。

func newValidator() *validator.Validate { return validator.New() }

func TestValidateTags(t *testing.T) {
	v := newValidator()
	longString65 := strings.Repeat("a", 65)

	cases := []struct {
		name    string
		input   any
		wantErr bool
	}{
		// 合法基线
		{"创建用户-合法", &types.CreateUserReq{Username: "admin", Password: "secret123", Gender: 1, Status: 1, Email: "a@b.com", RoleIds: []int64{1, 2}}, false},
		{"更新用户-全省略合法", &types.UpdateUserReq{Id: 1}, false},
		{"分页-省略合法", &types.GetUserListReq{}, false},
		{"分页-合法", &types.GetUserListReq{Page: 1, PageSize: 20}, false},
		{"可选枚举-零值跳过", &types.CreateUserReq{Username: "ab", Password: "12345678"}, false},

		// ID 约束
		{"路径ID-零拒绝", &types.GetUserReq{Id: 0}, true},
		{"路径ID-正常", &types.GetUserReq{Id: 1}, false},

		// 枚举约束
		{"status-非法值", &types.CreateUserReq{Username: "a", Password: "b", Status: 3}, true},
		{"gender-非法值", &types.CreateUserReq{Username: "a", Password: "b", Gender: 3}, true},
		{"数据范围-非法值", &types.CreateRoleReq{Name: "n", Code: "c", DataScope: "OTHER"}, true},
		{"数据范围-合法", &types.CreateRoleReq{Name: "n", Code: "c", DataScope: "DEPT"}, false},

		// 格式与长度
		{"邮箱-非法", &types.CreateUserReq{Username: "a", Password: "b", Email: "not-an-email"}, true},
		{"用户名-超长", &types.CreateUserReq{Username: longString65, Password: "b"}, true},
		{"密码-bcrypt上限内合法", &types.CreateUserReq{Username: "ab", Password: strings.Repeat("x", 72)}, false},
		{"分页-负数拒绝", &types.GetUserListReq{Page: -1}, true},

		// 切片元素
		{"角色ID切片-含零拒绝", &types.CreateUserReq{Username: "a", Password: "b", RoleIds: []int64{0}}, true},

		// 账号密码策略（契约：用户名 2-64，设置类密码 8-72，旧密码/登录不设最小长度）
		{"创建用户-用户名过短", &types.CreateUserReq{Username: "a", Password: "12345678"}, true},
		{"创建用户-用户名下限合法", &types.CreateUserReq{Username: "ab", Password: "12345678"}, false},
		{"创建用户-密码过短", &types.CreateUserReq{Username: "ab", Password: "1234567"}, true},
		{"重置密码-新密码过短", &types.ResetPasswordReq{Id: 1, NewPassword: "1234567"}, true},
		{"改密码-旧密码不设最小长度", &types.ChangePasswordReq{OldPassword: "x", NewPassword: "12345678"}, false},
		{"分页-page_size超上限", &types.GetUserListReq{PageSize: 101}, true},
		{"分页-page_size上限合法", &types.GetUserListReq{PageSize: 100}, false},

		// App 模块（RPC 规则：注册密码 8-64 位、反馈 2000 字符）
		{"App注册-密码过短", &types.AppRegisterReq{Account: "a@b.com", Password: "short"}, true},
		{"App注册-密码下限合法", &types.AppRegisterReq{Account: "a@b.com", Password: "12345678"}, false},
		{"App反馈-超长", &types.AppSubmitFeedbackReq{Content: strings.Repeat("好", 2001)}, true},
		{"App反馈-上限内合法", &types.AppSubmitFeedbackReq{Content: strings.Repeat("好", 2000)}, false},

		// AI 模块（RPC 规则：消息 16000、拒绝原因 512）
		{"AI消息-合法", &types.AiSendMessageReq{ConversationId: 1, Content: "你好"}, false},
		{"AI消息-会话ID零拒绝", &types.AiSendMessageReq{ConversationId: 0, Content: "你好"}, true},
		{"AI拒绝-原因超长", &types.AiRejectTaskReq{Id: 1, Reason: strings.Repeat("r", 513)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Struct(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("期望校验失败，实际通过: %+v", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("期望校验通过，实际失败: %v", err)
			}
		})
	}
}
