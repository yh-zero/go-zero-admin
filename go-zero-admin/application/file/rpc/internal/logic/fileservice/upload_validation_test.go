package fileservicelogic

import (
	"bytes"
	"encoding/base64"
	"io"
	"strings"
	"testing"
)

func TestValidateUploadContentAndMetadata(t *testing.T) {
	name, ext, scene, visibility, err := validateUploadMetadata(" report.txt ", "", "")
	if err != nil || name != "report.txt" || ext != ".txt" || scene != "common" || visibility != "private" {
		t.Fatalf("unexpected metadata: %s %s %s %s %v", name, ext, scene, visibility, err)
	}
	if _, _, _, _, err := validateUploadMetadata("bad.txt", "script", ""); err == nil {
		t.Fatal("invalid scene must fail")
	}
	mime, err := validateUploadContent(strings.NewReader("a,b\n1,2\n"), ".csv", "common")
	if err != nil || mime != "text/csv" {
		t.Fatalf("unexpected csv validation: %s %v", mime, err)
	}
	if _, err := validateUploadContent(strings.NewReader("<svg onload=alert(1)>"), ".txt", "common"); err == nil {
		t.Fatal("active text content must fail")
	}
}

func TestValidateUploadImageStream(t *testing.T) {
	const tinyPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
	data, err := base64.StdEncoding.DecodeString(tinyPNG)
	if err != nil {
		t.Fatal(err)
	}
	mime, err := validateUploadContent(bytes.NewReader(data), ".png", "avatar")
	if err != nil || mime != "image/png" {
		t.Fatalf("unexpected png validation: %s %v", mime, err)
	}
}

func TestValidUTF8StreamAcrossChunks(t *testing.T) {
	data := []byte("中文验收")
	for _, chunk := range []int{1, 2, 3, len(data)} {
		reader := io.LimitReader(bytes.NewReader(data), int64(chunk))
		if !validUTF8Stream(io.MultiReader(reader, bytes.NewReader(data[chunk:]))) {
			t.Fatalf("valid UTF-8 rejected at chunk %d", chunk)
		}
	}
	if validUTF8Stream(bytes.NewReader([]byte{0xff})) {
		t.Fatal("invalid UTF-8 accepted")
	}
}
