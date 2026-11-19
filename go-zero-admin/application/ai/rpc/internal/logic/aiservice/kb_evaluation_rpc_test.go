package aiservicelogic

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

func TestEvaluationRPCRequiresActorBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	service := &svc.ServiceContext{}
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"datasets", func() error {
			_, err := NewListKbEvaluationDatasetsLogic(ctx, service).ListKbEvaluationDatasets(&aipb.ListKbEvaluationDatasetsRequest{KbId: 1})
			return err
		}},
		{"start", func() error {
			_, err := NewStartKbEvaluationLogic(ctx, service).StartKbEvaluation(&aipb.StartKbEvaluationRequest{KbId: 1})
			return err
		}},
		{"list", func() error {
			_, err := NewListKbEvaluationsLogic(ctx, service).ListKbEvaluations(&aipb.ListKbEvaluationsRequest{KbId: 1})
			return err
		}},
		{"get", func() error {
			_, err := NewGetKbEvaluationLogic(ctx, service).GetKbEvaluation(&aipb.GetKbEvaluationRequest{KbId: 1, EvaluationId: 1})
			return err
		}},
		{"stop", func() error {
			_, err := NewStopKbEvaluationLogic(ctx, service).StopKbEvaluation(&aipb.StopKbEvaluationRequest{KbId: 1, EvaluationId: 1})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.run() == nil {
				t.Fatal("未登录不得进入评测依赖或返回空成功")
			}
		})
	}
}

type evaluationOwnedKb struct{ model.AiKbModel }

func TestEvaluationDatasetCatalogueProjections(t *testing.T) {
	l := NewListKbEvaluationDatasetsLogic(context.Background(), &svc.ServiceContext{AiKbModel: &evaluationOwnedKb{}, AiKbEvaluationModel: &evaluationReplayStore{}})
	req := &aipb.ListKbEvaluationDatasetsRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 5}
	legacy, err := l.ListKbEvaluationDatasets(req)
	if err != nil || len(legacy.List) == 0 {
		t.Fatal("初始化内置题集", err)
	}
	req.MetaOnly = true
	meta, err := l.ListKbEvaluationDatasets(req)
	if err != nil || len(meta.List) != len(legacy.List) {
		t.Fatal("目录缺失", err)
	}
	for i, item := range meta.List {
		original := legacy.List[i]
		if item.Dataset != "" || item.Name != original.Name || item.Key != original.Key || item.Sha256 != fmt.Sprintf("%x", sha256.Sum256([]byte(original.Dataset))) {
			t.Fatal("目录指纹或数据投影错误")
		}
		req.MetaOnly = false
		req.DatasetKey = item.Key
		detail, err := l.ListKbEvaluationDatasets(req)
		if err != nil || len(detail.List) != 1 || detail.List[0].Dataset != original.Dataset {
			t.Fatal("选中题集原始字节不符", err)
		}
	}
	t.Logf("目录 protobuf 字节数：完整 %d，轻量 %d", proto.Size(legacy), proto.Size(meta))
	req.DatasetKey = "missing"
	_, err = l.ListKbEvaluationDatasets(req)
	assertWorkspaceCode(t, err, 400)
	req.Actor.UserId = 8
	_, err = l.ListKbEvaluationDatasets(req)
	assertWorkspaceCode(t, err, 404)
}

func (k *evaluationOwnedKb) RequireOwned(ctx context.Context, kb, owner int64) error {
	_, err := k.FindOwned(ctx, kb, owner)
	return err
}

func (*evaluationOwnedKb) FindOwned(_ context.Context, kb, owner int64) (*model.KbRow, error) {
	if kb != 5 || owner != 7 {
		return nil, model.ErrNotFound
	}
	return &model.KbRow{Id: 5, OwnerId: 7}, nil
}

type evaluationExpiringList struct {
	model.AiKbEvaluationModel
	row   model.KbEvaluationRow
	reads int
}

func (s *evaluationExpiringList) ListOwned(_ context.Context, _, _, _, _ int64, status, _ string) ([]*model.KbEvaluationRow, int64, error) {
	s.reads++
	if s.row.Status != status && status != "all" {
		return []*model.KbEvaluationRow{}, 0, nil
	}
	copy := s.row
	return []*model.KbEvaluationRow{&copy}, 1, nil
}
func (s *evaluationExpiringList) ExpireOwned(context.Context, int64, int64, int64) (*model.KbEvaluationRow, error) {
	s.row.Status = "unknown"
	copy := s.row
	return &copy, nil
}
func TestEvaluationRPCExpiredFilterKeepsTotalConsistent(t *testing.T) {
	store := &evaluationExpiringList{row: model.KbEvaluationRow{Id: 1, OwnerId: 7, KbId: 5, Status: "running", DatabaseTime: 1000, LeaseExpiresAt: 999}}
	service := &svc.ServiceContext{AiKbModel: &evaluationOwnedKb{}, AiKbEvaluationModel: store}
	result, err := NewListKbEvaluationsLogic(context.Background(), service).ListKbEvaluations(&aipb.ListKbEvaluationsRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 5, Status: "running"})
	if err != nil || result.Page.Total != 0 || len(result.List) != 0 || store.reads != 2 {
		t.Fatalf("过期变状态后须重查当前筛选和总数: %+v reads=%d err=%v", result, store.reads, err)
	}
}

