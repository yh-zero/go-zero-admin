package aiservicelogic

import (
	"strings"
	"testing"
)

func TestChunkTextBasic(t *testing.T) {
	content := strings.Repeat("第一段内容。", 30) + "\n\n" + strings.Repeat("第二段内容。", 30)
	chunks := chunkText(content)
	if len(chunks) == 0 {
		t.Fatal("应产生分块")
	}
	for i, c := range chunks {
		if c.Seq != int64(i) {
			t.Fatalf("seq 不连续: %d != %d", c.Seq, i)
		}
		if c.TokenCount <= 0 || c.TokenCount > kbChunkTargetTokens+kbChunkOverlapTokens+16 {
			t.Fatalf("chunk %d token 数越界: %d", i, c.TokenCount)
		}
		if strings.TrimSpace(c.Content) != c.Content {
			t.Fatalf("chunk %d 未去首尾空白", i)
		}
	}
}

func TestChunkTextEmpty(t *testing.T) {
	if got := chunkText("  \n\n  "); len(got) != 0 {
		t.Fatalf("全空白不应产生分块: %d", len(got))
	}
}

func TestChunkTextOverlap(t *testing.T) {
	// 超长单段触发硬切与重叠
	content := strings.Repeat("长", kbChunkTargetTokens*3)
	chunks := chunkText(content)
	if len(chunks) < 3 {
		t.Fatalf("超长文本应切多块: %d", len(chunks))
	}
	for i := 1; i < len(chunks); i++ {
		prev := []rune(chunks[i-1].Content)
		curr := []rune(chunks[i].Content)
		if len(curr) < kbChunkOverlapTokens {
			t.Fatalf("chunk %d 缺少重叠前缀", i)
		}
		overlap := prev[len(prev)-kbChunkOverlapTokens:]
		if string(curr[:kbChunkOverlapTokens]) != string(overlap) {
			t.Fatalf("chunk %d 重叠前缀与上一块尾部不一致", i)
		}
	}
}

func TestChunkTextMultibyte(t *testing.T) {
	// 多字节字符不得被按字节切坏
	content := strings.Repeat("知识库分块测试，", 100)
	for _, c := range chunkText(content) {
		if !strings.Contains("，库识知试测块分", string([]rune(c.Content)[0])) {
			t.Fatalf("首字符被切断: %q", string([]rune(c.Content)[0]))
		}
	}
}
