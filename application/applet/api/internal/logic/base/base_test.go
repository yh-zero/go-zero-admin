package base

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-zero-admin/application/applet/api/internal/config"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

func uploadRequest(t *testing.T, field string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, "example.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestImageUploadValidation(t *testing.T) {
	var valid bytes.Buffer
	if err := png.Encode(&valid, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	data, mime, err := readImage(uploadRequest(t, "file_img", valid.Bytes()))
	if err != nil || mime != "image/png" || !bytes.Equal(data, valid.Bytes()) {
		t.Fatalf("valid PNG rejected: %v", err)
	}
	for _, tc := range []struct {
		name, field string
		data        []byte
	}{
		{"wrong field", "file", valid.Bytes()},
		{"empty", "file_img", nil},
		{"HTML disguised as PNG", "file_img", []byte("<html><script>alert(1)</script></html>")},
		{"too large", "file_img", bytes.Repeat([]byte("x"), maxFileSize+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := readImage(uploadRequest(t, tc.field, tc.data)); err == nil {
				t.Fatal("invalid upload accepted")
			}
		})
	}
	req := uploadRequest(t, "file_img", valid.Bytes())
	_, err = NewUploadFileImgLogic(req.Context(), &svc.ServiceContext{}).UploadFileImg(&types.UploadFileImgRequest{}, req)
	if err == nil || !strings.Contains(err.Error(), "未配置") {
		t.Fatalf("missing OSS not explained: %v", err)
	}
}

func TestImageURL(t *testing.T) {
	got := genFileURL("bucket", "https://oss.example.test/", "folder/image name.png")
	if got != "https://bucket.oss.example.test/folder/image%20name.png" {
		t.Fatal(got)
	}
}

func TestEmailValidation(t *testing.T) {
	for _, value := range []string{"", "invalid", "Display <test@example.test>", "test@example.test\r\nBcc: victim@example.test"} {
		if _, err := normalizeEmail(value); err == nil {
			t.Fatalf("accepted invalid recipient: %q", value)
		}
	}
	if got, err := normalizeEmail(" test@example.test "); err != nil || got != "test@example.test" {
		t.Fatalf("valid recipient: %q %v", got, err)
	}
	if err := validateMail(config.MailConfig{}); err == nil {
		t.Fatal("missing SMTP accepted")
	}
	for i := 0; i < 20; i++ {
		code, err := emailCode()
		if err != nil || len(code) != 6 || strings.Trim(code, "0123456789") != "" {
			t.Fatalf("invalid code: %v", err)
		}
	}
}
