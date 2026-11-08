package aiservicelogic

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/internal/model"
)

//go:embed evaluation_data/*.json
var evaluationData embed.FS

type evaluationDocument struct {
	Key              string `json:"key"`
	Kb               string `json:"kb"`
	Source           string `json:"source"`
	Title            string `json:"title"`
	SHA              string `json:"sha256"`
	SourceDocId      int64  `json:"source_doc_id,omitempty"`
	SourceDocVersion int64  `json:"source_doc_version,omitempty"`
	Content          string `json:"-"`
}
type evaluationCase struct {
	Id    string `json:"id"`
	Kb    string `json:"kb"`
	Query string `json:"query"`
}
type evaluationDataset struct {
	Version     int                  `json:"version"`
	Description string               `json:"description"`
	Documents   []evaluationDocument `json:"documents"`
	Cases       []evaluationCase     `json:"cases"`
	Raw         string               `json:"-"`
	SHA         string               `json:"-"`
}

var evaluationDatasetNames = map[string]string{"business-evidence-cases": "24题证据覆盖验收（校准）", "business-acceptance-cases": "24题场景验收", "business-cases": "32题历史业务", "business-failure-regression-cases": "14题失败回归"}
var evaluationDatasetKeys = []string{"business-evidence-cases", "business-acceptance-cases", "business-cases", "business-failure-regression-cases"}
var errEvaluationDataset = errors.New("evaluation dataset invalid")
var evaluationSHA = regexp.MustCompile(`^[a-f0-9]{64}$`)

func loadEvaluationDataset(key string) (*evaluationDataset, error) {
	if evaluationDatasetNames[key] == "" {
		return nil, errEvaluationDataset
	}
	raw, err := evaluationData.ReadFile("evaluation_data/" + key + ".json")
	if err != nil {
		return nil, err
	}
	var value evaluationDataset
	if len(raw) > 256<<10 || !utf8.Valid(raw) || json.Unmarshal(raw, &value) != nil || value.Version != 1 || len(value.Documents) < 1 || len(value.Documents) > 100 || len(value.Cases) < 1 || len(value.Cases) > 100 {
		return nil, errEvaluationDataset
	}
	docs, cases := make(map[string]bool, len(value.Documents)), make(map[string]bool, len(value.Cases))
	groups := make(map[string]bool)
	for _, doc := range value.Documents {
		if doc.Key == "" || docs[doc.Key] || doc.Kb == "" || len(doc.Kb) > 64 || strings.TrimSpace(doc.Title) == "" || !evaluationSHA.MatchString(doc.SHA) {
			return nil, errEvaluationDataset
		}
		docs[doc.Key] = true
		groups[doc.Kb] = true
	}
	for _, item := range value.Cases {
		if item.Id == "" || len(item.Id) > 128 || cases[item.Id] || !groups[item.Kb] || strings.TrimSpace(item.Query) == "" || len(item.Query) > 2000 {
			return nil, errEvaluationDataset
		}
		cases[item.Id] = true
	}
	sum := sha256.Sum256(raw)
	value.Raw = string(raw)
	value.SHA = hex.EncodeToString(sum[:])
	return &value, nil
}
func selectEvaluationCases(dataset *evaluationDataset, group string, ids []string) ([]evaluationCase, error) {
	if dataset == nil || group == "" || len(ids) < 1 || len(ids) > 30 {
		return nil, errEvaluationDataset
	}
	selected := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" || selected[id] {
			return nil, errEvaluationDataset
		}
		selected[id] = true
	}
	out := make([]evaluationCase, 0, len(ids))
	for _, item := range dataset.Cases {
		if selected[item.Id] {
			if item.Kb != group {
				return nil, errEvaluationDataset
			}
			out = append(out, item)
		}
	}
	if len(out) != len(ids) {
		return nil, errEvaluationDataset
	}
	return out, nil
}

type evaluationDocumentBinding struct {
	evaluationDocument
	DocId          int64  `json:"doc_id"`
	KbId           int64  `json:"kb_id"`
	Version        int64  `json:"version"`
	Status         string `json:"status"`
	EmbeddingModel string `json:"embedding_model"`
	ContentSHA     string `json:"content_sha256"`
}
type evaluationInventory struct {
	Kb             string `json:"kb"`
	Id             int64  `json:"id"`
	KbId           int64  `json:"kb_id"`
	Title          string `json:"title"`
	Version        int64  `json:"version"`
	Status         string `json:"status"`
	EmbeddingModel string `json:"embedding_model"`
}
type evaluationBindings struct {
	Documents []evaluationDocumentBinding `json:"documents"`
	Inventory []evaluationInventory       `json:"inventory"`
}

func buildEvaluationBindings(dataset *evaluationDataset, group string, kbId int64, rows []model.EvaluationCorpusDocument) (string, error) {
	if dataset == nil || group == "" || kbId <= 0 {
		return "", errEvaluationDataset
	}
	bindings := evaluationBindings{Documents: []evaluationDocumentBinding{}, Inventory: []evaluationInventory{}}
	titles := make(map[string][]model.EvaluationCorpusDocument, len(rows))
	byID := make(map[int64]model.EvaluationCorpusDocument, len(rows))
	seen := make(map[int64]bool, len(rows))
	for _, row := range rows {
		if row.Id <= 0 || seen[row.Id] || row.KbId != kbId || row.Version < 1 || strings.TrimSpace(row.Title) == "" || (row.Status != "ready" && row.Status != "pending" && row.Status != "failed") {
			return "", errEvaluationDataset
		}
		seen[row.Id] = true
		byID[row.Id] = row
		titles[row.Title] = append(titles[row.Title], row)
		bindings.Inventory = append(bindings.Inventory, evaluationInventory{Kb: group, Id: row.Id, KbId: kbId, Title: row.Title, Version: row.Version, Status: row.Status, EmbeddingModel: row.EmbeddingModel})
	}
	for _, doc := range dataset.Documents {
		if doc.Kb != group {
			continue
		}
		var row model.EvaluationCorpusDocument
		if doc.SourceDocId > 0 {
			var exists bool
			row, exists = byID[doc.SourceDocId]
			if !exists || doc.SourceDocVersion <= 0 || row.Version != doc.SourceDocVersion || row.Title != doc.Title {
				return "", errEvaluationDataset
			}
		} else {
			matches := titles[doc.Title]
			if len(matches) != 1 {
				return "", errEvaluationDataset
			}
			row = matches[0]
		}
		if row.Status != "ready" || row.EmbeddingModel == "" || row.ContentSHA != doc.SHA {
			return "", errEvaluationDataset
		}
		bindings.Documents = append(bindings.Documents, evaluationDocumentBinding{evaluationDocument: doc, DocId: row.Id, KbId: kbId, Version: row.Version, Status: row.Status, EmbeddingModel: row.EmbeddingModel, ContentSHA: row.ContentSHA})
	}
	if len(bindings.Documents) == 0 {
		return "", errEvaluationDataset
	}
	sort.Slice(bindings.Inventory, func(i, j int) bool { return bindings.Inventory[i].Id < bindings.Inventory[j].Id })
	raw, err := json.Marshal(bindings)
	if err != nil || len(raw) > 256<<10 {
		return "", errEvaluationDataset
	}
	return string(raw), nil
}
