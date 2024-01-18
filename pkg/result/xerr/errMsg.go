package xerr

var message map[uint32]string

func init() {
	message = make(map[uint32]string)
	message[OK] = "SUCCESS"
	message[SERVER_COMMON_ERROR] = "服务器开小差啦,稍后再来试一试"
	message[REUQEST_PARAM_ERROR] = "参数错误"
	message[TOKEN_EXPIRE_ERROR] = "token失效，请重新登陆"
	message[TOKEN_GENERATE_ERROR] = "生成token失败"
	message[DB_ERROR] = "数据库繁忙,请稍后再试"
	message[DB_UPDATE_AFFECTED_ZERO_ERROR] = "更新数据影响行数为0"

	message[CAPTCHA_ERROR] = "验证码错误"
	message[USER_PASSWORD_ERROR] = "账号或者密码错误"

	message[REPEAT_NAME_ERROR] = "添加失败,不能添加同样的name"

	// sys_base 模块
	message[REPEAT_OSS_GET_BUCKET_ERROR] = "获取oss_bucket实例失败"
	message[REPEAT_OSS_PUT_BUCKET_ERROR] = "上传oss_bucket失败"
	message[EMAIL_CANNOT_BE_EMPTY_ERROR] = "邮箱不能为空错误"
	message[ALREADY_EXISTS_ERROR] = "邮箱已发送"
}

func MapErrMsg(errcode uint32) string {
	if msg, ok := message[errcode]; ok {
		return msg
	} else {
		return "服务器开小差啦,稍后再来试一试"
	}
}

func IsCodeErr(errcode uint32) bool {
	if _, ok := message[errcode]; ok {
		return true
	} else {
		return false
	}
}
