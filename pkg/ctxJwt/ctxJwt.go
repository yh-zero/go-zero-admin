package ctxJwt

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

var CtxKeyJwtData = "jwtData"

type JWTData struct {
	AuthorityId int64  `map:"AuthorityId"`
	ID          int64  `map:"ID"`
	NickName    string `map:"NickName"`
	UUID        string `map:"UUID"`
	Username    string `map:"Username"`
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
	jwtData := JWTData{}
	v := reflect.ValueOf(&jwtData).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		mapTag := field.Tag.Get("map")

		if mapTag != "" {
			if value, ok := data[mapTag]; ok {
				switch v.Field(i).Kind() {
				case reflect.Int:
					if num, ok := value.(json.Number); ok {
						intValue, err := strconv.Atoi(num.String())
						if err != nil {
							fmt.Println("转换失败:", err)
						} else {
							v.Field(i).SetInt(int64(intValue))
						}
					}
				case reflect.Int64:
					if num, ok := value.(json.Number); ok {
						intValue, err := strconv.ParseInt(num.String(), 10, 64)
						if err != nil {
							fmt.Println("转换失败:", err)
						} else {
							v.Field(i).SetInt(intValue)
						}
					}
				default:
					if reflect.ValueOf(value).Type().AssignableTo(v.Field(i).Type()) {
						v.Field(i).Set(reflect.ValueOf(value))
					}
				}
			}
		}
	}
	fmt.Println("mapToJWTData", jwtData)
	return jwtData
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
