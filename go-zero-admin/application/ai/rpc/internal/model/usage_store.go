package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/pkg/llm"
)

var (
	ErrUsageQuota     = errors.New("每日供应商请求额度已用完")
	ErrUsageCapacity  = errors.New("AI供应商请求容量已满")
	ErrUsageConflict  = errors.New("用量策略版本已变化")
	ErrUsageInvalid   = errors.New("用量参数或状态无效")
	ErrUsageCommitted = errors.New("用量数据已提交但缓存失效未确认")
)

// UsageStore 的准入、结算与汇总都读取MySQL；Redis丢失不能绕过日额度或全局并发。
type UsageStore struct {
	conn   sqlx.SqlConn
	cached sqlc.CachedConn
}

func NewUsageStore(conn sqlx.SqlConn, c cache.CacheConf) *UsageStore {
	return &UsageStore{conn: conn, cached: sqlc.NewConn(conn, c)}
}

type UsageFilter struct {
	OwnerId                                                                int64
	AllUsers                                                               bool
	FromDay, ToDay, Source, Operation, Provider, Model, Status, UsageState string
	Page, PageSize                                                         int64
}

type UsageCounters struct {
	ConfirmedRequests     int64 `db:"confirmed_requests"`
	ReservedRequests      int64 `db:"reserved_requests"`
	NotSentRequests       int64 `db:"not_sent_requests"`
	UncertainRequests     int64 `db:"uncertain_requests"`
	SucceededRequests     int64 `db:"succeeded_requests"`
	FailedRequests        int64 `db:"failed_requests"`
	CancelledRequests     int64 `db:"cancelled_requests"`
	CompleteRequests      int64 `db:"complete_requests"`
	PartialRequests       int64 `db:"partial_requests"`
	UnknownRequests       int64 `db:"unknown_requests"`
	InvalidRequests       int64 `db:"invalid_requests"`
	KnownPromptTokens     int64 `db:"known_prompt_tokens"`
	KnownCompletionTokens int64 `db:"known_completion_tokens"`
	KnownEmbeddingTokens  int64 `db:"known_embedding_tokens"`
	KnownTotalTokens      int64 `db:"known_total_tokens"`
}

type UsageGroup struct {
	Source    string `db:"source"`
	Operation string `db:"operation"`
	Provider  string `db:"provider"`
	Model     string `db:"model"`
	UsageCounters
}

type UsageSummary struct {
	FromDay, ToDay, Timezone string
	RecordingStartedAt       int64
	UsageCounters
	Groups []*UsageGroup
}

type UsageCallRow struct {
	Id                int64  `db:"id"`
	AttemptId         string `db:"attempt_id"`
	OwnerId           int64  `db:"owner_id"`
	Source            string `db:"source"`
	Operation         string `db:"operation"`
	RequestId         string `db:"request_id"`
	SourceId          int64  `db:"source_id"`
	StepIndex         int64  `db:"step_index"`
	BatchIndex        int64  `db:"batch_index"`
	InputItems        int64  `db:"input_items"`
	Provider          string `db:"provider"`
	Model             string `db:"model"`
	QuotaDay          string `db:"quota_day"`
	QuotaCharged      bool   `db:"quota_charged"`
	DispatchConfirmed bool   `db:"dispatch_confirmed"`
	Status            string `db:"status"`
	UsageState        string `db:"usage_state"`
	PromptTokens      *int64 `db:"prompt_tokens"`
	CompletionTokens  *int64 `db:"completion_tokens"`
	TotalTokens       *int64 `db:"total_tokens"`
	HttpStatus        int64  `db:"http_status"`
	ProviderRequestId string `db:"provider_request_id"`
	ErrorCode         string `db:"error_code"`
	StartedAt         int64  `db:"started_at"`
	DispatchedAt      int64  `db:"dispatched_at"`
	CompletedAt       int64  `db:"completed_at"`
}

type UsagePolicyView struct {
	OwnerId               int64
	QuotaDay              string
	DailyRequestLimit     int64
	UsedRequests          int64
	RemainingRequests     int64
	Unlimited             bool
	Inherited             bool
	DailyTokenWarning     int64
	KnownTokens           int64
	WarningReached        bool
	IncompleteRequests    int64
	Version               int64
	GlobalConcurrentLimit int64
	ActiveCalls           int64
	ActiveKnown           bool
}

