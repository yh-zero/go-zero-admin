package aiservicelogic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var customEvaluationField = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

// 自定义题集不是受信任内置文件；逐键验证后才允许Go结构解码，防大小写别名改变实际query。
func parseCustomEvaluationDataset(key, expectedSHA, raw string) (*evaluationDataset, error) {
	if len(raw) == 0 || len(raw) > 256<<10 || !utf8.ValidString(raw) || !evaluationSHA.MatchString(expectedSHA) {
		return nil, errEvaluationDataset
	}
	sum := sha256.Sum256([]byte(raw))
	actual := hex.EncodeToString(sum[:])
	if actual != expectedSHA || key != "custom-"+actual[:32] {
		return nil, errEvaluationDataset
	}
	parsed, err := strictEvaluationJSON(raw)
	if err != nil {
		return nil, errEvaluationDataset
	}
	root, ok := evaluationObject(parsed, []string{"version", "description", "annotation", "metadata", "documents", "cases"}, "version", "documents", "cases")
	if !ok || root["version"] != json.Number("1") {
		return nil, errEvaluationDataset
	}
	for _, field := range []string{"annotation", "metadata"} {
		if value, exists := root[field]; exists {
			if _, ok := value.(map[string]any); !ok {
				return nil, errEvaluationDataset
			}
		}
	}
	if value, exists := root["description"]; exists && !evaluationText(value, 256<<10) {
		return nil, errEvaluationDataset
	}
	documents, ok := root["documents"].([]any)
	if !ok || len(documents) < 1 || len(documents) > 100 {
		return nil, errEvaluationDataset
	}
	cases, ok := root["cases"].([]any)
	if !ok || len(cases) < 1 || len(cases) > 100 {
		return nil, errEvaluationDataset
	}
	var dataset evaluationDataset
	if json.Unmarshal([]byte(raw), &dataset) != nil {
		return nil, errEvaluationDataset
	}
	docs := make(map[string]evaluationDocument, len(documents))
	ids := make(map[int64]bool, len(documents))
	groups := make(map[string]bool)
	for i, value := range documents {
		fields, ok := evaluationObject(value, []string{"key", "kb", "source", "title", "sha256", "source_doc_id", "source_doc_version", "content"}, "key", "kb", "source", "title", "sha256", "source_doc_id", "source_doc_version", "content")
		doc := dataset.Documents[i]
		content, contentOK := fields["content"].(string)
		if !ok || !evaluationText(doc.Key, 128) || !evaluationText(doc.Kb, 64) || !evaluationText(doc.Title, 512) || doc.Source != "inline:"+doc.Key || !evaluationSHA.MatchString(doc.SHA) || doc.SourceDocId < 1 || doc.SourceDocId > 9007199254740991 || doc.SourceDocVersion < 1 || doc.SourceDocVersion > 9007199254740991 || ids[doc.SourceDocId] || !contentOK || !evaluationText(content, 65536) {
			return nil, errEvaluationDataset
		}
		if _, duplicate := docs[doc.Key]; duplicate {
			return nil, errEvaluationDataset
		}
		contentSHA := sha256.Sum256([]byte(content))
		if hex.EncodeToString(contentSHA[:]) != doc.SHA {
			return nil, errEvaluationDataset
		}
		doc.Content = content
		dataset.Documents[i] = doc
		docs[doc.Key], ids[doc.SourceDocId], groups[doc.Kb] = doc, true, true
	}
	seen := make(map[string]bool, len(cases))
	for _, value := range cases {
		if !validateCustomEvaluationCase(value, docs, groups, seen) {
			return nil, errEvaluationDataset
		}
	}
	dataset.Raw, dataset.SHA = raw, actual
	return &dataset, nil
}

func evaluationObject(value any, allowed []string, required ...string) (map[string]any, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	for key := range object {
		found := false
		for _, name := range allowed {
			found = found || key == name
		}
		if !found {
			return nil, false
		}
	}
	for _, key := range required {
		if object[key] == nil {
			return nil, false
		}
	}
	return object, true
}

