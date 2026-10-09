package userlogic

import (
	"net/mail"
	"unicode/utf8"
)

// The production profile columns are utf8mb4 varchar(191), measured in characters.
func validateProfileField(field, value string) error {
	if utf8.RuneCountInString(value) > 191 {
		return userError(field + "不能超过191个字符")
	}
	// Validate only supplied new email values. Empty values retain clear semantics;
	// addresses already stored on a user are not revalidated by unrelated edits.
	if field == "email" && value != "" {
		address, err := mail.ParseAddress(value)
		if err != nil || address.Address != value || address.Name != "" {
			return userError("邮箱格式无效")
		}
	}
	return nil
}
