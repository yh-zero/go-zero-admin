package usernocasbin

import (
	"context"
	"fmt"
	"strings"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/mojocn/base64Captcha"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var store = base64Captcha.DefaultMemStore

const (
	prefixCaptcha    = "biz#captcha#ip:%s"
	expireCaptcha    = 60 * 3000 // 2分钟
	captchaImgWidth  = 105
	captchaImgHeight = 36
	captchaImgLength = 6
)

type RandomImageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRandomImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RandomImageLogic {
	return &RandomImageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RandomImageLogic) RandomImage(req *types.RandomImageRequest) (resp *types.RandomImageResponse, err error) {
	remoteAddrIp := l.ctx.Value("RemoteAddr") // 获取id 存redis
	driver := base64Captcha.NewDriverDigit(captchaImgHeight, captchaImgWidth, captchaImgLength, 0.1, 10)

	cp := base64Captcha.NewCaptcha(driver, store)

	//id, b64s, answer, err := cp.Generate()
	_, b64s, answer, err := cp.Generate()

	if err != nil {
		logx.Errorf("验证码获取失败! error: %v", err)
		return
	}

	err = saveActivationCache(GetIP(remoteAddrIp), answer, l.svcCtx.BizRedis)
	if err != nil {
		logx.Errorf("验证码存redis错误 error: %v", err)
		return nil, err
	}
	return &types.RandomImageResponse{CaptchaImg: b64s}, nil
}

func saveActivationCache(keyStr, code string, rds *redis.Redis) error {
	key := fmt.Sprintf(prefixCaptcha, keyStr)
	return rds.Setex(key, code, expireCaptcha)
}

func GetActivationCache(keyStr string, rds *redis.Redis) (string, error) {
	ket := fmt.Sprintf(prefixCaptcha, keyStr)
	return rds.Get(ket)
}

func GetIP(ipStr any) string {
	index := strings.Index(ipStr.(string), ":")
	return ipStr.(string)[:index]
}