func evaluationText(value any, limit int) bool {
	text, ok := value.(string)
	return ok && strings.TrimFunc(text, evaluationWhitespace) != "" && len(text) <= limit
}

func evaluationWhitespace(r rune) bool { return unicode.IsSpace(r) && r != '\u0085' || r == '\ufeff' }

func evaluationStrings(value any) ([]string, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	texts := make([]string, 0, len(items))
	for _, item := range items {
		if !evaluationText(item, 256<<10) {
			return nil, false
		}
		texts = append(texts, item.(string))
	}
	return texts, true
}

func customEvaluationGroups(value any) ([][]string, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	groups := make([][]string, 0, len(items))
	for _, item := range items {
		group, ok := evaluationStrings(item)
		if !ok || len(group) == 0 {
			return nil, false
		}
		groups = append(groups, group)
	}
	return groups, true
}

func normalizeEvaluationText(value string) string {
	return strings.Map(func(r rune) rune {
		if evaluationWhitespace(r) {
			return -1
		}
		return r
	}, norm.NFKC.String(value))
}

func customGroupsInTexts(groups [][]string, texts []string) bool {
	for _, group := range groups {
		found := false
		for _, alternative := range group {
			needle := normalizeEvaluationText(alternative)
			for _, text := range texts {
				found = found || strings.Contains(text, needle)
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func validateCustomEvaluationCase(value any, docs map[string]evaluationDocument, groups, seen map[string]bool) bool {
	row, ok := evaluationObject(value, []string{"id", "kb", "query", "split", "category", "rationale", "expected_documents", "evidence", "answer", "checks", "expected_answer_example", "source_case_id"}, "id", "kb", "query", "split", "category", "rationale", "expected_documents", "evidence", "answer")
	if !ok || !evaluationText(row["id"], 128) || !evaluationText(row["kb"], 64) || !evaluationText(row["query"], 2000) || !evaluationText(row["category"], 256<<10) || !evaluationText(row["rationale"], 256<<10) || (row["split"] != "development" && row["split"] != "validation") {
		return false
	}
	id, kb := row["id"].(string), row["kb"].(string)
	if seen[id] || !groups[kb] {
		return false
	}
	seen[id] = true
	for _, field := range []string{"expected_answer_example", "source_case_id"} {
		if text, exists := row[field]; exists && !evaluationText(text, 256<<10) {
			return false
		}
	}
	expected, ok := evaluationStrings(row["expected_documents"])
	if !ok {
		return false
	}
	expectedSet := make(map[string]bool, len(expected))
	texts := make([]string, 0, len(expected))
	for _, key := range expected {
		doc, exists := docs[key]
		if !exists || doc.Kb != kb || expectedSet[key] {
			return false
		}
		expectedSet[key] = true
		texts = append(texts, normalizeEvaluationText(doc.Content))
	}
	answer, ok := evaluationObject(row["answer"], []string{"kind", "required_groups", "forbidden_phrases", "absence_policy"}, "kind", "required_groups", "forbidden_phrases")
	if !ok || (answer["kind"] != "answerable" && answer["kind"] != "unanswerable") {
		return false
	}
	required, ok := customEvaluationGroups(answer["required_groups"])
	if !ok {
		return false
	}
	if _, ok := evaluationStrings(answer["forbidden_phrases"]); !ok {
		return false
	}
	if policy, exists := answer["absence_policy"]; exists && policy != "required" && policy != "forbidden" {
		return false
	}
	evidence, ok := row["evidence"].([]any)
	if !ok {
		return false
	}
	if answer["kind"] == "unanswerable" {
		if len(expected) != 0 || len(evidence) != 0 || len(required) != 0 {
			return false
		}
	} else if len(expected) == 0 || len(required) == 0 || len(evidence) != len(expected) || !customGroupsInTexts(required, texts) {
		return false
	}
	evidenceDocs := make(map[string]bool, len(evidence))
	for _, value := range evidence {
		item, ok := evaluationObject(value, []string{"document", "groups"}, "document", "groups")
		if !ok || !evaluationText(item["document"], 128) {
			return false
		}
		key := item["document"].(string)
		phrases, ok := customEvaluationGroups(item["groups"])
		if !ok || len(phrases) == 0 || !expectedSet[key] || evidenceDocs[key] || !customGroupsInTexts(phrases, []string{normalizeEvaluationText(docs[key].Content)}) {
			return false
		}
		evidenceDocs[key] = true
	}
	if value, exists := row["checks"]; exists {
		checks, ok := value.([]any)
		if !ok || len(checks) != len(required) {
			return false
		}
		checked := make(map[int64]bool, len(checks))
		for _, value := range checks {
			check, ok := evaluationObject(value, []string{"label", "kind", "required_group"}, "label", "kind", "required_group")
			if !ok || !evaluationText(check["label"], 256<<10) || (check["kind"] != "fact" && check["kind"] != "condition" && check["kind"] != "risk" && check["kind"] != "unknown") {
				return false
			}
			number, ok := check["required_group"].(json.Number)
			if !ok {
				return false
			}
			index, err := number.Int64()
			if err != nil || index < 0 || index >= int64(len(required)) || checked[index] {
				return false
			}
			checked[index] = true
		}
	}
	return true
}

func strictEvaluationJSON(raw string) (any, error) {
	if !json.Valid([]byte(raw)) || !validEvaluationSurrogates(raw) {
		return nil, errEvaluationDataset
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var read func(int) (any, error)
	read = func(depth int) (any, error) {
		if depth > 64 {
			return nil, errEvaluationDataset
		}
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		delim, object := token.(json.Delim)
		if !object {
			return token, nil
		}
		if delim == '{' {
			fields := map[string]any{}
			for decoder.More() {
				name, err := decoder.Token()
				key, ok := name.(string)
				if err != nil || !ok || !customEvaluationField.MatchString(key) || key == "constructor" || key == "prototype" {
					return nil, errEvaluationDataset
				}
				if _, duplicate := fields[key]; duplicate {
					return nil, errEvaluationDataset
				}
				value, err := read(depth + 1)
				if err != nil {
					return nil, err
				}
				fields[key] = value
			}
			if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
				return nil, errEvaluationDataset
			}
			return fields, nil
		}
		if delim == '[' {
			values := []any{}
			for decoder.More() {
				value, err := read(depth + 1)
				if err != nil {
					return nil, err
				}
				values = append(values, value)
			}
			if end, err := decoder.Token(); err != nil || end != json.Delim(']') {
				return nil, errEvaluationDataset
			}
			return values, nil
		}
		return nil, errEvaluationDataset
	}
	value, err := read(0)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errEvaluationDataset
	}
	return value, nil
}

// encoding/json会将孤立代理项替换为U+FFFD；提前拒绝以保持浏览器与后端的query字节一致。
func validEvaluationSurrogates(raw string) bool {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '"' {
			continue
		}
		for i++; i < len(raw) && raw[i] != '"'; i++ {
			if raw[i] != '\\' {
				continue
			}
			i++
			if i >= len(raw) {
				return false
			}
			if raw[i] != 'u' {
				continue
			}
			if i+4 >= len(raw) {
				return false
			}
			value, err := strconv.ParseUint(raw[i+1:i+5], 16, 16)
			if err != nil {
				return false
			}
			i += 4
			if value >= 0xdc00 && value <= 0xdfff {
				return false
			}
			if value >= 0xd800 && value <= 0xdbff {
				if i+6 >= len(raw) || raw[i+1:i+3] != `\u` {
					return false
				}
				low, err := strconv.ParseUint(raw[i+3:i+7], 16, 16)
				if err != nil || low < 0xdc00 || low > 0xdfff {
					return false
				}
				i += 6
			}
		}
	}
	return true
}
