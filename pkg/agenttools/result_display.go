package agenttools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxDisplayBytes = 4096
	maxDisplayRows  = 20
	maxCellRunes    = 64
	displayCutNote  = "\n明细已按显示上限截断；可在对应业务页面查看完整记录。\n"
)

var displayURL = regexp.MustCompile(`(?i)(?:[a-z][a-z0-9+.-]*://|(?:javascript|data|mailto|file):|www\.)[^\s<>\[\]{}()]+`)

// FormatResult projects a successful built-in tool result onto its display
// fields. It never returns raw JSON, unknown fields, URLs or tool instructions.
// The caller must still enforce authorization and successful task completion
// before persisting the block. Invalid/unsupported results produce no block,
// rather than a misleading claim that the query returned no data.
func FormatResult(name, content string, maxBytes int) string {
	if maxBytes < 256 || len(content) > 32768 || !utf8.ValidString(content) {
		return ""
	}
	content = strings.TrimSpace(content)
	if len(content) < 2 || content[0] != '{' || !json.Valid([]byte(content)) {
		return ""
	}
	limit := min(maxBytes, maxDisplayBytes)
	var lines []string
	var truncated bool
	headerLines := 2
	switch name {
	case "query_audit_logs":
		lines, truncated = displayAudit(content)
		headerLines = 3
	case "get_file_status":
		lines, truncated = displayFile(content)
	case "list_my_devices":
		lines, truncated = displayDevices(content)
	default:
		return ""
	}
	if len(lines) == 0 {
		return ""
	}
	// Counts and filter metadata are required context, even when the body is
	// shortened. If that context cannot fit, omit the block altogether.
	headerBytes := len(displayCutNote)
	for _, line := range lines[:headerLines] {
		headerBytes += len(line) + 1
	}
	if headerBytes > limit {
		return ""
	}
	var out strings.Builder
	for _, line := range lines {
		// Reserve a complete, explicit note before adding any table row. Do not
		// cut a UTF-8 sequence, an escape or the middle of a table cell.
		if out.Len()+len(line)+1+len(displayCutNote) > limit {
			truncated = true
			break
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if out.Len() == 0 {
		return ""
	}
	if truncated {
		out.WriteString(displayCutNote)
	}
	return strings.TrimSpace(out.String())
}

type displayAuditResult struct {
	StartTime *string            `json:"startTime"`
	EndTime   *string            `json:"endTime"`
	Total     *int64             `json:"total"`
	ByModule  *[]displayModule   `json:"byModule"`
	Recent    *[]displayAuditRow `json:"recent"`
}

type displayModule struct {
	Module *string `json:"module"`
	Count  *int64  `json:"count"`
}

type displayAuditRow struct {
	ID         *int64  `json:"id"`
	ActorName  *string `json:"actorName"`
	Module     *string `json:"module"`
	Action     *string `json:"action"`
	Result     *string `json:"result"`
	StatusCode *int64  `json:"statusCode"`
	CreatedAt  *string `json:"createdAt"`
}

func displayAudit(content string) ([]string, bool) {
	var r displayAuditResult
	if json.Unmarshal([]byte(content), &r) != nil || r.Total == nil || *r.Total < 0 || r.ByModule == nil || r.Recent == nil {
		return nil, false
	}
	start, ok := displayTime(r.StartTime)
	if !ok {
		return nil, false
	}
	end, ok := displayTime(r.EndTime)
	if !ok || int64(len(*r.Recent)) > *r.Total {
		return nil, false
	}
	for _, row := range *r.ByModule {
		if row.Module == nil || row.Count == nil || *row.Count < 0 || *row.Count > *r.Total {
			return nil, false
		}
	}
	for _, row := range *r.Recent {
		if row.ID == nil || *row.ID <= 0 || row.ActorName == nil || row.Module == nil || row.Action == nil || row.Result == nil || row.StatusCode == nil || *row.StatusCode < 0 || *row.StatusCode > 999 {
			return nil, false
		}
		if _, ok := displayTime(row.CreatedAt); !ok {
			return nil, false
		}
	}
	lines := []string{
		"### 审计日志查询明细",
		"时间范围（上海时间）：" + start + " 至 " + end + "。",
		fmt.Sprintf("匹配总数：%d。工具返回最近 %d 条记录；显示上限：%d 条。", *r.Total, len(*r.Recent), maxDisplayRows),
	}
	if *r.Total == 0 {
		if len(*r.ByModule) != 0 || len(*r.Recent) != 0 {
			return nil, false
		}
		return append(lines, "该时间范围和筛选条件下没有匹配的审计日志。"), false
	}
	truncated := len(*r.ByModule) > maxDisplayRows || len(*r.Recent) > maxDisplayRows
	if len(*r.ByModule) > 0 {
		lines = append(lines, "", "按模块汇总（每个计数覆盖完整匹配范围）：", "| 模块 | 条数 |", "| --- | --- |")
		for _, row := range (*r.ByModule)[:min(len(*r.ByModule), maxDisplayRows)] {
			lines = append(lines, displayRow(displayCell(*row.Module), fmt.Sprint(*row.Count)))
		}
	}
	if len(*r.Recent) > 0 {
		lines = append(lines, "", "最近记录：", "| ID | 时间（上海） | 操作人 | 模块 | 操作 | 结果 | 状态码 |", "| --- | --- | --- | --- | --- | --- | --- |")
		for _, row := range (*r.Recent)[:min(len(*r.Recent), maxDisplayRows)] {
			created, _ := displayTime(row.CreatedAt)
			lines = append(lines, displayRow(fmt.Sprint(*row.ID), created, displayCell(*row.ActorName), displayCell(*row.Module), displayCell(*row.Action), displayCell(*row.Result), fmt.Sprint(*row.StatusCode)))
		}
	} else {
		lines = append(lines, "工具未返回最近记录明细；匹配总数不代表此处显示了全部记录。")
	}
	if *r.Total > int64(len(*r.Recent)) {
		lines = append(lines, "这里只展示最近记录样本；完整记录请在审计日志页面按相同条件查询。")
	}
	return lines, truncated
}

type displayFileResult struct {
	ID          *int64  `json:"id"`
	Name        *string `json:"name"`
	Mime        *string `json:"mime"`
	Size        *int64  `json:"size"`
	Status      *string `json:"status"`
	Visibility  *string `json:"visibility"`
	References  *int64  `json:"references"`
	Explanation *string `json:"explanation"`
}

func displayFile(content string) ([]string, bool) {
	var r displayFileResult
	if json.Unmarshal([]byte(content), &r) != nil || r.ID == nil || *r.ID <= 0 || r.Size == nil || *r.Size < 0 || r.References == nil || *r.References < 0 || r.Name == nil || r.Mime == nil || r.Status == nil || r.Visibility == nil || r.Explanation == nil {
		return nil, false
	}
	return []string{
		"### 文件状态查询明细",
		"本次返回 1 个当前数据范围内的文件。",
		"| 字段 | 值 |",
		"| --- | --- |",
		displayRow("文件 ID", fmt.Sprint(*r.ID)),
		displayRow("名称", displayCell(*r.Name)),
		displayRow("类型", displayCell(*r.Mime)),
		displayRow("大小（字节）", fmt.Sprint(*r.Size)),
		displayRow("状态", displayCell(*r.Status)),
		displayRow("可见性", displayCell(*r.Visibility)),
		displayRow("业务引用数", fmt.Sprint(*r.References)),
		displayRow("说明", displayCell(*r.Explanation)),
	}, false
}

type displayDevicesResult struct {
	Total *int64              `json:"total"`
	Items *[]displayDeviceRow `json:"items"`
}

type displayDeviceRow struct {
	Device    *int64  `json:"device"`
	Client    *string `json:"client"`
	Current   *bool   `json:"current"`
	CreatedAt *string `json:"createdAt"`
	ExpiresAt *string `json:"expiresAt"`
}

func displayDevices(content string) ([]string, bool) {
	var r displayDevicesResult
	if json.Unmarshal([]byte(content), &r) != nil || r.Total == nil || *r.Total < 0 || r.Items == nil || int64(len(*r.Items)) > *r.Total {
		return nil, false
	}
	for _, row := range *r.Items {
		if row.Device == nil || *row.Device <= 0 || row.Client == nil || row.Current == nil {
			return nil, false
		}
		if _, ok := displayTime(row.CreatedAt); !ok {
			return nil, false
		}
		if _, ok := displayTime(row.ExpiresAt); !ok {
			return nil, false
		}
	}
	lines := []string{
		"### 本人登录设备查询明细",
		fmt.Sprintf("本人有效设备总数：%d。工具返回 %d 条；显示上限：%d 条。", *r.Total, len(*r.Items), maxDisplayRows),
	}
	if *r.Total == 0 {
		return append(lines, "当前没有有效登录设备。"), false
	}
	if len(*r.Items) > 0 {
		lines = append(lines, "| 设备序号 | 客户端 | 当前设备 | 登录时间（上海） | 到期时间（上海） |", "| --- | --- | --- | --- | --- |")
		for _, row := range (*r.Items)[:min(len(*r.Items), maxDisplayRows)] {
			created, _ := displayTime(row.CreatedAt)
			expires, _ := displayTime(row.ExpiresAt)
			current := "否"
			if *row.Current {
				current = "是"
			}
			lines = append(lines, displayRow(fmt.Sprint(*row.Device), displayCell(*row.Client), current, created, expires))
		}
	} else {
		lines = append(lines, "工具未返回设备明细；设备总数不代表此处显示了全部设备。")
	}
	if *r.Total > int64(len(*r.Items)) {
		lines = append(lines, "这里只展示部分设备；完整列表请在本人设备页面查看。")
	}
	return lines, len(*r.Items) > maxDisplayRows
}

func displayTime(value *string) (string, bool) {
	if value == nil {
		return "", false
	}
	t, err := time.Parse(time.RFC3339Nano, *value)
	if err != nil {
		return "", false
	}
	return t.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02 15:04:05"), true
}

func displayRow(cells ...string) string { return "| " + strings.Join(cells, " | ") + " |" }

func displayCell(value string) string {
	value = displayURL.ReplaceAllString(value, "链接已隐藏")
	var out strings.Builder
	count := 0
	for _, r := range value {
		if unicode.IsSpace(r) {
			r = ' '
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			if r == '\n' || r == '\r' || r == '\t' {
				r = ' '
			} else {
				continue
			}
		}
		if count >= maxCellRunes {
			out.WriteRune('…')
			break
		}
		if strings.ContainsRune("\\`*_{}[]()#+.!<>|~-", r) {
			out.WriteByte('\\')
		}
		out.WriteRune(r)
		count++
	}
	return strings.TrimSpace(out.String())
}
