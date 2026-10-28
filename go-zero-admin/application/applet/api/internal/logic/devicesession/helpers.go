package devicesession

import (
	"context"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
)

func actor(ctx context.Context) *pb.SessionRequest {
	data := ctxJwt.GetJwtData(ctx)
	return &pb.SessionRequest{UserID: data.ID, AuthorityId: data.AuthorityId, SessionVersion: data.SessionVersion, SessionID: data.SessionID}
}
func deviceResponse(response *pb.DeviceSessionListResponse, page, size int64) *types.DeviceSessionListResponse {
	result := &types.DeviceSessionListResponse{List: make([]types.DeviceSession, 0, len(response.List)), PageResponse: types.PageResponse{Total: response.Total, PageNo: page, PageSize: size}}
	for _, item := range response.List {
		result.List = append(result.List, types.DeviceSession{Id: item.ID, UserId: item.UserID, Username: item.Username, Ip: item.IP, UserAgent: item.UserAgent, CreatedAt: item.CreatedAt, ExpiresAt: item.ExpiresAt, Current: item.Current})
	}
	return result
}
