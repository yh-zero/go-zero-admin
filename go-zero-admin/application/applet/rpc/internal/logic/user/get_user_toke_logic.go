package userlogic

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/ctxJwt"
	"go-zero-admin/pkg/result/xerr"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GetUserTokeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserTokeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserTokeLogic {
	return &GetUserTokeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取Token
func (l *GetUserTokeLogic) GetUserToke(in *pb.GetUserTokeRequest) (*pb.GetUserTokeResponse, error) {
	if in == nil {
		return nil, sessionExpired()
	}
	accessExpire := l.svcCtx.Config.JwtAuth.AccessExpire
	if accessExpire <= 0 || accessExpire > 365*24*3600 || l.svcCtx.Config.JwtAuth.AccessSecret == "" {
		return nil, xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "登录有效期配置无效")
	}
	var response pb.GetUserTokeResponse
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		var current model.SysUser
		err := sessionQuery(tx, &pb.SessionRequest{UserID: in.ID, SessionVersion: in.SessionVersion, AuthorityId: in.AuthorityId}).Clauses(clause.Locking{Strength: "UPDATE"}).First(&current).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return sessionExpired()
			}
			return err
		}
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		session := model.SysDeviceSession{ID: id.String(), UserID: current.ID, AuthorityID: current.AuthorityId, SessionVersion: current.SessionVersion, IP: clipMetadata(in.IP, 64), UserAgent: clipMetadata(in.UserAgent, 512), CreatedAt: now, ExpiresAt: now.Add(time.Duration(accessExpire) * time.Second)}
		token, err := l.getJwtToken(l.svcCtx.Config.JwtAuth.AccessSecret, now.Unix(), accessExpire, ctxJwt.JWTData{
			SessionID: session.ID, SessionVersion: current.SessionVersion, UUID: current.UUID.String(), ID: current.ID, NickName: current.NickName, Username: current.Username, AuthorityId: current.AuthorityId,
		})
		if err != nil {
			return err
		}
		if err = tx.Create(&session).Error; err != nil {
			return err
		}
		if err = audit.Record(l.ctx, tx, audit.Event{ActorID: current.ID, ActorName: current.Username, AuthorityID: current.AuthorityId, Module: "session", Action: "createDeviceSession", Object: session.ID}); err != nil {
			return err
		}
		response.Token, response.ExpiresAt = token, session.ExpiresAt.Unix()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func clipMetadata(value string, maximum int) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, value)
	runes := []rune(value)
	if len(runes) > maximum {
		return string(runes[:maximum])
	}
	return value
}

func (l *GetUserTokeLogic) getJwtToken(secretKey string, iat, seconds int64, jwtData ctxJwt.JWTData) (string, error) {
	claims := make(jwt.MapClaims)
	claims["exp"] = iat + seconds
	claims["iat"] = iat
	claims[ctxJwt.CtxKeyJwtData] = jwtData
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	return token.SignedString([]byte(secretKey))
}
