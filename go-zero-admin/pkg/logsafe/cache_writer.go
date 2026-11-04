package logsafe

import (
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// Setup 在依赖组装前安装唯一日志出口，后续框架 SetUp 不会覆盖它。
func Setup(c logx.LogConf, serviceName string) error {
	if c.ServiceName == "" {
		c.ServiceName = serviceName
	}
	if err := logx.SetUp(c); err != nil {
		return err
	}
	logx.SetWriter(cacheWriter{logx.Reset()})
	return nil
}

type cacheWriter struct{ logx.Writer }

func safeCacheEvent(v any) any {
	message, ok := v.(string)
	if !ok {
		return v
	}
	// Linux stat.Report 在同一事件前加时间、集群和主机；只识别该明确封装。
	lines := strings.Split(message, "\n")
	if len(lines) > 2 {
		if _, err := time.Parse(time.DateTime, lines[0]); err == nil {
			i := 1
			if strings.HasPrefix(lines[i], "cluster: ") {
				i++
			}
			if i < len(lines) && strings.HasPrefix(lines[i], "host: ") {
				i++
				if i < len(lines) && strings.HasPrefix(lines[i], "dropped: ") {
					i++
				}
				message = strings.Join(lines[i:], "\n")
			}
		}
	}
	if strings.HasPrefix(message, "[REDIS] slowcall on executing: ") {
		return "redis slow operation; sensitive details omitted"
	}
	for _, prefix := range []string{
		"failed to clear cache with key: ", "failed to clear cache with keys: ",
		"unmarshal cache, node: ", "delete invalid cache, node: ",
		"failed to set timer for keys: ", "failed to set timer for key: ",
		"retried but failed to clear cache with keys: ",
	} {
		if strings.HasPrefix(message, prefix) {
			return "cache operation failed; sensitive details omitted"
		}
	}
	return v
}

func (w cacheWriter) Alert(v any)                     { w.Writer.Alert(safeCacheEvent(v)) }
func (w cacheWriter) Debug(v any, f ...logx.LogField) { w.Writer.Debug(safeCacheEvent(v), f...) }
func (w cacheWriter) Error(v any, f ...logx.LogField) { w.Writer.Error(safeCacheEvent(v), f...) }
func (w cacheWriter) Info(v any, f ...logx.LogField)  { w.Writer.Info(safeCacheEvent(v), f...) }
func (w cacheWriter) Severe(v any)                    { w.Writer.Severe(safeCacheEvent(v)) }
func (w cacheWriter) Slow(v any, f ...logx.LogField)  { w.Writer.Slow(safeCacheEvent(v), f...) }
func (w cacheWriter) Stack(v any)                     { w.Writer.Stack(safeCacheEvent(v)) }
func (w cacheWriter) Stat(v any, f ...logx.LogField)  { w.Writer.Stat(safeCacheEvent(v), f...) }