type UsagePolicyPatch struct {
	DailyRequestLimit, DailyTokenWarning, GlobalConcurrentLimit *int64
	InheritDefault                                              bool
	UpdatedBy                                                   int64
}

func (m *UsageStore) Reserve(ctx context.Context, attempt llm.UsageAttempt) error {
	parsed, err := uuid.Parse(attempt.AttemptID)
	if err != nil || len(attempt.AttemptID) != 36 || !strings.EqualFold(parsed.String(), attempt.AttemptID) || attempt.UserID <= 0 || attempt.SourceID < 0 || attempt.StepIndex < -1 || attempt.BatchIndex < 0 || attempt.InputItems < 0 ||
		!usageOneOf(attempt.Source, "chat", "analysis", "kb_search", "kb_ingest", "evaluation") || !usageOneOf(attempt.Operation, "chat", "embedding") || strings.TrimSpace(attempt.Provider) == "" || strings.TrimSpace(attempt.Model) == "" ||
		!utf8.ValidString(attempt.RequestID+attempt.Provider+attempt.Model) || utf8.RuneCountInString(attempt.RequestID) > 64 || utf8.RuneCountInString(attempt.Provider) > 32 || utf8.RuneCountInString(attempt.Model) > 128 || strings.IndexFunc(attempt.RequestID+attempt.Provider+attempt.Model, unicode.IsControl) >= 0 {
		return ErrUsageInvalid
	}
	// 准备继承行采用独立自动提交，避免多个事务先持INSERT IGNORE共享锁再升级排他锁。
	// 此步不扣额度；真正的配额检查和调用预留仍在下面同一用户行锁事务内。
	prepared, err := m.conn.ExecCtx(ctx, "INSERT IGNORE INTO ai_usage_policy(owner_id,daily_request_limit) VALUES (?,NULL)", attempt.UserID)
	if err != nil {
		return err
	}
	if inserted, err := prepared.RowsAffected(); err != nil {
		return err
	} else if inserted > 0 {
		if err := m.cached.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiUsagePolicyOwnerIdPrefix, attempt.UserID)); err != nil {
			return usageCommittedError(err)
		}
	}
	var id int64
	err = m.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		// 全局行串行化全栈容量准入，再锁用户额度；策略更新采用相同顺序。
		// 两个锁到手后才建立普通读取快照，Redis丢失不能绕过MySQL最终准入。
		var local, global AiUsagePolicy
		if err := s.QueryRowCtx(ctx, &global, "SELECT "+aiUsagePolicyRows+" FROM ai_usage_policy WHERE owner_id=0 FOR UPDATE"); err != nil {
			return err
		}
		if err := s.QueryRowCtx(ctx, &local, "SELECT "+aiUsagePolicyRows+" FROM ai_usage_policy WHERE owner_id=? FOR UPDATE", attempt.UserID); err != nil {
			return err
		}
		if global.GlobalConcurrentLimit < 1 || global.GlobalConcurrentLimit > 64 {
			return ErrUsageInvalid
		}
		var active int64
		if err := s.QueryRowCtx(ctx, &active, "SELECT COUNT(*) FROM ai_usage_call WHERE status IN ('reserved','running') AND lease_until>NOW(3)"); err != nil {
			return err
		}
		if active >= int64(global.GlobalConcurrentLimit) {
			return ErrUsageCapacity
		}
		limit := local.DailyRequestLimit
		if !limit.Valid {
			limit = global.DailyRequestLimit
		}
		if !limit.Valid || limit.Int64 < 0 {
			return ErrUsageInvalid
		}
		var day string
		if err := s.QueryRowCtx(ctx, &day, "SELECT DATE_FORMAT(UTC_TIMESTAMP()+INTERVAL 8 HOUR,'%Y-%m-%d')"); err != nil {
			return err
		}
		var used int64
		if err := s.QueryRowCtx(ctx, &used, "SELECT COUNT(*) FROM ai_usage_call WHERE owner_id=? AND quota_day=? AND quota_charged=1", attempt.UserID, day); err != nil {
			return err
		}
		if limit.Int64 > 0 && used >= limit.Int64 {
			return ErrUsageQuota
		}
		result, err := s.ExecCtx(ctx, "INSERT INTO ai_usage_call(attempt_id,owner_id,source,operation,request_id,source_id,step_index,batch_index,input_items,provider,model,quota_day,lease_until) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,DATE_ADD(NOW(3),INTERVAL 120 SECOND))", attempt.AttemptID, attempt.UserID, attempt.Source, attempt.Operation, attempt.RequestID, attempt.SourceID, attempt.StepIndex, attempt.BatchIndex, attempt.InputItems, attempt.Provider, attempt.Model, day)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
		return err
	})
	if err != nil {
		return err
	}
	return usageCommittedError(m.cached.DelCacheCtx(ctx, m.usageCallKeys(id, attempt.AttemptID, attempt.UserID)...))
}

