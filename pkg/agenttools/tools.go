// Package agenttools describes the bounded, read-only business tools shared by
// the AI RPC and the business RPC. It has no model or database dependencies.
package agenttools

import "errors"

type Spec struct {
	Name, Label, Description, Path, Schema string
	SessionOnly                            bool
}

type Output struct {
	Content, Summary string
	Count            int
}

var ErrArguments = errors.New("invalid agent tool arguments")

var Specs = []Spec{
	{Name: "query_audit_logs", Label: "审计日志", Description: "查询指定时间范围审计日志、完整按模块计数和最多20条最近记录；默认上海时间今天，最多31天，不含请求原文或IP。", Path: "/v1/sys/audit/getAuditLogList", Schema: `{"type":"object","properties":{"startTime":{"type":"string","description":"RFC3339含时区"},"endTime":{"type":"string","description":"RFC3339含时区"},"module":{"type":"string"},"eventType":{"type":"string","enum":["operation","login"]},"result":{"type":"string","enum":["success","failure"]}},"additionalProperties":false}`},
	{Name: "get_file_status", Label: "文件状态", Description: "按fileId查询当前数据范围内文件的状态、大小及引用数，说明引用对删除的影响，不返回下载链接。", Path: "/v1/sys/files/list", Schema: `{"type":"object","properties":{"fileId":{"type":"integer","minimum":1}},"required":["fileId"],"additionalProperties":false}`},
	{Name: "list_my_devices", Label: "本人设备", Description: "查询本人最多20个有效登录设备，脱敏设备信息；只读取本人，不能查询其他用户。", Path: "/v1/sys/session/devices", SessionOnly: true, Schema: `{"type":"object","properties":{},"additionalProperties":false}`},
}
