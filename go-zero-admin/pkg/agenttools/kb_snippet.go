package agenttools

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RelevantKbSnippet 在预算不足时保留问题词附近的连续原文；无词法匹配时沿用原前缀。
// 选择只依赖原文和本次问题，不补写省略号、事实或来源标识。
func RelevantKbSnippet(content, query string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(content) <= maxBytes {
		return content
	}
	return newKbSnippet(content, query).text(maxBytes)
}

type kbSnippet struct {
	content string
	center  int // 固定锚点让预算二分产生嵌套窗口，JSON转义大小也随之单调。
}

type kbSnippetMatch struct {
	start, end, term int
}

func newKbSnippet(content, query string) kbSnippet {
	plan := kbSnippet{content: content, center: -1}
	folded := foldSnippetASCII(content)
	terms := kbSnippetTerms(query)
	var matches []kbSnippetMatch
	for termIndex, term := range terms {
		for offset := 0; offset < len(folded); {
			found := strings.Index(folded[offset:], term)
			if found < 0 {
				break
			}
			start := offset + found
			end := start + len(term)
			// 英文按完整词或标识符匹配，避免把 limit 匹配到 unlimited。
			if !isSnippetWord(term[0]) || (start == 0 || !isSnippetWord(folded[start-1])) && (end == len(folded) || !isSnippetWord(folded[end])) {
				matches = append(matches, kbSnippetMatch{start: start, end: end, term: termIndex})
			}
			_, size := utf8.DecodeRuneInString(folded[start:])
			offset = start + size
		}
	}
	if len(matches) == 0 {
		return plan
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].start != matches[j].start {
			return matches[i].start < matches[j].start
		}
		return matches[i].end < matches[j].end
	})
	// 用局部不同词的覆盖选位置，重复出现的前言词不会累加压过局部事实。
	const localityBytes = 256
	counts := make([]int, len(terms))
	left, score, bestScore, bestStart, bestEnd := 0, 0, -1, 0, 0
	weight := func(term int) int { return min(8, utf8.RuneCountInString(terms[term])) }
	for right, match := range matches {
		if counts[match.term] == 0 {
			score += weight(match.term)
		}
		counts[match.term]++
		for left < right && match.end-matches[left].start > localityBytes {
			prior := matches[left]
			counts[prior.term]--
			if counts[prior.term] == 0 {
				score -= weight(prior.term)
			}
			left++
		}
		start, end := matches[left].start, match.end
		if score > bestScore || score == bestScore && end-start < bestEnd-bestStart {
			bestScore, bestStart, bestEnd = score, start, end
		}
	}
	// 锚点固定在实际命中词内；整句中点或词间空白会让小窗口丢掉全部相关证据。
	clusterCenter := bestStart + (bestEnd-bestStart)/2
	closest := len(content)
	for _, match := range matches {
		if match.start < bestStart || match.end > bestEnd {
			continue
		}
		center := match.start + (match.end-match.start)/2
		distance := max(clusterCenter-center, center-clusterCenter)
		if distance < closest {
			plan.center, closest = center, distance
		}
	}
	return plan
}

func (s kbSnippet) text(maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s.content) <= maxBytes {
		return s.content
	}
	if s.center < 0 {
		return truncateUTF8(s.content, maxBytes)
	}
	start := max(0, min(s.center-maxBytes/2, len(s.content)-maxBytes))
	end := start + maxBytes
	for start < end && !utf8.RuneStart(s.content[start]) {
		start++
	}
	for end > start && end < len(s.content) && !utf8.RuneStart(s.content[end]) {
		end--
	}
	return s.content[start:end]
}

func foldSnippetASCII(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, value)
}

func isSnippetWord(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '_'
}

func kbSnippetTerms(query string) []string {
	seen := make(map[string]bool)
	var terms []string
	add := func(value string) {
		if value != "" && !seen[value] {
			seen[value] = true
			terms = append(terms, value)
		}
	}
	runes := []rune(foldSnippetASCII(query))
	for i := 0; i < len(runes); {
		if unicode.Is(unicode.Han, runes[i]) {
			end := i + 1
			for end < len(runes) && unicode.Is(unicode.Han, runes[end]) {
				end++
			}
			if end-i == 1 {
				add(string(runes[i]))
			} else {
				for j := i; j+1 < end; j++ {
					add(string(runes[j : j+2]))
				}
			}
			i = end
			continue
		}
		if runes[i] < utf8.RuneSelf && isSnippetWord(byte(runes[i])) {
			end := i + 1
			for end < len(runes) && runes[end] < utf8.RuneSelf && (isSnippetWord(byte(runes[end])) || strings.ContainsRune("./:-", runes[end])) {
				end++
			}
			token := strings.Trim(string(runes[i:end]), "./:-")
			if len(token) >= 2 {
				add(token)
			}
			for _, word := range strings.FieldsFunc(token, func(r rune) bool { return strings.ContainsRune("_./:-", r) }) {
				if len(word) >= 2 {
					add(word)
				}
			}
			i = end
			continue
		}
		i++
	}
	return terms
}