func usageCommittedError(err error) error {
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUsageCommitted, err)
	}
	return nil
}

func (m *UsageStore) usageCallKeys(id int64, attempt string, owner int64) []string {
	return []string{fmt.Sprintf("%s%d", cacheGozeroAdminAiUsageCallIdPrefix, id), cacheGozeroAdminAiUsageCallAttemptIdPrefix + attempt, fmt.Sprintf("%s%d", cacheGozeroAdminAiUsagePolicyOwnerIdPrefix, owner)}
}

func (m *UsageStore) MarkDispatched(ctx context.Context, attempt string) error {
	var row AiUsageCall
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if err := s.QueryRowCtx(ctx, &row, "SELECT "+aiUsageCallRows+" FROM ai_usage_call WHERE attempt_id=? FOR UPDATE", attempt); err != nil {
			return err
		}
		if row.Status == "not_sent" {
			return ErrUsageInvalid
		}
		if row.DispatchConfirmed != 0 {
			return nil
		}
		_, err := s.ExecCtx(ctx, "UPDATE ai_usage_call SET dispatch_confirmed=1,dispatched_at=COALESCE(dispatched_at,NOW(3)),status=IF(status='reserved','running',status) WHERE attempt_id=?", attempt)
		return err
	})
	if err != nil {
		return err
	}
	return usageCommittedError(m.cached.DelCacheCtx(ctx, m.usageCallKeys(int64(row.Id), attempt, int64(row.OwnerId))...))
}

var usageProviderID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var usageErrorCode = regexp.MustCompile(`^[a-z0-9_]{0,64}$`)

func normalizeUsageObservation(operation string, observation llm.UsageObservation) (llm.UsageObservation, error) {
	if !usageOneOf(observation.Status, "succeeded", "failed", "cancelled", "unknown", "not_sent") || !usageOneOf(observation.UsageState, "unknown", "partial", "complete", "invalid") {
		return observation, ErrUsageInvalid
	}
	if !usageProviderID.MatchString(observation.ProviderRequestID) {
		observation.ProviderRequestID = ""
	}
	if !usageErrorCode.MatchString(observation.ErrorCode) {
		observation.ErrorCode = "internal_error"
	}
	if observation.HTTPStatus != 0 && (observation.HTTPStatus < 100 || observation.HTTPStatus > 599) {
		observation.HTTPStatus = 0
	}
	if operation == "embedding" {
		observation.CompletionTokens = nil
	}
	for _, token := range []**int64{&observation.PromptTokens, &observation.CompletionTokens, &observation.TotalTokens} {
		if *token != nil && (**token < 0 || **token > 10_000_000) {
			*token = nil
			observation.UsageState = "invalid"
		}
	}
	if observation.UsageState == "unknown" && (observation.PromptTokens != nil || observation.CompletionTokens != nil || observation.TotalTokens != nil) {
		observation.UsageState = "partial"
	}
	if observation.UsageState == "complete" {
		valid := observation.PromptTokens != nil && observation.TotalTokens != nil
		if operation == "chat" {
			valid = valid && observation.CompletionTokens != nil && *observation.TotalTokens == *observation.PromptTokens+*observation.CompletionTokens
		} else {
			valid = valid && *observation.TotalTokens == *observation.PromptTokens
		}
		if !valid {
			observation.UsageState = "invalid"
		}
	}
	return observation, nil
}

