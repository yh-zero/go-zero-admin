package userservicelogic

import (
	"go-zero-admin/pkg/xerr"
	"unicode/utf8"
)

func validateNewPassword(password string) error {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "密码至少8个字符且最多72字节")
	}
	return nil
}
