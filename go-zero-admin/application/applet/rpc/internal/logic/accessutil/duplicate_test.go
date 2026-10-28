package accessutil_test

import (
	"errors"
	"fmt"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/pkg/result/xerr"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestDuplicateConstraintMessages(t *testing.T) {
	for index, message := range map[string]string{
		"uk_sys_menus_active_name":            "菜单路由名称已存在",
		"uk_sys_menus_active_path":            "菜单路径已存在",
		"uk_sys_apis_active_route":            "相同路径和请求方法的API已存在",
		"uk_sys_dictionaries_active_type":     "字典类型已存在",
		"uk_sys_dictionary_info_active_value": "当前字典中已存在相同的字典值",
	} {
		original := fmt.Errorf("write: %w", &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry for key '" + index + "'"})
		var business *xerr.CodeError
		if translated := accessutil.FriendlyDuplicate(original); !errors.As(translated, &business) || business.GetErrMsg() != message {
			t.Fatalf("lost duplicate message for %s: %v", index, translated)
		}
	}
	for _, original := range []error{nil, errors.New("connection failed"), &mysqldriver.MySQLError{Number: 1062, Message: "PRIMARY"}} {
		if accessutil.FriendlyDuplicate(original) != original {
			t.Fatal("unrelated error should remain unchanged")
		}
	}
}

// Exercise the exact nullable-marker/index semantics in an isolated database.
// MySQL uses STORED; SQLite ALTER TABLE supports only VIRTUAL generated columns.
func TestActiveBusinessConstraintsAndSoftDeleteReuse(t *testing.T) {
	cases := []struct{ name, table, columns, insert string }{
		{"menu-name", "sys_base_menus", "name,active_unique", "INSERT INTO sys_base_menus(name,path) VALUES ('unique-name','a-path')"},
		{"menu-path", "sys_base_menus", "path,active_unique", "INSERT INTO sys_base_menus(name,path) VALUES ('a-name','unique-path')"},
		{"api-route", "sys_apis", "path,method,active_unique", "INSERT INTO sys_apis(path,method) VALUES ('/v1/sys/test','GET')"},
		{"dictionary-type", "sys_dictionaries", "type,active_unique", "INSERT INTO sys_dictionaries(name,type) VALUES ('Status','status')"},
		{"dictionary-value", "sys_dictionary_info", "sys_dictionary_id,value,active_unique", "INSERT INTO sys_dictionary_info(sys_dictionary_id,value) VALUES (10,0)"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			s := service(t)
			must(t, s.DB.Exec("ALTER TABLE "+test.table+" ADD COLUMN active_unique INTEGER GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN 1 ELSE NULL END) VIRTUAL").Error)
			must(t, s.DB.Exec("CREATE UNIQUE INDEX test_active_unique ON "+test.table+"("+test.columns+")").Error)
			for i := 0; i < 2; i++ {
				must(t, s.DB.Exec(test.insert).Error)
				mustFail(t, s.DB.Exec(test.insert).Error) // Simulates a competing writer past the read check.
				must(t, s.DB.Exec("UPDATE "+test.table+" SET deleted_at=CURRENT_TIMESTAMP WHERE deleted_at IS NULL").Error)
			}
			must(t, s.DB.Exec(test.insert).Error)
			var active int64
			must(t, s.DB.Table(test.table).Where("deleted_at IS NULL").Count(&active).Error)
			if active != 1 {
				t.Fatalf("expected 1 active record; got %d", active)
			}
		})
	}
}
