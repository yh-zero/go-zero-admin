package session

import (
	"context"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
)

func sessionRequest(ctx context.Context) *pb.SessionRequest {
	data := ctxJwt.GetJwtData(ctx)
	return &pb.SessionRequest{UserID: data.ID, AuthorityId: data.AuthorityId, SessionVersion: data.SessionVersion}
}
