package model

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrEvaluationState     = errors.New("evaluation state changed")
	ErrEvaluationConflict  = errors.New("evaluation request changed")
	ErrEvaluationStopped   = errors.New("evaluation stop requested")
	ErrEvaluationCommitted = errors.New("evaluation committed; cache invalidation failed")
)

type EvaluationObservation struct {
	CaseId            string          `json:"case_id"`
	RequestId         string          `json:"request_id"`
	TaskId            int64           `json:"task_id"`
	Status            string          `json:"status"`
	Stage             string          `json:"stage"`
	Query             string          `json:"query"`
	List              json.RawMessage `json:"list"`
	SearchClientMs    int64           `json:"search_client_ms"`
	SearchServerMs    int64           `json:"search_server_ms"`
	AnalysisElapsedMs int64           `json:"analysis_elapsed_ms"`
	Analysis          json.RawMessage `json:"analysis,omitempty"`
	Error             string          `json:"error"`
}

type EvaluationCorpusDocument struct {
	Id             int64  `db:"id"`
	KbId           int64  `db:"kb_id"`
	Title          string `db:"title"`
	Version        int64  `db:"version"`
	Status         string `db:"status"`
	EmbeddingModel string `db:"embedding_model"`
	ContentSHA     string `db:"content_sha256"`
}

type KbEvaluationRow struct {
	Id               int64  `db:"id"`
	OwnerId          int64  `db:"owner_id"`
	KbId             int64  `db:"kb_id"`
	RequestId        string `db:"request_id"`
	RequestSignature string `db:"request_signature"`
	DatasetKey       string `db:"dataset_key"`
	DatasetSHA       string `db:"dataset_sha256"`
	GroupKey         string `db:"group_key"`
	Status           string `db:"status"`
	CaseCount        int64  `db:"case_count"`
	CompletedCount   int64  `db:"completed_count"`
	FailedCount      int64  `db:"failed_count"`
	StopRequested    bool   `db:"stop_requested"`
	CorpusVerified   bool   `db:"corpus_verified"`
	Error            string `db:"error"`
	Snapshot         string `db:"dataset_snapshot"`
	Bindings         string `db:"bindings"`
	Context          string `db:"run_context"`
	Observations     string `db:"observations"`
	LeaseToken       string `db:"lease_token"`
	LeaseExpiresAt   int64  `db:"lease_expires_at"`
	StartedAt        int64  `db:"started_at"`
	CompletedAt      int64  `db:"completed_at"`
	CreatedAt        int64  `db:"created_at"`
	UpdatedAt        int64  `db:"updated_at"`
	DatabaseTime     int64  `db:"database_time"`
}

func validEvaluationObservation(row EvaluationObservation) bool {
	if row.CaseId == "" || len(row.CaseId) > 128 || !utf8.ValidString(row.CaseId) || !utf8.ValidString(row.Query) || strings.TrimSpace(row.Query) == "" || len(row.Query) > 2000 || row.TaskId < 0 || row.Stage == "" || len(row.Stage) > 64 || !utf8.ValidString(row.Error) || len(row.Error) > 1000 || row.SearchClientMs < 0 || row.SearchServerMs < 0 || row.AnalysisElapsedMs < 0 {
		return false
	}
	parsed, err := uuid.Parse(row.RequestId)
	if err != nil || parsed.String() != row.RequestId {
		return false
	}
	var hits []json.RawMessage
	list := strings.TrimSpace(string(row.List))
	if len(list) == 0 || list[0] != '[' || json.Unmarshal(row.List, &hits) != nil || len(hits) > 5 {
		return false
	}
	switch row.Status {
	case "waiting":
		return row.TaskId == 0 && len(row.Analysis) == 0
	case "running":
		return len(row.Analysis) == 0
	case "succeeded":
		return row.TaskId > 0 && len(row.Analysis) > 0 && json.Valid(row.Analysis) && strings.HasPrefix(strings.TrimSpace(string(row.Analysis)), "{")
	case "failed", "unknown":
		return strings.TrimSpace(row.Error) != "" && len(row.Analysis) == 0
	default:
		return false
	}
}

func decodeEvaluationObservations(raw string) ([]EvaluationObservation, error) {
	var rows []EvaluationObservation
	if len(raw) == 0 || len(raw) > 1<<20 || !utf8.ValidString(raw) || json.Unmarshal([]byte(raw), &rows) != nil || len(rows) < 1 || len(rows) > 30 {
		return nil, ErrEvaluationState
	}
	cases, requests := make(map[string]bool, len(rows)), make(map[string]bool, len(rows))
	for _, row := range rows {
		if !validEvaluationObservation(row) || cases[row.CaseId] || requests[row.RequestId] {
			return nil, ErrEvaluationState
		}
		cases[row.CaseId] = true
		requests[row.RequestId] = true
	}
	return rows, nil
}

func replaceEvaluationObservation(raw string, index int64, next EvaluationObservation, claim bool) (string, int64, int64, error) {
	rows, err := decodeEvaluationObservations(raw)
	if err != nil || index < 0 || index >= int64(len(rows)) || !validEvaluationObservation(next) {
		return "", 0, 0, ErrEvaluationState
	}
	old := rows[index]
	if old.CaseId != next.CaseId || old.RequestId != next.RequestId || old.Query != next.Query || (old.TaskId > 0 && old.TaskId != next.TaskId) {
		return "", 0, 0, ErrEvaluationState
	}
	if claim {
		if old.Status != "waiting" || next.Status != "running" || next.TaskId != 0 {
			return "", 0, 0, ErrEvaluationState
		}
		for i, row := range rows {
			if int64(i) < index && row.Status != "succeeded" {
				return "", 0, 0, ErrEvaluationState
			}
			if row.Status == "running" {
				return "", 0, 0, ErrEvaluationState
			}
		}
	} else if old.Status != "running" || next.Status == "waiting" {
		return "", 0, 0, ErrEvaluationState
	}
	rows[index] = next
	encoded, err := json.Marshal(rows)
	if err != nil || len(encoded) > 1<<20 {
		return "", 0, 0, ErrEvaluationState
	}
	var completed, failed int64
	for _, row := range rows {
		if row.Status == "succeeded" {
			completed++
		}
		if row.Status == "failed" || row.Status == "unknown" {
			failed++
		}
	}
	return string(encoded), completed, failed, nil
}
