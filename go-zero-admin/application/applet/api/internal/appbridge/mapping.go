package appbridge

import (
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"
)

func Copy[T any](source any) (*T, error) {
	var target T
	if err := copier.Copy(&target, source); err != nil {
		return nil, err
	}
	return &target, nil
}
func Page(no, size int64, keyword string) *pb.PageRequest {
	if no < 1 {
		no = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return &pb.PageRequest{PageNo: no, PageSize: size, Keyword: keyword}
}
