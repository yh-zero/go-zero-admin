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

type analysisResponseError uint8

const (
	analysisResponseInvalidJSON analysisResponseError = iota
	analysisResponseInvalidObject
	analysisResponseUnknownFields
	analysisResponseMissingFields
	analysisResponseInvalidCount
	analysisResponseNullField
	analysisResponseInvalidTarget
	analysisResponseRepeatedTarget
	analysisResponseInvalidSupported
	analysisResponseInvalidCitation
	analysisResponseUncitedInsufficiency
	analysisResponseInvalidInsufficient
	analysisResponseBackgroundTarget
	analysisResponseInvalidState
	analysisResponseOversize
	analysisResponseTooManySources
)

// 枚举只映射到静态原因，解码错误中的模型内容和字段不得进入日志。
func (reason analysisResponseError) Error() string {
	switch reason {
	case analysisResponseInvalidObject:
		return "内部回复必须为对象"
	case analysisResponseUnknownFields:
		return "内部回复含未知或重复字段"
	case analysisResponseMissingFields:
		return "内部回复缺少字段"
	case analysisResponseInvalidCount:
		return "内部回复问项数量无效"
	case analysisResponseNullField:
		return "内部回复字段不得为null"
	case analysisResponseInvalidTarget:
		return "内部回复目标不是原问题中的有效文字"
	case analysisResponseRepeatedTarget:
		return "内部回复目标重复或互相包含"
	case analysisResponseInvalidSupported:
		return "有据问项的正文或引用格式无效"
	case analysisResponseInvalidCitation:
		return "内部回复引用编号无效或重复"
	case analysisResponseTooManySources:
		return "内部回复引用来源超过8条"
	case analysisResponseUncitedInsufficiency:
		return "有据问项不得附带自行生成的不足判断"
	case analysisResponseInvalidInsufficient:
		return "不足问项不得附带正文或引用"
	case analysisResponseBackgroundTarget:
		return "已结束的背景引文不能独立成为问项"
	case analysisResponseInvalidState:
		return "内部回复问项状态无效"
	case analysisResponseOversize:
		return "渲染后的内部回复超过大小限制"
	default:
		return "JSON格式无效"
	}
}

func analysisResponseFailureReason(err error) string {
	var reason analysisResponseError
	if errors.As(err, &reason) {
		return reason.Error()
	}
	return analysisResponseInvalidJSON.Error()
}

var analysisCitationMarker = regexp.MustCompile(`[\[【［]\s*[0-9０-９]`)

// 只识别高置信的转述背景，不把这组通用语言标记当作完整意图解析。
var analysisQuoteReporting = regexp.MustCompile(`(?i)建议|声称|写道|写着|示例|例如|说|\b(?:said|says?|suggest(?:ed|s)?|claim(?:ed|s)?|example)\b`)
var analysisQuoteRequest = regexp.MustCompile(`(?i)请|回答|告诉|说明|解释|核实|核对|确认|判断|询问|想知道|想问|\b(?:please|tell|answer|explain|describe|verify|confirm|show)\b`)
var analysisQuoteQuestion = regexp.MustCompile(`(?i)[?？]|什么|是否|多少|如何|怎么|能否|为何|哪里|哪个|哪些|何时|\b(?:what|when|where|why|how|whether|who|which)\b`)
var analysisQuoteNextRequest = regexp.MustCompile(`(?i)^(?:请|说明|解释|回答|核实|核对|确认|判断|告诉|please\b|tell\b|answer\b|explain\b|describe\b|verify\b|confirm\b|show\b|what\b|when\b|where\b|why\b|how\b)`)
var analysisQuoteReference = regexp.MustCompile(`(?i)引文|引号|引用|上述|上文|这|该|其|它|\b(?:quoted|quotation|above|this|that|it)\b`)

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
		return nil, analysisResponseInvalidObject
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
			return nil, analysisResponseUnknownFields
		}
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return nil, err
		}
		fields[name] = value
	}
	if _, err := d.Token(); err != nil || len(fields) != len(names) {
		return nil, analysisResponseMissingFields
	}
	return fields, nil
}

