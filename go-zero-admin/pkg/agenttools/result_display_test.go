package agenttools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

const displayAuditFixture = `{"startTime":"2026-10-04T00:00:00+08:00","endTime":"2026-10-04T12:00:00+08:00","total":3,"byModule":[{"module":"user","count":3}],"recent":[{"id":7,"actorName":"测试用户","module":"user","action":"login","result":"success","statusCode":200,"createdAt":"2026-10-04T02:00:00Z"}],"recentLimit":20,"moduleLimit":100}`
const displayFileFixture = `{"id":8,"name":"报表.csv","mime":"text/csv","size":42,"status":"active","visibility":"private","references":2,"explanation":"文件存在业务引用，必须先在对应业务移除引用"}`
const displayDevicesFixture = `{"total":1,"items":[{"device":1,"client":"Chrome","current":true,"createdAt":"2026-10-04T02:00:00Z","expiresAt":"2026-10-05T02:00:00Z"}],"limit":20}`

func TestFormatResultDisplaysAuthorizedFields(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		want          []string
	}{
		{"query_audit_logs", displayAuditFixture, []string{"匹配总数：3", "工具返回最近 1 条", "2026-10-04 10:00:00", "| 7 |", "测试用户", "完整记录请在审计日志页面"}},
		{"get_file_status", displayFileFixture, []string{"文件状态查询明细", "报表\\.csv", "| 大小（字节） | 42 |", "| 业务引用数 | 2 |", "文件存在业务引用"}},
		{"list_my_devices", displayDevicesFixture, []string{"本人有效设备总数：1", "Chrome", "| 1 | Chrome | 是 |", "2026-10-05 10:00:00"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatResult(tc.name, tc.content, 4096)
			if got == "" || len(got) > 4096 || !utf8.ValidString(got) {
				t.Fatalf("invalid display length %d", len(got))
			}
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in %s", want, got)
				}
			}
		})
	}
}

func TestFormatResultDistinguishesEmptyFromInvalid(t *testing.T) {
	for _, tc := range []struct{ name, content, want string }{
		{"query_audit_logs", `{"startTime":"2026-10-04T00:00:00+08:00","endTime":"2026-10-04T12:00:00+08:00","total":0,"byModule":[],"recent":[]}`, "没有匹配的审计日志"},
		{"list_my_devices", `{"total":0,"items":[]}`, "没有有效登录设备"},
	} {
		if got := FormatResult(tc.name, tc.content, 4096); !strings.Contains(got, tc.want) {
			t.Errorf("empty result %s: %q", tc.name, got)
		}
	}
	for _, tc := range []struct{ name, content string }{
		{"unknown", displayFileFixture},
		{"get_file_status", "[]"},
		{"get_file_status", "null"},
		{"get_file_status", "{}"},
		{"get_file_status", displayFileFixture + "{}"},
		{"get_file_status", strings.Replace(displayFileFixture, `"size":42`, `"size":"42"`, 1)},
		{"get_file_status", strings.Replace(displayFileFixture, `"size":42`, `"size":42.1`, 1)},
		{"get_file_status", strings.Replace(displayFileFixture, `"name":"报表.csv"`, `"name":null`, 1)},
		{"get_file_status", strings.Replace(displayFileFixture, `"references":2`, `"references":-1`, 1)},
		{"get_file_status", strings.Replace(displayFileFixture, "报表.csv", string([]byte{0xff}), 1)},
		{"query_audit_logs", strings.Replace(displayAuditFixture, `"total":3`, `"total":0`, 1)},
		{"query_audit_logs", strings.Replace(displayAuditFixture, `"statusCode":200`, `"statusCode":null`, 1)},
		{"query_audit_logs", strings.Replace(displayAuditFixture, "2026-10-04T02:00:00Z", "not a time", 1)},
		{"query_audit_logs", strings.Replace(displayAuditFixture, `"byModule":[{"module":"user","count":3}]`, `"byModule":null`, 1)},
		{"list_my_devices", `{"total":0,"items":[null]}`},
		{"list_my_devices", strings.Replace(displayDevicesFixture, `"current":true`, `"current":"true"`, 1)},
		{"list_my_devices", strings.Replace(displayDevicesFixture, `"total":1`, `"total":0`, 1)},
	} {
		if got := FormatResult(tc.name, tc.content, 4096); got != "" {
			t.Errorf("invalid result displayed for %s: %q", tc.name, got)
		}
	}
}

