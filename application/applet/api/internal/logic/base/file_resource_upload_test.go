package base

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

func resourceUploadRequest(t *testing.T, name, visibility string) *http.Request {
	t.Helper()
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file_img", name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(content.Bytes())
	if visibility != "" {
		_ = writer.WriteField("visibility", visibility)
	}
	_ = writer.Close()
	request := httptest.NewRequest(http.MethodPost, "/v1/sys/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request.WithContext(ctxWithJwt(request.Context()))
}

// Invalid visibility must be rejected before any RPC call, because the API
// tier is the policy boundary for user-supplied form values.
func TestInvalidVisibilityRejectedBeforeRPC(t *testing.T) {
	rpc := &capturingFileClient{}
	s := &svc.ServiceContext{AppletFileRPC: rpc}
	r := resourceUploadRequest(t, "example.png", "friends-only")
	if _, err := NewUploadFileImgLogic(r.Context(), s).UploadFileImg(&types.UploadFileImgRequest{}, r); err == nil {
		t.Fatal("invalid visibility accepted")
	}
	if rpc.request != nil {
		t.Fatal("invalid visibility still reached RPC")
	}
}

// The original filename must reach the RPC verbatim; path components and
// extension sanity are enforced server-side, but the API must not mangle it.
func TestUploadFileImgPreservesFilename(t *testing.T) {
	rpc := &capturingFileClient{}
	s := &svc.ServiceContext{AppletFileRPC: rpc}
	r := resourceUploadRequest(t, "a b.png", "private")
	if _, err := NewUploadFileImgLogic(r.Context(), s).UploadFileImg(&types.UploadFileImgRequest{}, r); err != nil {
		t.Fatal(err)
	}
	if rpc.request == nil || rpc.request.Filename != "a b.png" || rpc.request.Visibility != "private" {
		t.Fatalf("filename/visibility not forwarded: %+v", rpc.request)
	}
}
