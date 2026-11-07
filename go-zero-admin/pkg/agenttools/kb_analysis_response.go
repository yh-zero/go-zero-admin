package agenttools

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const kbAnalysisTruncatedText = "分析达到输出上限，结构未完整返回，无法展示分析正文。请缩小分析范围后重试。"

var analysisCitationMarker = regexp.MustCompile(`[\[【［]\s*[0-9０-９]`)

type kbModelAnswer struct {
	Target    string `json:"target"`
	State     string `json:"state"`
	Text      string `json:"text"`
	Citations []int  `json:"citations"`
}

// analysisResponseObject 只用于内部两层对象，逐键读取以拒绝重复键、大小写替代和未知字段。
func analysisResponseObject(raw json.RawMessage, names ...string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(strings.NewReader(string(raw)))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return nil, errors.New("内部回复必须为对象")
	}
	fields := make(map[string]json.RawMessage, len(names))
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil, err
		}
		name, ok := key.(string)
		allowed := false
		for _, candidate := range names {
			allowed = allowed || name == candidate
		}
		if !ok || !allowed || fields[name] != nil {
			return nil, errors.New("内部回复含未知或重复字段")
		}
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return nil, err
		}
		fields[name] = value
	}
	if _, err := d.Token(); err != nil || len(fields) != len(names) {
		return nil, errors.New("内部回复缺少字段")
	}
	return fields, nil
}

func renderKbAnalysisResponse(raw, query string, sources []KbAnalysisSource) (string, error) {
	if !utf8.ValidString(raw) || !json.Valid([]byte(raw)) {
		return "", errors.New("内部回复不是有效UTF-8 JSON")
	}
	root, err := analysisResponseObject(json.RawMessage(raw), "answers")
	if err != nil {
		return "", err
	}
	var items []json.RawMessage
	if json.Unmarshal(root["answers"], &items) != nil || len(items) < 1 || len(items) > 16 {
		return "", errors.New("内部回复问项数量无效")
	}
	answers := make([]kbModelAnswer, 0, len(items))
	supported := 0
	for _, item := range items {
		fields, err := analysisResponseObject(item, "target", "state", "text", "citations")
		if err != nil {
			return "", err
		}
		for _, value := range fields {
			if strings.TrimSpace(string(value)) == "null" {
				return "", errors.New("内部回复字段不得为null")
			}
		}
		var answer kbModelAnswer
		if json.Unmarshal(item, &answer) != nil || !utf8.ValidString(answer.Target) || answer.Target == "" || answer.Target != strings.TrimSpace(answer.Target) || len(answer.Target) > len(query) || !strings.Contains(query, answer.Target) {
			return "", errors.New("内部回复目标不是原问题中的有效文字")
		}
		for _, previous := range answers {
			if strings.Contains(previous.Target, answer.Target) || strings.Contains(answer.Target, previous.Target) {
				return "", errors.New("内部回复目标重复或互相包含")
			}
		}
		switch answer.State {
		case "supported":
			if !utf8.ValidString(answer.Text) || strings.TrimSpace(answer.Text) == "" || len(answer.Text) > 2000 || len(answer.Citations) == 0 || analysisCitationMarker.MatchString(answer.Text) {
				return "", errors.New("有据问项的正文或引用格式无效")
			}
			seen := make(map[int]bool, len(answer.Citations))
			for _, citation := range answer.Citations {
				if citation < 1 || citation > len(sources) || seen[citation] {
					return "", errors.New("内部回复引用编号无效或重复")
				}
				seen[citation] = true
			}
			if analysisTextHasInsufficiency(answer.Text) {
				quoted := false
				for _, citation := range answer.Citations {
					quoted = quoted || strings.Contains(sources[citation-1].Snippet, answer.Text)
				}
				if !quoted {
					return "", errors.New("有据问项不得附带自行生成的不足判断")
				}
			}
			supported++
		case "insufficient":
			if answer.Text != "" || len(answer.Citations) != 0 {
				return "", errors.New("不足问项不得附带正文或引用")
			}
		default:
			return "", errors.New("内部回复问项状态无效")
		}
		answers = append(answers, answer)
	}
	if supported == 0 {
		return "本次片段资料不足，无法回答该分析目标。", nil
	}
	parts := make([]string, 0, len(answers))
	for _, answer := range answers {
		if answer.State == "insufficient" {
			parts = append(parts, "本次片段未能确定“"+escapeAnalysisTarget(answer.Target)+"”。")
			continue
		}
		text := escapeAnalysisTarget(answer.Target) + "：" + strings.TrimSpace(answer.Text) + " "
		for _, citation := range answer.Citations {
			text += fmt.Sprintf("[%d]", citation)
		}
		parts = append(parts, text)
	}
	text := strings.Join(parts, "\n\n")
	encoded, _ := json.Marshal(text)
	if len(text) > kbAnalysisTextBytes || len(encoded) > 13000 {
		return "", errors.New("渲染后的内部回复超过大小限制")
	}
	return text, nil
}

// 这是内部state格式约束：明示不足须为被引用原文的完整摘录，不删词，也不判断任意语义是否正确。
func analysisTextHasInsufficiency(text string) bool {
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, text)
	for _, phrase := range []string{"未提供", "未涉及", "未说明", "未记载", "未提及", "未包含", "未给出", "无法确定", "无法确认", "无法判断", "不能确定", "不能确认", "资料不足", "信息不足", "证据不足", "没有提供", "没有说明"} {
		if strings.Contains(compact, phrase) {
			return true
		}
	}
	return false
}

func escapeAnalysisTarget(target string) string {
	var out strings.Builder
	for _, r := range target {
		if r == '\n' || r == '\r' || r == '\t' {
			out.WriteByte(' ')
			continue
		}
		if strings.ContainsRune("\\`*_{}[]<>()#+-.!|~&", r) {
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}

func incompleteKbAnalysisJSON(raw string) bool {
	var value json.RawMessage
	err := json.NewDecoder(strings.NewReader(raw)).Decode(&value)
	return errors.Is(err, io.ErrUnexpectedEOF)
}