func TestFormatResultNeverCopiesPrivateOrUnknownFields(t *testing.T) {
	private := `,"key":"private-key","ip":"private-ip","token":"private-token","url":"https://private.test/?signed=secret","params":{"password":"private-password"},"rawRequest":"private-request"`
	for _, tc := range []struct{ name, content string }{
		{"query_audit_logs", strings.TrimSuffix(displayAuditFixture, "}") + private + "}"},
		{"get_file_status", strings.TrimSuffix(displayFileFixture, "}") + private + "}"},
		{"list_my_devices", strings.TrimSuffix(displayDevicesFixture, "}") + private + "}"},
	} {
		got := FormatResult(tc.name, tc.content, 4096)
		if got == "" {
			t.Fatalf("valid %s result with extra fields was discarded", tc.name)
		}
		for _, bad := range []string{"private-key", "private-ip", "private-token", "private.test", "secret", "private-password", "private-request"} {
			if strings.Contains(got, bad) {
				t.Errorf("private field %q appeared in %s", bad, tc.name)
			}
		}
	}
}

func TestFormatResultTreatsCellContentAsText(t *testing.T) {
	var result map[string]any
	if err := json.Unmarshal([]byte(displayFileFixture), &result); err != nil {
		t.Fatal(err)
	}
	result["name"] = "| injected |\n**override** `code` [visit](https://example.test/?token=hidden-token) <script>\u202e"
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	got := FormatResult("get_file_status", string(raw), 4096)
	for _, want := range []string{`\| injected \|`, `\*\*override\*\*`, "\\`code\\`", `\[visit\]`, "链接已隐藏", `\<script\>`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing escaped value %q: %s", want, got)
		}
	}
	for _, bad := range []string{"https://", "hidden-token", "\u202e", "\n**override**", "\n| injected |"} {
		if strings.Contains(got, bad) {
			t.Errorf("unsafe cell material %q: %s", bad, got)
		}
	}
}

func TestFormatResultBoundsRowsCellsAndByteBudget(t *testing.T) {
	items := make([]map[string]any, 25)
	for i := range items {
		items[i] = map[string]any{"device": i + 1, "client": strings.Repeat("测", 200), "current": false, "createdAt": "2026-10-04T02:00:00Z", "expiresAt": "2026-10-05T02:00:00Z"}
	}
	raw, err := json.Marshal(map[string]any{"total": 50, "items": items})
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{-1, 0, 1, 255, 256, 400, 1024, 4096, 10000} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			got := FormatResult("list_my_devices", string(raw), limit)
			if limit < 256 {
				if got != "" {
					t.Fatal("too-small budget displayed a partial result")
				}
				return
			}
			if len(got) > min(limit, maxDisplayBytes) || !utf8.ValidString(got) {
				t.Fatalf("display exceeds %d bytes or is not UTF-8: %d", limit, len(got))
			}
			if !strings.Contains(got, "明细已按显示上限截断") {
				t.Fatal("truncation has no explicit explanation")
			}
			if strings.Contains(got, "| 21 |") || strings.Contains(got, strings.Repeat("测", 65)) {
				t.Fatal("row or cell limit exceeded")
			}
		})
	}
	if got := FormatResult("get_file_status", strings.Repeat(" ", 32769), 4096); got != "" {
		t.Fatal("oversized tool JSON was accepted")
	}
}
