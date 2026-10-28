// Package httpprivacy prevents framework HTTP dumps from persisting credentials.
package httpprivacy

import (
	"errors"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

// ConfigureServer disables go-zero's request/response dumps. Its brief logger
// dumps headers and the request body on 5xx; verbose mode also dumps responses.
func ConfigureServer(conf *rest.RestConf) {
	conf.Middlewares.Log = false
	conf.Verbose = false
}

// Install preserves the already configured console/file/rotation writer and
// filters framework dumps at the final logging boundary. go-zero's JWT failure
// logger is unconditional and runs before UnauthorizedCallback, so a normal
// rest middleware cannot hide its Authorization header or password body.
// Call once during startup, after rest.MustNewServer and before server.Start.
func Install() error {
	previous := logx.Reset()
	if previous == nil {
		return errors.New("HTTP privacy logging must be installed after logger initialization")
	}
	if _, ok := previous.(*privacyWriter); ok {
		logx.SetWriter(previous)
		return nil
	}
	logx.SetWriter(&privacyWriter{next: previous})
	return nil
}

type privacyWriter struct{ next logx.Writer }

func safe(value any) any {
	var message string
	switch v := value.(type) {
	case string:
		message = v
	case error:
		message = v.Error()
	default:
		return value
	}
	if strings.HasPrefix(message, "authorize failed:") {
		return "JWT authorization failed; request content omitted"
	}
	// Defense in depth against re-enabling framework dumps. Structured business
	// logs, metrics and the explicit audit trail retain their existing writers.
	if strings.HasPrefix(message, "[HTTP]") {
		return "HTTP framework request dump omitted; use audit and metrics"
	}
	if strings.HasPrefix(message, "[http] dropped,") {
		return "HTTP request dropped by load shedding; request URI omitted"
	}
	return value
}
func (w *privacyWriter) Alert(v any)                     { w.next.Alert(safe(v)) }
func (w *privacyWriter) Close() error                    { return w.next.Close() }
func (w *privacyWriter) Debug(v any, f ...logx.LogField) { w.next.Debug(safe(v), f...) }
func (w *privacyWriter) Error(v any, f ...logx.LogField) { w.next.Error(safe(v), f...) }
func (w *privacyWriter) Info(v any, f ...logx.LogField)  { w.next.Info(safe(v), f...) }
func (w *privacyWriter) Severe(v any)                    { w.next.Severe(safe(v)) }
func (w *privacyWriter) Slow(v any, f ...logx.LogField)  { w.next.Slow(safe(v), f...) }
func (w *privacyWriter) Stack(v any)                     { w.next.Stack(safe(v)) }
func (w *privacyWriter) Stat(v any, f ...logx.LogField)  { w.next.Stat(safe(v), f...) }

var _ logx.Writer = (*privacyWriter)(nil)
