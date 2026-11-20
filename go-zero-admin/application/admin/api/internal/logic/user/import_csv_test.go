package user

import (
	"strings"
	"testing"
)

func TestParseUserImportCSVTracksPhysicalRows(t *testing.T) {
	input := "\ufeffusername,dept_id,real_name\r\n alice ,12,\"张,三\"\r\n\r\nli,0,\"李\n四\"\r\nwrong,-1,王\r\nmissing,12\r\n"
	entries, err := parseUserImportCSV(input)
	if err != nil || len(entries) != 4 {
		t.Fatalf("entries = %#v, err = %v", entries, err)
	}
	if got := entries[0]; got.row != 2 || got.username != "alice" || got.deptId != 12 || got.realName != "张,三" || got.message != "" {
		t.Fatalf("first row = %#v", got)
	}
	if got := entries[1]; got.row != 4 || got.deptId != 0 || got.realName != "李\n四" || got.message != "" {
		t.Fatalf("multiline row = %#v", got)
	}
	if got := entries[2]; got.row != 6 || got.message == "" {
		t.Fatalf("negative department row = %#v", got)
	}
	if got := entries[3]; got.row != 7 || got.message == "" {
		t.Fatalf("missing column row = %#v", got)
	}
}

func TestParseUserImportCSVRejectsInvalidFile(t *testing.T) {
	for _, input := range []string{
		"username,dept_id,password\n",
		"username,dept_id,real_name\n\n",
		"username,dept_id,real_name\nname,1,\"unterminated",
		"username,dept_id,real_name\n" + strings.Repeat("name,1,\n", 101),
		"username,dept_id,real_name\nname,1,\xff",
	} {
		if _, err := parseUserImportCSV(input); err == nil {
			t.Fatalf("accepted invalid CSV with %d bytes", len(input))
		}
	}
}
