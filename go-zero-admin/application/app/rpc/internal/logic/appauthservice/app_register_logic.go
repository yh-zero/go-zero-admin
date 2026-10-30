package appauthservicelogic

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"go-zero-admin/application/app/rpc/internal/model"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/zeromicro/go-zero/core/limit"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type AppRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AppRegisterLogic {
	return &AppRegisterLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *AppRegisterLogic) AppRegister(in *apppb.AppRegisterRequest) (*apppb.NoDataResponse, error) {
	account := strings.ToLower(strings.TrimSpace(in.Account))

	// 限流：IP 维度 10 次/小时 + 账号维度 5 次/小时（go-zero PeriodLimit，Lua 原子）
	limiterIP := limit.NewPeriodLimit(3600, 10, l.svcCtx.Redis, "app:register:ip:")
	if code, err := limiterIP.TakeCtx(l.ctx, in.Ip); err == nil && code == limit.OverQuota {
		return nil, xerr.NewErrCodeMsg(400, "注册过于频繁，请稍后再试")
	}
	limiterAcct := limit.NewPeriodLimit(3600, 5, l.svcCtx.Redis, "app:register:acct:")
	if code, err := limiterAcct.TakeCtx(l.ctx, account); err == nil && code == limit.OverQuota {
		return nil, xerr.NewErrCodeMsg(400, "该账号注册尝试过多，请稍后再试")
	}

	// 密码强度校验：至少 8 位，且同时含字母和数字
	if err := validatePasswordStrength(in.Password); err != nil {
		return nil, err
	}

	kind := "email"
	if !strings.Contains(account, "@") {
		kind = "phone"
	}

	// 唯一校验
	_, err := l.svcCtx.AppUserModel.FindOneByAccount(l.ctx, account)
	if err == nil {
		return nil, xerr.NewErrCodeMsg(400, "账号已存在")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 10)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}

	now := time.Now()
	user := &model.AppUser{
		Account: account, IdentityKind: kind, PasswordHash: string(hash),
		Nickname: in.Nickname, Status: "pending", CreatedAt: now, UpdatedAt: now,
	}
	_, err = l.svcCtx.AppUserModel.Insert(l.ctx, user)
	if err != nil {
		l.Errorf("insert app user error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}

// validatePasswordStrength 密码强度校验：至少 8 位且同时含字母与数字
func validatePasswordStrength(password string) error {
	if len(password) < 8 || len(password) > 64 {
		return xerr.NewErrCodeMsg(400, "密码长度需为 8-64 位")
	}
	var hasLetter, hasDigit bool
	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
			hasLetter = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return xerr.NewErrCodeMsg(400, "密码需同时包含字母和数字")
	}
	return nil
}
