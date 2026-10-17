package files

import (
	"context"

	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
)

func actorSession(ctx context.Context) *pb.SessionRequest {
	actor := ctxJwt.GetJwtData(ctx)
	return &pb.SessionRequest{UserID: actor.ID, AuthorityId: actor.AuthorityId, SessionVersion: actor.SessionVersion, SessionID: actor.SessionID}
}
func fileType(file *pb.FileResource) types.FileResource {
	return types.FileResource{Id: file.ID, Name: file.Name, Mime: file.Mime, Size: file.Size, OwnerId: file.OwnerID,
		DepartmentId: file.DepartmentId, Visibility: file.Visibility, Status: file.Status, References: file.References, CreatedAt: file.CreatedAt}
}
// publicFileURL moved to the RPC tier (fileresourceservice/oss_helpers.go):
// URL construction requires the OSS endpoint/bucket, which the API no longer
// holds.
