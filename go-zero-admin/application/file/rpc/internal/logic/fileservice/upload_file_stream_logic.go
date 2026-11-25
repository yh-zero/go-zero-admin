package fileservicelogic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

const uploadStreamMaxBytes = 10 << 20

type UploadFileStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadFileStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileStreamLogic {
	return &UploadFileStreamLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UploadFileStreamLogic) UploadFileStream(stream filepb.FileService_UploadFileStreamServer) error {
	first, err := stream.Recv()
	if err != nil {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "缺少上传元数据")
	}
	if first.GetActor().GetUserID() <= 0 {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if len(first.GetContent()) > 0 {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "首帧只能包含上传元数据")
	}
	name, ext, scene, visibility, err := validateUploadMetadata(first.OriginalName, first.Scene, first.Visibility)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp("", "gozero-upload-*")
	if err != nil {
		return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	path := file.Name()
	defer func() {
		_ = file.Close()
		_ = os.Remove(path)
	}()
	hash := sha256.New()
	var total int64
	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return recvErr
		}
		if chunk.GetActor() != nil || chunk.GetOriginalName() != "" || chunk.GetScene() != "" || chunk.GetVisibility() != "" {
			return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "上传过程中不能修改元数据")
		}
		content := chunk.GetContent()
		if len(content) == 0 {
			continue
		}
		total += int64(len(content))
		if total > uploadStreamMaxBytes {
			return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文件大小须为 1 字节至 10 MiB")
		}
		if _, err = file.Write(content); err != nil {
			return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		_, _ = hash.Write(content)
	}
	if total == 0 {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文件大小须为 1 字节至 10 MiB")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	mime, err := validateUploadContent(file, ext, scene)
	if err != nil {
		return err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	response, err := persistUpload(stream.Context(), l.svcCtx, first.Actor, persistUploadInput{
		Name: name, Ext: ext, Mime: mime, Scene: scene, Visibility: visibility,
		Body: file, Size: total, SHA256: hex.EncodeToString(hash.Sum(nil)),
	})
	if err != nil {
		return err
	}
	return stream.SendAndClose(response)
}