func parseKbAnalysisResponse(raw, query string, sources []KbAnalysisSource) ([]kbModelAnswer, error) {
	if !utf8.ValidString(raw) || !json.Valid([]byte(raw)) {
		return nil, analysisResponseInvalidJSON
	}
	root, err := analysisResponseObject(json.RawMessage(raw), "answers")
	if err != nil {
		return nil, err
	}
	var items []json.RawMessage
	if json.Unmarshal(root["answers"], &items) != nil || len(items) < 1 || len(items) > 16 {
		return nil, analysisResponseInvalidCount
	}
	answers := make([]kbModelAnswer, 0, len(items))
	queryQuotes := inspectAnalysisQueryQuotes(query)
	cited := make(map[int]bool)
	for _, item := range items {
		fields, err := analysisResponseObject(item, "target", "state", "text", "citations")
		if err != nil {
			return nil, err
		}
		for _, value := range fields {
			if strings.TrimSpace(string(value)) == "null" {
				return nil, analysisResponseNullField
			}
		}
		var answer kbModelAnswer
		if json.Unmarshal(item, &answer) != nil || !utf8.ValidString(answer.Target) || answer.Target == "" || answer.Target != strings.TrimSpace(answer.Target) || len(answer.Target) > len(query) || !strings.Contains(query, answer.Target) {
			return nil, analysisResponseInvalidTarget
		}
		for _, previous := range answers {
			if strings.Contains(previous.Target, answer.Target) || strings.Contains(answer.Target, previous.Target) {
				return nil, analysisResponseRepeatedTarget
			}
		}
		if queryQuotes.targetOnlyBackground(query, answer.Target) {
			return nil, analysisResponseBackgroundTarget
		}
		switch answer.State {
		case "supported":
			if !utf8.ValidString(answer.Text) || strings.TrimSpace(answer.Text) == "" || len(answer.Text) > 2000 || len(answer.Citations) == 0 || analysisCitationMarker.MatchString(answer.Text) {
				return nil, analysisResponseInvalidSupported
			}
			seen := make(map[int]bool, len(answer.Citations))
			for _, citation := range answer.Citations {
				if citation < 1 || citation > len(sources) || seen[citation] {
					return nil, analysisResponseInvalidCitation
				}
				seen[citation] = true
				cited[citation] = true
			}
			if analysisTextHasInsufficiency(answer.Text) {
				quoted := false
				for _, citation := range answer.Citations {
					quoted = quoted || strings.Contains(sources[citation-1].Snippet, answer.Text)
				}
				if !quoted {
					return nil, analysisResponseUncitedInsufficiency
				}
			}
		case "insufficient":
			if answer.Text != "" || len(answer.Citations) != 0 {
				return nil, analysisResponseInvalidInsufficient
			}
		default:
			return nil, analysisResponseInvalidState
		}
		answers = append(answers, answer)
	}
	if len(cited) > kbAnalysisMaxSources {
		return nil, analysisResponseTooManySources
	}
	return answers, nil
}

func renderKbAnalysisResponse(raw, query string, sources []KbAnalysisSource) (string, error) {
	answers, err := parseKbAnalysisResponse(raw, query, sources)
	if err != nil {
		return "", err
	}
	return renderKbAnalysisAnswers(answers)
}

// 全量校验后才裁剪引用来源；只映射结构化整数，不替换正文或用户目标中的编号。
func renderKbAnalysisWithSources(raw, query string, sources []KbAnalysisSource) (string, []KbAnalysisSource, error) {
	answers, err := parseKbAnalysisResponse(raw, query, sources)
	if err != nil {
		return "", nil, err
	}
	cited := make(map[int]bool)
	for _, answer := range answers {
		for _, citation := range answer.Citations {
			cited[citation] = true
		}
	}
	selected := make([]KbAnalysisSource, 0, min(len(sources), kbAnalysisMaxSources))
	indices := make(map[int]int, len(cited))
	for _, source := range sources {
		if !cited[source.Index] && (len(cited) > 0 || len(selected) >= kbAnalysisMaxSources) {
			continue
		}
		oldIndex := source.Index
		source.Index = len(selected) + 1
		selected = append(selected, source)
		indices[oldIndex] = source.Index
	}
	for i := range answers {
		for j, citation := range answers[i].Citations {
			answers[i].Citations[j] = indices[citation]
		}
	}
	text, err := renderKbAnalysisAnswers(answers)
	return text, selected, err
}

func renderKbAnalysisAnswers(answers []kbModelAnswer) (string, error) {
	supported := 0
	for _, answer := range answers {
		if answer.State == "supported" {
			supported++
		}
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
		return "", analysisResponseOversize
	}
	return text, nil
}

type analysisQueryQuotes struct {
	covered       []bool
	hasBackground bool
	background    []string
	request       string
}

