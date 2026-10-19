package captchalogic

import (
	"regexp"
)

// Captcha schema is owned by the RPC tier so the API tier has no Redis handle.
// Key layout, TTL, and image dimensions are all private to this package.
const (
	prefixCaptcha    = "biz#captcha#id:%s"
	expireCaptcha    = 120 // seconds before a freshly generated captcha is discarded
	captchaImgWidth  = 105
	captchaImgHeight = 36
	captchaImgLength = 6
)

var captchaIDPattern = regexp.MustCompile("^[A-Za-z0-9_-]{10,128}$")

// validCaptchaID rejects anything that could escape the key prefix or cause a
// Redis glob/CRLF surprise. Keep this strict even though base64Captcha
// generates compliant ids, because clients send the id back over JSON.
func validCaptchaID(id string) bool {
	return captchaIDPattern.MatchString(id)
}

func validCaptchaAnswer(answer string) bool {
	return len(answer) == captchaImgLength
}
