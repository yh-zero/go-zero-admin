package xerr

func MapErrMsg(errcode uint32) string {
	switch errcode {
	case SUCCESS:
		return "操作成功"
	case SERVER_COMMON_ERROR:
		return "服务器繁忙，请稍后再试"
	case REUQEST_PARAM_ERROR:
		return "参数错误"
	case TOKEN_EXPIRE_ERROR:
		return "登录已过期，请重新登录"
	case TOKEN_GENERATE_ERROR:
		return "令牌生成失败"
	case PERMISSION_DENIED:
		return "没有接口权限"
	case DB_ERROR:
		return "数据库错误"
	case DB_UPDATE_AFFECTED_ZERO:
		return "更新失败，记录不存在"
	case USER_NOT_FOUND:
		return "用户不存在"
	case USER_PASSWORD_ERROR:
		return "用户名或密码错误"
	case USER_DISABLED:
		return "账号已停用"
	case CAPTCHA_EXPIRED:
		return "验证码已过期"
	case CAPTCHA_WRONG:
		return "验证码错误"
	case USERNAME_EXISTS:
		return "用户名已存在"
	case OLD_PASSWORD_WRONG:
		return "原密码错误"
	case ROLE_NOT_FOUND:
		return "角色不存在"
	case ROLE_CODE_EXISTS:
		return "角色编码已存在"
	case MENU_NOT_FOUND:
		return "菜单不存在"
	case DEPT_NOT_FOUND:
		return "部门不存在"
	case DICT_NOT_FOUND:
		return "字典不存在"
	case DICT_CODE_EXISTS:
		return "字典编码已存在"
	case API_NOT_FOUND:
		return "API 资源不存在"
	default:
		return "未知错误"
	}
}
