package captchalogic

import (
	"context"
	"fmt"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/mojocn/base64Captcha"
	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateImageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenerateImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateImageLogic {
	return &GenerateImageLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// GenerateImage renders a fresh digit captcha and stores the answer in Redis
// under the canonical key. base64Captcha.DefaultMemStore is touched only to
// keep its in-memory cache in sync; Redis is the single source of truth for
// verification.
func (l *GenerateImageLogic) GenerateImage(_ *pb.NoDataResponse) (*pb.GenerateImageResponse, error) {
	driver := base64Captcha.NewDriverDigit(captchaImgHeight, captchaImgWidth, captchaImgLength, 0.1, 10)
	store := base64Captcha.DefaultMemStore
	id, image, answer, err := base64Captcha.NewCaptcha(driver, store).Generate()
	if err != nil {
		return nil, err
	}
	store.Get(id, true)
	if err := l.svcCtx.BizRedis.SetexCtx(l.ctx, fmt.Sprintf(prefixCaptcha, id), answer, expireCaptcha); err != nil {
		return nil, err
	}
	return &pb.GenerateImageResponse{Id: id, Image: image}, nil
}
