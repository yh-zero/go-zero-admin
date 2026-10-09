package user

import (
	"context"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
)

func resourceActor(ctx context.Context) *pb.SessionRequest {
	identity := ctxJwt.GetJwtData(ctx)
	return &pb.SessionRequest{UserID: identity.ID, AuthorityId: identity.AuthorityId, SessionVersion: identity.SessionVersion, SessionID: identity.SessionID}
}