type evaluationReplayStore struct {
	model.AiKbEvaluationModel
	row   *model.KbEvaluationRow
	reads int
}

type evaluationProgressFixture struct {
	evaluationExpiringList
}

func (s *evaluationProgressFixture) ProgressOwned(_ context.Context, owner, kb, id int64, _ string) (*model.KbEvaluationRow, error) {
	if s.row.OwnerId != owner || s.row.KbId != kb || s.row.Id != id {
		return nil, model.ErrNotFound
	}
	s.reads++
	row := s.row
	return &row, nil
}

func TestEvaluationRPCProgressChecksLeaseOwnerAndDigest(t *testing.T) {
	store := &evaluationProgressFixture{evaluationExpiringList{row: model.KbEvaluationRow{Id: 1, KbId: 5, OwnerId: 7, Status: "running", DatabaseTime: 50, LeaseExpiresAt: 49}}}
	l := NewGetKbEvaluationLogic(context.Background(), &svc.ServiceContext{AiKbModel: &evaluationOwnedKb{}, AiKbEvaluationModel: store})
	req := &aipb.GetKbEvaluationRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 5, EvaluationId: 1, ProgressOnly: true}
	result, err := l.GetKbEvaluation(req)
	if err != nil || result.Evaluation.Status != "unknown" || store.reads != 2 || result.Evaluation.Snapshot != "" {
		t.Fatal("轻量轮询应处理租约过期且只返回摘要", err)
	}
	req.KnownObservationsSha256 = "invalid"
	_, err = l.GetKbEvaluation(req)
	assertWorkspaceCode(t, err, 400)
	req.KnownObservationsSha256 = ""
	req.Actor.UserId = 8
	_, err = l.GetKbEvaluation(req)
	assertWorkspaceCode(t, err, 404)
	if store.reads != 2 {
		t.Fatal("无效指纹或非归属用户不可触达进度")
	}
}

func (s *evaluationReplayStore) FindRequestOwned(context.Context, int64, string) (*model.KbEvaluationRow, error) {
	s.reads++
	return s.row, nil
}
func TestEvaluationRPCReplaysBeforeCurrentConfigurationAndSlots(t *testing.T) {
	request := &aipb.StartKbEvaluationRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 5, RequestId: uuid.NewString(), DatasetKey: "removed-historical-dataset", DatasetSha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", GroupKey: "ops", CaseIds: []string{"one"}}
	signature, err := evaluationRequestSignature(request)
	if err != nil {
		t.Fatal(err)
	}
	store := &evaluationReplayStore{row: &model.KbEvaluationRow{Id: 9, OwnerId: 7, KbId: 5, RequestId: request.RequestId, RequestSignature: signature, Status: "failed"}}
	service := &svc.ServiceContext{AiKbModel: &evaluationOwnedKb{}, AiKbEvaluationModel: store}
	for i := 0; i < 2; i++ {
		if !reserveEvaluationSlot() {
			t.Fatal("fixture slots occupied")
		}
		defer releaseEvaluationSlot()
	}
	result, err := NewStartKbEvaluationLogic(context.Background(), service).StartKbEvaluation(request)
	if err != nil || result.Evaluation.Id != 9 || result.Evaluation.Status != "failed" || store.reads != 1 {
		t.Fatal("已接受原请求的读取不依赖当前题集/模型/执行槽，也不重放失败")
	}
	changed := proto.Clone(request).(*aipb.StartKbEvaluationRequest)
	changed.CaseIds = []string{"changed"}
	_, err = NewStartKbEvaluationLogic(context.Background(), service).StartKbEvaluation(changed)
	assertWorkspaceCode(t, err, 409)
}
func TestEvaluationRPCCrossIdentityCannotReadBatch(t *testing.T) {
	store := &evaluationReplayStore{row: &model.KbEvaluationRow{Id: 9, OwnerId: 7, KbId: 5, Status: "succeeded"}}
	service := &svc.ServiceContext{AiKbModel: &evaluationOwnedKb{}, AiKbEvaluationModel: store}
	_, err := NewGetKbEvaluationLogic(context.Background(), service).GetKbEvaluation(&aipb.GetKbEvaluationRequest{Actor: &aipb.Actor{UserId: 8}, KbId: 5, EvaluationId: 9})
	assertWorkspaceCode(t, err, 404)
	_, err = NewStopKbEvaluationLogic(context.Background(), service).StopKbEvaluation(&aipb.StopKbEvaluationRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 6, EvaluationId: 9})
	assertWorkspaceCode(t, err, 404)
	if store.reads != 0 {
		t.Fatal("跨身份或错KB不能读取评测依赖")
	}
}
