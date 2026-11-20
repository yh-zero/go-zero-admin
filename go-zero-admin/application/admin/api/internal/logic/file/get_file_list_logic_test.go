package file

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/file/rpc/client/fileservice"
	"go-zero-admin/application/file/rpc/filepb"
)

type fileListCapture struct {
	fileservice.FileService
	request *filepb.GetFileListRequest
}

func (c *fileListCapture) GetFileList(_ context.Context, in *filepb.GetFileListRequest, _ ...grpc.CallOption) (*filepb.GetFileListResponse, error) {
	c.request = in
	return &filepb.GetFileListResponse{
		List: []*filepb.FileInfo{{Id: 4, CleanupPending: in.IncludeCleanupPending}}, Page: &filepb.PageResponse{Total: 1},
	}, nil
}

func TestFileCleanupListUsesAuthenticatedViewer(t *testing.T) {
	for _, query := range []string{"", "&include_cleanup_pending=true"} {
		req := httptest.NewRequest("GET", "/v1/sys/file/list?uploader_id=93"+query, nil)
		var input types.GetFileListReq
		if err := httpx.Parse(req, &input); err != nil {
			t.Fatal(err)
		}
		capture := &fileListCapture{}
		service := &svc.ServiceContext{FileRpc: capture, Validator: svc.NewValidator()}
		ctx := context.WithValue(req.Context(), middleware.CtxKeyUserID, int64(16))
		result, err := NewGetFileListLogic(ctx, service).GetFileList(&input)
		if err != nil {
			t.Fatal(err)
		}
		wantPending := query != ""
		if capture.request.GetActor().GetUserID() != 16 || capture.request.UploaderID != 93 ||
			capture.request.IncludeCleanupPending != wantPending || len(result.List) != 1 || result.List[0].CleanupPending != wantPending {
			t.Fatalf("清理查询的操作者、筛选条件或回执标记不符: %s", query)
		}
	}
}
