package aiservicelogic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

var evaluationSlots = make(chan struct{}, 2)

func reserveEvaluationSlot() bool {
	select {
	case evaluationSlots <- struct{}{}:
		return true
	default:
		return false
	}
}
func releaseEvaluationSlot() { <-evaluationSlots }

func evaluationError(err error) error {
	switch {
	case errors.Is(err, model.ErrEvaluationConflict), errors.Is(err, model.ErrEvaluationState), errors.Is(err, model.ErrEvaluationStopped):
		return xerr.NewErrCodeMsg(409, "评测请求或状态已变化，请按原标识核对")
	case errors.Is(err, model.ErrEvaluationCommitted):
		return xerr.NewErrCodeMsg(503, "评测已保存，缓存或回读未确认，请使用原request_id核对，不要重复创建")
	default:
		return aiModelError(err)
	}
}
func checkEvaluationKb(ctx context.Context, service *svc.ServiceContext, actor *aipb.Actor, kb int64) error {
	if err := requireAiActor(actor); err != nil {
		return err
	}
	if kb <= 0 {
		return xerr.NewErrCode(400)
	}
	if service == nil || service.AiKbModel == nil || service.AiKbEvaluationModel == nil {
		return xerr.NewErrCodeMsg(503, "知识库评测暂不可用")
	}
	_, err := service.AiKbModel.FindOwned(ctx, kb, actor.UserId)
	if err != nil {
		return aiModelError(err)
	}
	return nil
}
func evaluationUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}
func evaluationRequestSignature(in *aipb.StartKbEvaluationRequest) (string, error) {
	if !evaluationUUID(in.RequestId) || len(in.DatasetKey) < 1 || len(in.DatasetKey) > 64 || !evaluationSHA.MatchString(in.DatasetSha256) || len(in.GroupKey) < 1 || len(in.GroupKey) > 64 || len(in.CaseIds) < 1 || len(in.CaseIds) > 30 {
		return "", xerr.NewErrCode(400)
	}
	if len(in.DatasetSnapshot) > 256<<10 || !utf8.ValidString(in.DatasetSnapshot) {
		return "", xerr.NewErrCode(400)
	}
	ids := append([]string(nil), in.CaseIds...)
	sort.Strings(ids)
	for i, id := range ids {
		if id == "" || len(id) > 128 || (i > 0 && ids[i-1] == id) {
			return "", xerr.NewErrCode(400)
		}
	}
	identity := map[string]any{"kb_id": in.KbId, "dataset_key": in.DatasetKey, "dataset_sha256": in.DatasetSha256, "group_key": in.GroupKey, "case_ids": ids}
	if in.DatasetSnapshot != "" {
		rawSHA := sha256.Sum256([]byte(in.DatasetSnapshot))
		identity["dataset_snapshot_sha256"] = hex.EncodeToString(rawSHA[:])
	}
	value, _ := json.Marshal(identity)
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:]), nil
}
func evaluationProto(row *model.KbEvaluationRow, summary bool) *aipb.KbEvaluationInfo {
	value := &aipb.KbEvaluationInfo{Id: row.Id, KbId: row.KbId, RequestId: row.RequestId, DatasetKey: row.DatasetKey, DatasetSha256: row.DatasetSHA, GroupKey: row.GroupKey, Status: row.Status, CaseCount: row.CaseCount, CompletedCount: row.CompletedCount, FailedCount: row.FailedCount, StopRequested: row.StopRequested, CorpusVerified: row.CorpusVerified, Error: row.Error, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, StartedAt: row.StartedAt, CompletedAt: row.CompletedAt}
	if !summary {
		value.Snapshot = row.Snapshot
		value.Bindings = row.Bindings
		value.Context = row.Context
		value.Observations = row.Observations
	}
	return value
}
func evaluationNeedsExpiry(row *model.KbEvaluationRow) bool {
	return (row.Status == "running" && row.LeaseExpiresAt <= row.DatabaseTime) || (row.Status == "pending" && row.UpdatedAt <= row.DatabaseTime-120)
}
func refreshEvaluationExpiry(ctx context.Context, service *svc.ServiceContext, row *model.KbEvaluationRow) (*model.KbEvaluationRow, error) {
	if !evaluationNeedsExpiry(row) {
		return row, nil
	}
	return service.AiKbEvaluationModel.ExpireOwned(ctx, row.OwnerId, row.KbId, row.Id)
}
func evaluationJSONEqual(left, right string) bool {
	var a, b any
	if json.Unmarshal([]byte(left), &a) != nil || json.Unmarshal([]byte(right), &b) != nil {
		return false
	}
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func evaluationStatus(value string) bool {
	switch value {
	case "all", "pending", "running", "succeeded", "failed", "cancelled", "unknown":
		return true
	default:
		return false
	}
}
func evaluationListStatus(value string) string {
	if strings.TrimSpace(value) == "" {
		return "all"
	}
	return value
}
