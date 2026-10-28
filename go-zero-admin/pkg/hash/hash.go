package hash

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinPasswordBytes = 8
	MaxPasswordBytes = 72
)

// ValidatePassword applies to newly created passwords, not legacy login checks.
// bcrypt limits passwords by byte length, including UTF-8 characters.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordBytes || len(password) > MaxPasswordBytes {
		return errors.New("密码长度必须为8至72字节")
	}
	return nil
}

// BcryptHash 使用 bcrypt 对密码进行加密
func BcryptHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// BcryptCheck 对比明文密码和数据库的哈希值
func BcryptCheck(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
