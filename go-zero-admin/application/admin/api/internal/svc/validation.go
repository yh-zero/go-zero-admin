package svc

import (
	"strconv"

	"github.com/go-playground/validator/v10"
)

// NewValidator 将契约中的字节上限和可清空邮箱规则用于实际 API 与回归测试。
func NewValidator() *validator.Validate {
	v := validator.New()
	// 指针非 nil 时 omitempty 不跳过空字符串，邮箱清空须单独放行。
	v.RegisterAlias("email_or_empty", "email|len=0")
	if err := v.RegisterValidation("maxbytes", func(fl validator.FieldLevel) bool {
		max, err := strconv.Atoi(fl.Param())
		return err == nil && len(fl.Field().String()) <= max
	}); err != nil {
		panic("注册 API 字节长度校验失败")
	}
	return v
}
