// Package filerpc defines transport limits for the file resource RPC boundary.
package filerpc

import (
	"time"

	"google.golang.org/grpc"
)

// OperationTimeout matches the file HTTP routes and per-method RPC server
// deadlines. Other business calls retain their configured RPC timeout.
const OperationTimeout = 30 * time.Second

// ServerOptions leaves room for protobuf metadata around the 10MiB image limit.
// The business upload validator continues to enforce the 10MiB file limit.
func ServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{grpc.MaxRecvMsgSize(11 << 20)}
}
