package userlogic

import (
	"context"
	"fmt"
	"time"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/logx"
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
	fmt.Println("GenerateToken:----", in)
	now := time.Now().Unix()
	accessExpire := l.svcCtx.Config.JwtAuth.AccessExpire
	accessToken, err := l.getJwtToken(l.svcCtx.Config.JwtAuth.AccessSecret, now, accessExpire, ctxJwt.JWTData{
		UUID:        in.UUID,
		ID:          in.ID,
		NickName:    in.NickName,
		Username:    in.Username,
		AuthorityId: in.AuthorityId,
	})

	return &pb.GetUserTokeResponse{
		Token:     accessToken,
		ExpiresAt: now + accessExpire,
	}, err
}

func (l *GetUserTokeLogic) getJwtToken(secretKey string, iat, seconds int64, jwtData ctxJwt.JWTData) (string, error) {
	fmt.Println("======== getJwtToken jwtData", jwtData)
	claims := make(jwt.MapClaims)
	claims["exp"] = iat + seconds
	claims["iat"] = iat
	claims[ctxJwt.CtxKeyJwtData] = jwtData
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	return token.SignedString([]byte(secretKey))
}
