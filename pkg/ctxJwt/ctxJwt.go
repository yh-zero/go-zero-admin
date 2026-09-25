package ctxJwt

import (
	"context"
	"encoding/json"
)

var CtxKeyJwtData = "jwtData"

type JWTData struct {
	SessionVersion int64  `map:"SessionVersion"`
	AuthorityId    int64  `map:"AuthorityId"`
	ID             int64  `map:"ID"`
	NickName       string `map:"NickName"`
	UUID           string `map:"UUID"`
	Username       string `map:"Username"`
}

func GetJwtData(ctx context.Context) JWTData {
	ctxKeyJwtData := ctx.Value(CtxKeyJwtData)
	var jwtData JWTData
	if ctxKeyJwtDataClaim, ok := ctxKeyJwtData.(map[string]interface{}); ok {
		jwtData = mapToJWTData(ctxKeyJwtDataClaim)
	}
	return jwtData
}

func mapToJWTData(data map[string]interface{}) JWTData {
	var value JWTData
	encoded, err := json.Marshal(data)
	if err == nil {
		_ = json.Unmarshal(encoded, &value)
	}
	return value
}

func GetJwtDataID(ctx context.Context) int64 {
	return GetJwtData(ctx).ID
}

func GetJwtDataAuthorityId(ctx context.Context) int64 {
	return GetJwtData(ctx).AuthorityId
}
func GetJwtDataUUID(ctx context.Context) string {
	return GetJwtData(ctx).UUID
}

func GetJwtDataUsername(ctx context.Context) string {
	return GetJwtData(ctx).Username
}

func GetJwtDataNickName(ctx context.Context) string {
	return GetJwtData(ctx).NickName
}
