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
	AuthorityId int    `map:"AuthorityId"`
	ID          int    `map:"ID"`
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
				// 检查当前字段的类型是否为 int
				if v.Field(i).Kind() == reflect.Int {
					// 检查值的类型是否为 json.Number
					if num, ok := value.(json.Number); ok {
						// 将 json.Number 转换为 int
						intValue, err := strconv.Atoi(num.String())
						if err != nil {
							// 处理转换错误
							fmt.Println("转换失败:", err)
							// 可以返回默认值或其他适当操作
						} else {
							// 设置字段的 int 值
							v.Field(i).SetInt(int64(intValue))
						}
					}
				} else {
					// 非 int 类型的字段直接赋值（假设类型匹配）
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

func GetJwtDataID(ctx context.Context) int {
	return GetJwtData(ctx).ID
}

func GetJwtDataAuthorityId(ctx context.Context) int {
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