// 仅识别确定的平衡引文；无法确定边界时沿用既有目标校验，不猜测缺失引号。
func inspectAnalysisQueryQuotes(query string) analysisQueryQuotes {
	if !utf8.ValidString(query) {
		return analysisQueryQuotes{}
	}
	opening := map[rune]rune{'“': '”', '‘': '’', '「': '」', '『': '』', '"': '"', '\'': '\''}
	closing := map[rune]bool{'”': true, '’': true, '」': true, '』': true}
	type quote struct {
		start int
		end   rune
	}
	type span struct{ start, end int }
	var stack []quote
	var spans []span
	runes, runeIndex := []rune(query), 0
	escaped := false
	for offset, r := range query {
		index := runeIndex
		runeIndex++
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		// 英文词中的撇号不是引文边界；中文紧邻的ASCII单引号仍可成对引用。
		if (r == '\'' || r == '’') && index > 0 && index+1 < len(runes) && analysisApostropheWord(runes[index-1]) && analysisApostropheWord(runes[index+1]) {
			continue
		}
		if len(stack) > 0 && r == stack[len(stack)-1].end {
			if len(stack) == 1 {
				spans = append(spans, span{stack[0].start, offset + utf8.RuneLen(r)})
			}
			stack = stack[:len(stack)-1]
			continue
		}
		if end, ok := opening[r]; ok {
			stack = append(stack, quote{offset, end})
		} else if closing[r] {
			return analysisQueryQuotes{}
		}
	}
	if len(stack) != 0 || len(spans) == 0 {
		return analysisQueryQuotes{}
	}
	result := analysisQueryQuotes{covered: make([]bool, len(query))}
	for _, span := range spans {
		suffix := analysisQuoteBackgroundSuffix(query, span.start, span.end)
		if suffix == "" {
			continue
		}
		result.hasBackground = true
		result.background = append(result.background, query[span.start:span.end])
		prefix := strings.TrimSpace(query[:span.start])
		switch {
		case strings.HasSuffix(prefix, ":"):
			prefix = strings.TrimSuffix(prefix, ":")
		case strings.HasSuffix(prefix, "："):
			prefix = strings.TrimSuffix(prefix, "：")
		}
		// 只移除未分句的转述引导；不能用有限请求词表排除前文的实际任务。
		if result.request == "" && strings.IndexFunc(prefix, unicode.IsPunct) < 0 && !strings.ContainsAny(prefix, "\r\n") && !analysisQuoteRequest.MatchString(prefix) && !analysisQuoteQuestion.MatchString(prefix) && analysisQuoteNextRequest.MatchString(suffix) {
			result.request = suffix
		}
		for i := span.start; i < span.end; i++ {
			result.covered[i] = true
		}
	}
	return result
}

func analysisQuoteBackgroundSuffix(query string, start, end int) string {
	_, openingBytes := utf8.DecodeRuneInString(query[start:])
	_, closingBytes := utf8.DecodeLastRuneInString(query[:end])
	content := strings.TrimSpace(query[start+openingBytes : end-closingBytes])
	if content == "" || analysisQuoteQuestion.MatchString(content) {
		return ""
	}
	prefix := query[:start]
	if boundary := strings.LastIndexAny(prefix, "。！？.!?\n"); boundary >= 0 {
		_, size := utf8.DecodeRuneInString(prefix[boundary:])
		prefix = prefix[boundary+size:]
	}
	if !analysisQuoteReporting.MatchString(prefix) || analysisQuoteRequest.MatchString(prefix) || analysisQuoteQuestion.MatchString(prefix) {
		return ""
	}
	suffix := strings.TrimSpace(query[end:])
	if suffix == "" {
		return ""
	}
	first, _ := utf8.DecodeRuneInString(suffix)
	last, _ := utf8.DecodeLastRuneInString(content)
	if !strings.ContainsRune("。.!！？?", first) && !strings.ContainsRune("。.!！？?", last) {
		return ""
	}
	suffix = strings.TrimLeftFunc(suffix, func(r rune) bool { return unicode.IsSpace(r) || strings.ContainsRune("。.!！？?", r) })
	if (analysisQuoteNextRequest.MatchString(suffix) || analysisQuoteQuestion.MatchString(suffix)) && !analysisQuoteReference.MatchString(suffix) {
		return suffix
	}
	return ""
}

func analysisApostropheWord(r rune) bool {
	return unicode.In(r, unicode.Latin) || unicode.IsDigit(r) || r == '_'
}

func (quotes analysisQueryQuotes) targetOnlyBackground(query, target string) bool {
	if !quotes.hasBackground || target == "" {
		return false
	}
	found := false
	for start := 0; start+len(target) <= len(query); {
		offset := strings.Index(query[start:], target)
		if offset < 0 {
			break
		}
		offset += start
		found = true
		for i := offset; i < offset+len(target); i++ {
			if !quotes.covered[i] {
				return false
			}
		}
		start = offset + 1
	}
	return found
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
