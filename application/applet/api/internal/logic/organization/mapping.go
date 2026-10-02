package organization

import (
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
)

func departmentInput(d *types.Department) *pb.Department {
	if d == nil {
		return nil
	}
	return &pb.Department{ID: d.Id, ParentId: d.ParentId, Name: d.Name, Code: d.Code, Sort: d.Sort, Status: d.Status, Leader: d.Leader}
}
func departmentOutput(d *pb.Department) *types.Department {
	if d == nil {
		return nil
	}
	out := &types.Department{Id: d.ID, ParentId: d.ParentId, Name: d.Name, Code: d.Code, Sort: d.Sort, Status: d.Status, Leader: d.Leader, Children: []types.Department{}}
	for _, child := range d.Children {
		if value := departmentOutput(child); value != nil {
			out.Children = append(out.Children, *value)
		}
	}
	return out
}
func positionInput(p *types.Position) *pb.Position {
	if p == nil {
		return nil
	}
	return &pb.Position{ID: p.Id, Name: p.Name, Code: p.Code, Sort: p.Sort, Status: p.Status}
}
func positionOutput(p *pb.Position) *types.Position {
	if p == nil {
		return nil
	}
	return &types.Position{Id: p.ID, Name: p.Name, Code: p.Code, Sort: p.Sort, Status: p.Status}
}
