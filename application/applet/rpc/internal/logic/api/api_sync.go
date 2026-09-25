package apilogic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type swaggerOperation struct {
	Summary        string                `json:"summary"`
	Description    string                `json:"description"`
	Tags           []string              `json:"tags"`
	Security       []map[string][]string `json:"security"`
	CasbinResource *bool                 `json:"x-casbin-resource"`
}

type syncResource struct {
	Key         string
	ID          int64
	Path        string
	Method      string
	ApiGroup    string
	Description string
}

// readSwaggerResources reads only authenticated operations. Public login and
// CAPTCHA endpoints never need Casbin grants and are not permission resources.
func readSwaggerResources(data []byte) ([]syncResource, error) {
	var doc struct {
		Swagger  string                                `json:"swagger"`
		BasePath string                                `json:"basePath"`
		Paths    map[string]map[string]json.RawMessage `json:"paths"`
		Security []map[string][]string                 `json:"security"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("读取内置Swagger失败: %w", err)
	}
	if doc.Swagger != "2.0" || len(doc.Paths) == 0 {
		return nil, fmt.Errorf("内置Swagger格式无效，请重新生成文档后构建")
	}
	resources := []syncResource{}
	seen := map[string]bool{}
	for route, operations := range doc.Paths {
		for method, raw := range operations {
			switch method {
			case "get", "post", "put", "patch", "delete", "head", "options":
			default:
				continue // Path-level parameters and extensions are not operations.
			}
			var op swaggerOperation
			if err := json.Unmarshal(raw, &op); err != nil {
				return nil, err
			}
			security := op.Security
			if security == nil {
				security = doc.Security
			}
			authenticated := len(security) > 0
			for _, requirement := range security {
				if len(requirement) == 0 {
					authenticated = false
				}
			}
			if !authenticated {
				continue
			}
			// Generated from the .api middleware declaration, not merely JWT:
			// /me and other personal endpoints are authenticated but never granted.
			if op.CasbinResource != nil && !*op.CasbinResource {
				continue
			}
			path, method, err := accessutil.API(strings.TrimRight(doc.BasePath, "/")+route, method)
			if err != nil {
				return nil, err
			}
			key := method + " " + path
			if seen[key] {
				return nil, fmt.Errorf("内置Swagger包含重复接口: %s", key)
			}
			seen[key] = true
			group := "系统管理"
			if len(op.Tags) > 0 && strings.TrimSpace(op.Tags[0]) != "" {
				group = strings.TrimSpace(op.Tags[0])
			}
			description := strings.TrimSpace(op.Summary)
			if description == "" {
				description = strings.TrimSpace(op.Description)
			}
			resources = append(resources, syncResource{Key: key, Path: path, Method: method, ApiGroup: group, Description: description})
		}
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Key < resources[j].Key })
	return resources, nil
}

func apiSyncError(message string) error { return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, message) }

func previewApiSync(resources []syncResource, records []model.SysApi) (*pb.PreviewApiSyncResponse, error) {
	result := &pb.PreviewApiSyncResponse{Added: []*pb.ApiSyncItem{}, Changed: []*pb.ApiSyncItem{}, Obsolete: []*pb.ApiSyncItem{}}
	current := make(map[string]syncResource, len(records))
	snapshot := make([]syncResource, 0, len(records))
	for _, row := range records {
		key := row.Method + " " + row.Path
		if _, ok := current[key]; ok {
			return nil, apiSyncError("现有API资源存在重复路径和方法，请先处理重复数据")
		}
		item := syncResource{Key: key, ID: row.ID, Path: row.Path, Method: row.Method, ApiGroup: row.ApiGroup, Description: row.Description}
		current[key] = item
		snapshot = append(snapshot, item)
	}
	for _, source := range resources {
		item := &pb.ApiSyncItem{Key: source.Key, Path: source.Path, Method: source.Method, ApiGroup: source.ApiGroup, Description: source.Description}
		old, exists := current[source.Key]
		if !exists {
			result.Added = append(result.Added, item)
		} else {
			item.Id, item.CurrentApiGroup, item.CurrentDescription = old.ID, old.ApiGroup, old.Description
			if old.ApiGroup != source.ApiGroup || old.Description != source.Description {
				result.Changed = append(result.Changed, item)
			}
		}
		delete(current, source.Key)
	}
	for _, old := range current {
		result.Obsolete = append(result.Obsolete, &pb.ApiSyncItem{Key: old.Key, Id: old.ID, Path: old.Path, Method: old.Method, ApiGroup: old.ApiGroup, Description: old.Description, CurrentApiGroup: old.ApiGroup, CurrentDescription: old.Description})
	}
	sort.Slice(result.Obsolete, func(i, j int) bool { return result.Obsolete[i].Key < result.Obsolete[j].Key })
	sort.Slice(snapshot, func(i, j int) bool { return snapshot[i].Key < snapshot[j].Key })
	encoded, err := json.Marshal(struct{ Source, Current []syncResource }{resources, snapshot})
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encoded)
	result.Version = hex.EncodeToString(digest[:])
	return result, nil
}

// applyApiSync validates the full selection before changing anything, then only
// creates selected resources or updates their descriptions. It never edits IDs,
// methods, paths, Casbin rules or obsolete resources.
func applyApiSync(tx *gorm.DB, resources []syncResource, request *pb.ApplyApiSyncRequest) (*pb.ApplyApiSyncResponse, error) {
	if request == nil || request.Version == "" || len(request.Keys) == 0 || len(request.Keys) > len(resources) {
		return nil, apiSyncError("请选择需要同步的接口并携带预览版本")
	}
	var records []model.SysApi
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id").Find(&records).Error; err != nil {
		return nil, err
	}
	preview, err := previewApiSync(resources, records)
	if err != nil {
		return nil, err
	}
	if request.Version != preview.Version {
		return nil, apiSyncError("接口资源或Swagger已变化，请重新预览后再同步")
	}
	available := map[string]*pb.ApiSyncItem{}
	for _, item := range preview.Added {
		available[item.Key] = item
	}
	for _, item := range preview.Changed {
		available[item.Key] = item
	}
	selected := make([]*pb.ApiSyncItem, 0, len(request.Keys))
	seen := map[string]bool{}
	for _, key := range request.Keys {
		item, ok := available[key]
		if !ok || seen[key] {
			return nil, apiSyncError("同步选择包含重复、失效或无需更新的接口，请重新预览")
		}
		seen[key] = true
		selected = append(selected, item)
	}
	result := &pb.ApplyApiSyncResponse{}
	for _, item := range selected {
		if item.Id == 0 {
			if err := tx.Create(&model.SysApi{Path: item.Path, Method: item.Method, ApiGroup: item.ApiGroup, Description: item.Description}).Error; err != nil {
				return nil, accessutil.FriendlyDuplicate(err)
			}
			result.Added++
		} else {
			if err := tx.Model(&model.SysApi{}).Where("id = ?", item.Id).Updates(map[string]any{"api_group": item.ApiGroup, "description": item.Description}).Error; err != nil {
				return nil, err
			}
			result.Updated++
		}
	}
	return result, nil
}
