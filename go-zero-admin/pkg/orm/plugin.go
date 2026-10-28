package orm

import (
	"context"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type CustomePlugin struct{}

type operationTrace struct {
	start   time.Time
	span    oteltrace.Span
	context context.Context
}

func NewCustomePlugin() *CustomePlugin { return &CustomePlugin{} }
func (p *CustomePlugin) Name() string  { return "CustomePlugin" }

func (p *CustomePlugin) Initialize(db *gorm.DB) error {
	callbacks := []struct {
		operation string
		before    func(string, func(*gorm.DB)) error
		after     func(string, func(*gorm.DB)) error
	}{
		{"create", db.Callback().Create().Before("*").Register, db.Callback().Create().After("*").Register},
		{"query", db.Callback().Query().Before("*").Register, db.Callback().Query().After("*").Register},
		{"update", db.Callback().Update().Before("*").Register, db.Callback().Update().After("*").Register},
		{"delete", db.Callback().Delete().Before("*").Register, db.Callback().Delete().After("*").Register},
		{"row", db.Callback().Row().Before("*").Register, db.Callback().Row().After("*").Register},
		{"raw", db.Callback().Raw().Before("*").Register, db.Callback().Raw().After("*").Register},
	}
	for _, callback := range callbacks {
		if err := callback.before("metric:trace:before:"+callback.operation, beginOperation(callback.operation)); err != nil {
			return err
		}
		if err := callback.after("metric:trace:after:"+callback.operation, endOperation(callback.operation)); err != nil {
			return err
		}
	}
	return nil
}

func beginOperation(operation string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		original := db.Statement.Context
		ctx, span := trace.TracerFromContext(original).Start(original, "gorm:"+operation, oteltrace.WithSpanKind(oteltrace.SpanKindClient))
		db.InstanceSet("gorm:operation_trace:"+operation, operationTrace{start: time.Now(), span: span, context: original})
		db.Statement.Context = ctx
	}
}

func endOperation(operation string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		value, found := db.InstanceGet("gorm:operation_trace:" + operation)
		if !found {
			return
		}
		state, ok := value.(operationTrace)
		if !ok {
			return
		}
		defer state.span.End()
		defer func() { db.Statement.Context = state.context }()
		metricClientReqDur.ObserveFloat(float64(time.Since(state.start))/float64(time.Millisecond), db.Statement.Table, operation)
		metricClientReqErrTotal.Inc(db.Statement.Table, operation, strconv.FormatBool(db.Error != nil))
		if db.Error != nil {
			// Error text can contain duplicate-key values, so only record the status.
			state.span.SetStatus(codes.Error, "database operation failed")
		}
		state.span.SetAttributes(semconv.DBSQLTableKey.String(db.Statement.Table), semconv.DBStatementKey.String(redactSQL(db.Statement.SQL.String())))
	}
}
