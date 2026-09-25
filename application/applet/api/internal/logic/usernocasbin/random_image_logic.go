package usernocasbin

import (
	"context"
	"fmt"
	"github.com/mojocn/base64Captcha"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result/xerr"
	"regexp"
)

const (
	prefixCaptcha    = "biz#captcha#id:%s"
	expireCaptcha    = 120
	captchaImgWidth  = 105
	captchaImgHeight = 36
	captchaImgLength = 6
)

var captchaIDPattern = regexp.MustCompile("^[A-Za-z0-9_-]{10,128}$")

type RandomImageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRandomImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RandomImageLogic {
	return &RandomImageLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}
func (l *RandomImageLogic) RandomImage(req *types.RandomImageRequest) (*types.RandomImageResponse, error) {
	driver := base64Captcha.NewDriverDigit(captchaImgHeight, captchaImgWidth, captchaImgLength, 0.1, 10)
	store := base64Captcha.DefaultMemStore
	id, image, answer, err := base64Captcha.NewCaptcha(driver, store).Generate()
	if err != nil {
		return nil, err
	}
	store.Get(id, true) // Redis 是唯一校验存储，避免重复保留答案。
	if err := l.svcCtx.BizRedis.SetexCtx(l.ctx, fmt.Sprintf(prefixCaptcha, id), answer, expireCaptcha); err != nil {
		return nil, err
	}
	return &types.RandomImageResponse{CaptchaId: id, CaptchaImg: image}, nil
}

// 每张图片仅允许一次提交，成功或失败后均需刷新；多客户端不会覆盖彼此的验证码。
func consumeCaptcha(ctx context.Context, id, answer string, rds *redis.Redis) error {
	if !captchaIDPattern.MatchString(id) || len(answer) != captchaImgLength {
		return xerr.NewErrCode(xerr.CAPTCHA_ERROR)
	}
	result, err := rds.EvalCtx(ctx, "local value = redis.call('GET', KEYS[1]); redis.call('DEL', KEYS[1]); return value", []string{fmt.Sprintf(prefixCaptcha, id)})
	if err != nil {
		return err
	}
	expected, ok := result.(string)
	if !ok || expected == "" || expected != answer {
		return xerr.NewErrCode(xerr.CAPTCHA_ERROR)
	}
	return nil
}
