package accessutil

import (
	"errors"
	"strings"

	mysqldriver "github.com/go-sql-driver/mysql"
	"go-zero-admin/pkg/result/xerr"
)

// The pre-write uniqueness check gives quick feedback; these database indexes
// also catch concurrent writes that both passed that check.
func FriendlyDuplicate(err error) error {
	var mysqlError *mysqldriver.MySQLError
	if !errors.As(err, &mysqlError) || mysqlError.Number != 1062 {
		return err
	}
	messages := map[string]string{
		"uk_sys_menus_active_name":            "菜单路由名称已存在",
		"uk_sys_menus_active_path":            "菜单路径已存在",
		"uk_sys_apis_active_route":            "相同路径和请求方法的API已存在",
		"uk_sys_dictionaries_active_type":     "字典类型已存在",
		"uk_sys_dictionary_info_active_value": "当前字典中已存在相同的字典值",
	}
	for index, message := range messages {
		if strings.Contains(mysqlError.Message, index) {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, message)
		}
	}
	return err
}
