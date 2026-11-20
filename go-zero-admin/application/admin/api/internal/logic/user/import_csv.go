package user

import (
	"encoding/csv"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"go-zero-admin/pkg/xerr"
)

const maxUserImportRows = 100

type userImportEntry struct {
	row      int64
	username string
	deptId   int64
	realName string
	message  string
}

func parseUserImportCSV(input string) ([]userImportEntry, error) {
	if !utf8.ValidString(input) {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "请上传 UTF-8 编码的 CSV")
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(input, "\ufeff")))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil || !slices.Equal(header, []string{"username", "dept_id", "real_name"}) {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "CSV 表头须为 username,dept_id,real_name")
	}

	var entries []userImportEntry
	for {
		fields, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "CSV 格式不正确，请检查引号及分隔符")
		}
		if len(entries) == maxUserImportRows {
			return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "每次最多导入 100 条记录")
		}
		line, _ := reader.FieldPos(0)
		entry := userImportEntry{row: int64(line)}
		if len(fields) != 3 {
			entry.message = "每行须包含账号、部门 ID 和姓名三列"
		} else {
			entry.username = strings.TrimSpace(fields[0])
			entry.realName = strings.TrimSpace(fields[2])
			entry.deptId, err = strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
			switch {
			case err != nil || entry.deptId < 0:
				entry.deptId = 0
				entry.message = "部门 ID 须为非负整数"
			case utf8.RuneCountInString(entry.username) < 2 || utf8.RuneCountInString(entry.username) > 64 || strings.ContainsFunc(entry.username, unicode.IsControl):
				entry.username = ""
				entry.message = "账号须为 2 至 64 个字符，且不能包含控制字符"
			case utf8.RuneCountInString(entry.realName) > 64:
				entry.message = "姓名最多 64 个字符"
			}
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "CSV 须包含至少一条用户记录")
	}
	return entries, nil
}
