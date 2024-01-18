package base

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"math/rand"
	"net/smtp"
	"os"
	"strconv"
	"time"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result/xerr"

	"github.com/jordan-wright/email"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	prefixEmailCode = "biz#emailcode#email:%s"
	expireEmailCode = 60 * 5 // 5分钟过期
)

var MailPassword = os.Getenv("MailPassword")

type SendEmailCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendEmailCodeLogic) SendEmailCode(req *types.SendEmailCodeRequest) (resp *types.MessageResponse, err error) {
	if req.Email == "" {
		return nil, xerr.NewErrCode(300003)
	}
	cacheCode, err := l.GetEmailCodeCache(req.Email, l.svcCtx.BizRedis)
	if err != nil {
		return nil, err
	}
	if !req.IsForce && cacheCode != "" {
		return nil, xerr.NewErrCode(300004)
	}

	code := l.GetCode()
	err = l.saveEmailCodeCache(req.Email, code, l.svcCtx.BizRedis)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(900000, "保存email到redis 错误")
	}
	err = l.SendCode(req.Email, code)
	if err != nil {
		return nil, errors.New("验证码发送失败！ err:" + err.Error())
	}

	return &types.MessageResponse{Message: "验证码发送成功!"}, nil
}

func (l *SendEmailCodeLogic) SendCode(mail, code string) (err error) {
	e := email.NewEmail()
	e.From = "验证码 <go__zero@163.com>" //发送者 自己
	e.To = []string{mail}             //接收者
	e.Subject = "验证码发送--<请勿泄露>"       //发送主题
	e.HTML = []byte("验证码为：<h1>" + code + "</h1>")
	err = e.SendWithTLS("smtp.163.com:465", smtp.PlainAuth("", "go__zero@163.com", MailPassword, "smtp.163.com"),
		&tls.Config{InsecureSkipVerify: true, ServerName: "smtp.163.com"})
	if err != nil {
		return err
	}
	return nil
}

func (l *SendEmailCodeLogic) GetCode() string {
	rand.Seed(time.Now().UnixNano())
	res := ""
	for i := 0; i < 6; i++ {
		res += strconv.Itoa(rand.Intn(10))
	}
	return res
}

func (l *SendEmailCodeLogic) saveEmailCodeCache(email, code string, rds *redis.Redis) error {
	key := fmt.Sprintf(prefixEmailCode, email)
	return rds.Setex(key, code, expireEmailCode)
}

func (l *SendEmailCodeLogic) GetEmailCodeCache(email string, rds *redis.Redis) (string, error) {
	ket := fmt.Sprintf(prefixEmailCode, email)
	return rds.Get(ket)
}