func (m *UsageStore) Finish(ctx context.Context, attempt string, observation llm.UsageObservation) error {
	var row AiUsageCall
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if err := s.QueryRowCtx(ctx, &row, "SELECT "+aiUsageCallRows+" FROM ai_usage_call WHERE attempt_id=? FOR UPDATE", attempt); err != nil {
			return err
		}
		// 重复/迟到结算不增加账本，不覆盖已知用量，更不能把已发请求改成未发送退款。
		if row.Status != "reserved" && row.Status != "running" {
			return nil
		}
		var err error
		observation, err = normalizeUsageObservation(row.Operation, observation)
		if err != nil {
			return err
		}
		confirmed := row.DispatchConfirmed != 0 || observation.DispatchConfirmed
		if observation.Status == "not_sent" && confirmed {
			return ErrUsageInvalid
		}
		charged := observation.Status != "not_sent"
		_, err = s.ExecCtx(ctx, "UPDATE ai_usage_call SET status=?,usage_state=?,quota_charged=?,dispatch_confirmed=?,dispatched_at=IF(?=1,COALESCE(dispatched_at,NOW(3)),dispatched_at),prompt_tokens=?,completion_tokens=?,total_tokens=?,http_status=?,provider_request_id=?,error_code=?,completed_at=NOW(3) WHERE attempt_id=?", observation.Status, observation.UsageState, charged, confirmed, confirmed, observation.PromptTokens, observation.CompletionTokens, observation.TotalTokens, observation.HTTPStatus, observation.ProviderRequestID, observation.ErrorCode, attempt)
		return err
	})
	if err != nil {
		return err
	}
	return usageCommittedError(m.cached.DelCacheCtx(ctx, m.usageCallKeys(int64(row.Id), attempt, int64(row.OwnerId))...))
}

var usageTimezone = time.FixedZone("Asia/Shanghai", 8*60*60)

func usagePeriod(from, to string, now time.Time) (string, string, int64, int64, error) {
	if to == "" {
		to = now.In(usageTimezone).Format("2006-01-02")
	}
	end, err := time.ParseInLocation("2006-01-02", to, usageTimezone)
	if err != nil || end.Format("2006-01-02") != to {
		return "", "", 0, 0, ErrUsageInvalid
	}
	if from == "" {
		from = end.AddDate(0, 0, -6).Format("2006-01-02")
	}
	start, err := time.ParseInLocation("2006-01-02", from, usageTimezone)
	if err != nil || start.Format("2006-01-02") != from || start.After(end) || end.Sub(start) > 30*24*time.Hour {
		return "", "", 0, 0, ErrUsageInvalid
	}
	return from, to, start.Unix(), end.AddDate(0, 0, 1).Unix(), nil
}

func usageOneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func usageWhere(f UsageFilter) (UsageFilter, string, []any, error) {
	if f.OwnerId < 0 || !f.AllUsers && f.OwnerId == 0 || f.Page < 0 || f.PageSize < 0 || f.PageSize > 100 ||
		!usageOneOf(f.Source, "", "chat", "analysis", "kb_search", "kb_ingest", "evaluation") || !usageOneOf(f.Operation, "", "chat", "embedding") ||
		!usageOneOf(f.Status, "", "reserved", "running", "succeeded", "failed", "cancelled", "unknown", "not_sent") || !usageOneOf(f.UsageState, "", "unknown", "partial", "complete", "invalid") {
		return f, "", nil, ErrUsageInvalid
	}
	f.Provider, f.Model = strings.TrimSpace(f.Provider), strings.TrimSpace(f.Model)
	if !utf8.ValidString(f.Provider+f.Model) || utf8.RuneCountInString(f.Provider) > 32 || utf8.RuneCountInString(f.Model) > 128 {
		return f, "", nil, ErrUsageInvalid
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page-1 > math.MaxInt64/f.PageSize {
		return f, "", nil, ErrUsageInvalid
	}
	from, to, start, end, err := usagePeriod(f.FromDay, f.ToDay, time.Now())
	if err != nil {
		return f, "", nil, err
	}
	f.FromDay, f.ToDay = from, to
	where, args := " WHERE c.created_at>=FROM_UNIXTIME(?) AND c.created_at<FROM_UNIXTIME(?)", []any{start, end}
	if f.OwnerId > 0 {
		where += " AND c.owner_id=?"
		args = append(args, f.OwnerId)
	}
	for _, field := range []struct{ name, value string }{{"source", f.Source}, {"operation", f.Operation}, {"provider", f.Provider}, {"model", f.Model}, {"status", f.Status}, {"usage_state", f.UsageState}} {
		if field.value != "" {
			where += " AND c." + field.name + "=?"
			args = append(args, field.value)
		}
	}
	return f, where, args, nil
}

const usageKnownTotal = "CASE WHEN c.usage_state IN ('complete','partial') THEN COALESCE(c.total_tokens,COALESCE(c.prompt_tokens,0)+COALESCE(c.completion_tokens,0)) ELSE 0 END"
const usageUncertain = "(c.status='unknown' OR (c.dispatch_confirmed=0 AND c.quota_charged=1 AND c.status IN ('succeeded','failed','cancelled')))"
const usageMetered = "(c.dispatch_confirmed=1 OR " + usageUncertain + ")"
const usageAggregate = `COALESCE(SUM(c.dispatch_confirmed=1),0) AS confirmed_requests,
COALESCE(SUM(c.status='reserved'),0) AS reserved_requests,
COALESCE(SUM(c.status='not_sent'),0) AS not_sent_requests,
COALESCE(SUM(` + usageUncertain + `),0) AS uncertain_requests,
COALESCE(SUM(c.status='succeeded'),0) AS succeeded_requests,
COALESCE(SUM(c.status='failed'),0) AS failed_requests,
COALESCE(SUM(c.status='cancelled'),0) AS cancelled_requests,
COALESCE(SUM(` + usageMetered + ` AND c.usage_state='complete'),0) AS complete_requests,
COALESCE(SUM(` + usageMetered + ` AND c.usage_state='partial'),0) AS partial_requests,
COALESCE(SUM(` + usageMetered + ` AND c.usage_state='unknown'),0) AS unknown_requests,
COALESCE(SUM(` + usageMetered + ` AND c.usage_state='invalid'),0) AS invalid_requests,
COALESCE(SUM(CASE WHEN c.operation='chat' AND c.usage_state IN ('complete','partial') THEN COALESCE(c.prompt_tokens,0) ELSE 0 END),0) AS known_prompt_tokens,
COALESCE(SUM(CASE WHEN c.operation='chat' AND c.usage_state IN ('complete','partial') THEN COALESCE(c.completion_tokens,0) ELSE 0 END),0) AS known_completion_tokens,
COALESCE(SUM(CASE WHEN c.operation='embedding' THEN ` + usageKnownTotal + ` ELSE 0 END),0) AS known_embedding_tokens,
COALESCE(SUM(` + usageKnownTotal + `),0) AS known_total_tokens`

const usageCallSelect = `SELECT c.id,c.attempt_id,c.owner_id,c.source,c.operation,c.request_id,c.source_id,c.step_index,c.batch_index,c.input_items,c.provider,c.model,DATE_FORMAT(c.quota_day,'%Y-%m-%d') AS quota_day,c.quota_charged,c.dispatch_confirmed,c.status,c.usage_state,c.prompt_tokens,c.completion_tokens,c.total_tokens,c.http_status,c.provider_request_id,c.error_code,FLOOR(UNIX_TIMESTAMP(c.started_at)) AS started_at,COALESCE(FLOOR(UNIX_TIMESTAMP(c.dispatched_at)),0) AS dispatched_at,COALESCE(FLOOR(UNIX_TIMESTAMP(c.completed_at)),0) AS completed_at FROM ai_usage_call c`

func (m *UsageStore) Summary(ctx context.Context, filter UsageFilter) (*UsageSummary, error) {
	f, where, args, err := usageWhere(filter)
	if err != nil {
		return nil, err
	}
	if err := m.expire(ctx, f.OwnerId); err != nil {
		return nil, err
	}
	summary := &UsageSummary{FromDay: f.FromDay, ToDay: f.ToDay, Timezone: "Asia/Shanghai", Groups: make([]*UsageGroup, 0)}
	if err := m.conn.QueryRowCtx(ctx, &summary.RecordingStartedAt, "SELECT FLOOR(UNIX_TIMESTAMP(created_at)) FROM ai_usage_policy WHERE owner_id=0"); err != nil {
		return nil, err
	}
	if err := m.conn.QueryRowCtx(ctx, &summary.UsageCounters, "SELECT "+usageAggregate+" FROM ai_usage_call c"+where, args...); err != nil {
		return nil, err
	}
	if err := m.conn.QueryRowsCtx(ctx, &summary.Groups, "SELECT c.source,c.operation,c.provider,c.model,"+usageAggregate+" FROM ai_usage_call c"+where+" GROUP BY c.source,c.operation,c.provider,c.model ORDER BY c.source,c.operation,c.provider,c.model", args...); err != nil {
		return nil, err
	}
	return summary, nil
}

func (m *UsageStore) List(ctx context.Context, filter UsageFilter) ([]*UsageCallRow, int64, error) {
	f, where, args, err := usageWhere(filter)
	if err != nil {
		return nil, 0, err
	}
	if err := m.expire(ctx, f.OwnerId); err != nil {
		return nil, 0, err
	}
	var total int64
	if err := m.conn.QueryRowCtx(ctx, &total, "SELECT COUNT(*) FROM ai_usage_call c"+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*UsageCallRow, 0)
	err = m.conn.QueryRowsCtx(ctx, &rows, usageCallSelect+where+" ORDER BY c.id DESC LIMIT ? OFFSET ?", append(args, f.PageSize, (f.Page-1)*f.PageSize)...)
	return rows, total, err
}

// Expire 只收敛过期预留，不重放外部调用；未知预留继续扣住当日额度。
func (m *UsageStore) Expire(ctx context.Context) error { return m.expire(ctx, 0) }

func (m *UsageStore) expire(ctx context.Context, ownerId int64) error {
	var rows []struct {
		Id        int64  `db:"id"`
		AttemptId string `db:"attempt_id"`
		OwnerId   int64  `db:"owner_id"`
	}
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		where, args := " WHERE status IN ('reserved','running') AND lease_until<=NOW(3)", []any{}
		if ownerId > 0 {
			where += " AND owner_id=?"
			args = append(args, ownerId)
		}
		if err := s.QueryRowsCtx(ctx, &rows, "SELECT id,attempt_id,owner_id FROM ai_usage_call"+where+" ORDER BY id LIMIT 500 FOR UPDATE", args...); err != nil {
			return err
		}
		for _, row := range rows {
			if _, err := s.ExecCtx(ctx, "UPDATE ai_usage_call SET status='unknown',error_code='lease_expired',completed_at=NOW(3) WHERE id=? AND status IN ('reserved','running') AND lease_until<=NOW(3)", row.Id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(rows)*3)
	for _, row := range rows {
		keys = append(keys, m.usageCallKeys(row.Id, row.AttemptId, row.OwnerId)...)
	}
	if len(keys) == 0 {
		return nil
	}
	return usageCommittedError(m.cached.DelCacheCtx(ctx, keys...))
}

func (m *UsageStore) GetPolicy(ctx context.Context, ownerId int64) (*UsagePolicyView, error) {
	if ownerId < 0 {
		return nil, ErrUsageInvalid
	}
	if err := m.expire(ctx, ownerId); err != nil {
		return nil, err
	}
	var global, local AiUsagePolicy
	if err := m.conn.QueryRowCtx(ctx, &global, "SELECT "+aiUsagePolicyRows+" FROM ai_usage_policy WHERE owner_id=0"); err != nil {
		return nil, err
	}
	local = global
	if ownerId > 0 {
		if err := m.conn.QueryRowCtx(ctx, &local, "SELECT "+aiUsagePolicyRows+" FROM ai_usage_policy WHERE owner_id=?", ownerId); errors.Is(err, ErrNotFound) {
			local = AiUsagePolicy{OwnerId: uint64(ownerId)}
		} else if err != nil {
			return nil, err
		}
	}
	limit := local.DailyRequestLimit
	inherited := ownerId > 0 && !limit.Valid
	if inherited {
		limit = global.DailyRequestLimit
	}
	if !limit.Valid || limit.Int64 < 0 || global.GlobalConcurrentLimit < 1 || global.GlobalConcurrentLimit > 64 {
		return nil, ErrUsageInvalid
	}
	view := &UsagePolicyView{OwnerId: ownerId, DailyRequestLimit: limit.Int64, Unlimited: limit.Int64 == 0, Inherited: inherited, DailyTokenWarning: int64(local.DailyTokenWarning), Version: int64(local.Version), GlobalConcurrentLimit: int64(global.GlobalConcurrentLimit)}
	if err := m.conn.QueryRowCtx(ctx, &view.QuotaDay, "SELECT DATE_FORMAT(UTC_TIMESTAMP()+INTERVAL 8 HOUR,'%Y-%m-%d')"); err != nil {
		return nil, err
	}
	where, args := " WHERE c.quota_day=?", []any{view.QuotaDay}
	if ownerId > 0 {
		where += " AND c.owner_id=?"
		args = append(args, ownerId)
	}
	var stats struct {
		UsedRequests       int64 `db:"used_requests"`
		KnownTokens        int64 `db:"known_tokens"`
		IncompleteRequests int64 `db:"incomplete_requests"`
	}
	if err := m.conn.QueryRowCtx(ctx, &stats, "SELECT COALESCE(SUM(c.quota_charged=1),0) AS used_requests,COALESCE(SUM("+usageKnownTotal+"),0) AS known_tokens,COALESCE(SUM("+usageMetered+" AND c.usage_state<>'complete'),0) AS incomplete_requests FROM ai_usage_call c"+where, args...); err != nil {
		return nil, err
	}
	view.UsedRequests, view.KnownTokens, view.IncompleteRequests = stats.UsedRequests, stats.KnownTokens, stats.IncompleteRequests
	if ownerId > 0 && !view.Unlimited {
		view.RemainingRequests = max(int64(0), view.DailyRequestLimit-view.UsedRequests)
	}
	view.WarningReached = view.DailyTokenWarning > 0 && view.KnownTokens >= view.DailyTokenWarning
	return view, nil
}

func (m *UsageStore) GlobalConcurrentLimit(ctx context.Context) (int64, error) {
	var limit int64
	if err := m.conn.QueryRowCtx(ctx, &limit, "SELECT global_concurrent_limit FROM ai_usage_policy WHERE owner_id=0"); err != nil {
		return 0, err
	}
	if limit < 1 || limit > 64 {
		return 0, ErrUsageInvalid
	}
	return limit, nil
}

// Active 返回与最终容量准入同口径的MySQL活跃预留，不依赖可能丢失的Redis key。
func (m *UsageStore) Active(ctx context.Context) (int64, error) {
	var active int64
	err := m.conn.QueryRowCtx(ctx, &active, "SELECT COUNT(*) FROM ai_usage_call WHERE status IN ('reserved','running') AND lease_until>NOW(3)")
	return active, err
}

func (m *UsageStore) SetPolicy(ctx context.Context, ownerId, expectedVersion int64, patch UsagePolicyPatch) (*UsagePolicyView, error) {
	if ownerId < 0 || expectedVersion < 0 || patch.UpdatedBy < 0 || patch.DailyRequestLimit != nil && *patch.DailyRequestLimit < 0 || patch.DailyTokenWarning != nil && *patch.DailyTokenWarning < 0 ||
		patch.GlobalConcurrentLimit != nil && (ownerId != 0 || *patch.GlobalConcurrentLimit < 1 || *patch.GlobalConcurrentLimit > 64) || patch.InheritDefault && (ownerId == 0 || patch.DailyRequestLimit != nil) ||
		patch.DailyRequestLimit == nil && patch.DailyTokenWarning == nil && patch.GlobalConcurrentLimit == nil && !patch.InheritDefault {
		return nil, ErrUsageInvalid
	}
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		var row AiUsagePolicy
		err := s.QueryRowCtx(ctx, &row, "SELECT "+aiUsagePolicyRows+" FROM ai_usage_policy WHERE owner_id=0 FOR UPDATE")
		if err != nil {
			return err
		}
		if ownerId > 0 {
			row = AiUsagePolicy{}
			err = s.QueryRowCtx(ctx, &row, "SELECT "+aiUsagePolicyRows+" FROM ai_usage_policy WHERE owner_id=? FOR UPDATE", ownerId)
		}
		missing := errors.Is(err, ErrNotFound)
		if err != nil && !missing {
			return err
		}
		if missing {
			if ownerId == 0 {
				return ErrUsageInvalid
			}
			row.OwnerId, row.GlobalConcurrentLimit = uint64(ownerId), 8
		}
		if int64(row.Version) != expectedVersion {
			return ErrUsageConflict
		}
		if patch.InheritDefault {
			row.DailyRequestLimit = sql.NullInt64{}
		} else if patch.DailyRequestLimit != nil {
			row.DailyRequestLimit = sql.NullInt64{Int64: *patch.DailyRequestLimit, Valid: true}
		}
		if patch.DailyTokenWarning != nil {
			row.DailyTokenWarning = uint64(*patch.DailyTokenWarning)
		}
		if patch.GlobalConcurrentLimit != nil {
			row.GlobalConcurrentLimit = uint64(*patch.GlobalConcurrentLimit)
		}
		row.Version++
		row.UpdatedBy = sql.NullInt64{Int64: patch.UpdatedBy, Valid: patch.UpdatedBy > 0}
		if missing {
			_, err = s.ExecCtx(ctx, "INSERT INTO ai_usage_policy(owner_id,daily_request_limit,daily_token_warning,global_concurrent_limit,version,updated_by) VALUES (?,?,?,?,?,?)", row.OwnerId, row.DailyRequestLimit, row.DailyTokenWarning, row.GlobalConcurrentLimit, row.Version, row.UpdatedBy)
		} else {
			_, err = s.ExecCtx(ctx, "UPDATE ai_usage_policy SET daily_request_limit=?,daily_token_warning=?,global_concurrent_limit=?,version=?,updated_by=? WHERE owner_id=?", row.DailyRequestLimit, row.DailyTokenWarning, row.GlobalConcurrentLimit, row.Version, row.UpdatedBy, ownerId)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := m.cached.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiUsagePolicyOwnerIdPrefix, ownerId)); err != nil {
		return nil, usageCommittedError(err)
	}
	return m.GetPolicy(ctx, ownerId)
}

func (m *UsageStore) ListPolicies(ctx context.Context, page, size, userId int64) ([]*UsagePolicyView, int64, error) {
	if page < 0 || size < 0 || size > 100 || userId < 0 {
		return nil, 0, ErrUsageInvalid
	}
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	if page-1 > math.MaxInt64/size {
		return nil, 0, ErrUsageInvalid
	}
	where, args := "", []any{}
	if userId > 0 {
		where, args = " WHERE owner_id=?", []any{userId}
	}
	var count int64
	if err := m.conn.QueryRowCtx(ctx, &count, "SELECT COUNT(*) FROM ai_usage_policy"+where, args...); err != nil {
		return nil, 0, err
	}
	var owners []int64
	if err := m.conn.QueryRowsCtx(ctx, &owners, "SELECT owner_id FROM ai_usage_policy"+where+" ORDER BY owner_id LIMIT ? OFFSET ?", append(args, size, (page-1)*size)...); err != nil {
		return nil, 0, err
	}
	rows := make([]*UsagePolicyView, 0, len(owners))
	for _, owner := range owners {
		row, err := m.GetPolicy(ctx, owner)
		if err != nil {
			return nil, 0, err
		}
		rows = append(rows, row)
	}
	return rows, count, nil
}

func (m *UsageStore) IsAdmin(ctx context.Context, userId int64) (bool, error) {
	if userId <= 0 {
		return false, nil
	}
	var count int64
	err := m.conn.QueryRowCtx(ctx, &count, "SELECT COUNT(*) FROM casbin_rule g JOIN sys_role r ON r.code=g.v1 JOIN sys_user u ON g.v0=CONCAT('user:',u.id) WHERE g.ptype='g' AND g.v2='*' AND r.code='super_admin' AND r.status=1 AND r.deleted_at IS NULL AND u.status=1 AND u.deleted_at IS NULL AND u.id=?", userId)
	return count > 0, err
}
