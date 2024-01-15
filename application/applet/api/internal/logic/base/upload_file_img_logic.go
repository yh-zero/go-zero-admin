package base

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxFileSize = 10 << 20 // 10MB

type UploadFileImgLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadFileImgLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileImgLogic {
	return &UploadFileImgLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}
func (l *UploadFileImgLogic) UploadFileImg(req *types.UploadFileImgRequest, r *http.Request) (resp *types.UploadFileImgResponse, err error) {
	_ = r.ParseMultipartForm(maxFileSize)
	file, handler, err := r.FormFile("file_img")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bucket, err := l.svcCtx.OssClient.Bucket(l.svcCtx.Config.Oss.BucketName)
	if err != nil {
		logx.Errorf("get bucket failed, err: %v", err)
		return nil, xerr.NewErrCode(300001)
	}

	objectKey := genFilename(handler.Filename, "go-zero-admin") // 指定目录为 go-zero-admin
	err = bucket.PutObject(objectKey, file)
	if err != nil {
		logx.Errorf("put object failed, err: %v", err)
		return nil, xerr.NewErrCode(300002)
	}

	return &types.UploadFileImgResponse{
		FileImgUrl: genFileURL(l.svcCtx.Config.Oss.BucketName, l.svcCtx.Config.Oss.Endpoint, objectKey),
	}, nil
}

func genFilename(filename, directory string) string {
	// 将文件名拼接到指定目录中
	return fmt.Sprintf("%s/%d_%s", directory, time.Now().UnixMilli(), filename)
}

func genFileURL(bucketName, Endpoint, objectKey string) string {
	// 返回文件的 URL，注意 OSS 对象键中的目录名称
	return fmt.Sprintf("https://%s.%s/%s", bucketName, Endpoint, objectKey)
}
