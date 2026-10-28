package userlogic

import (
	"context"
	"errors"
	"strconv"

	mysqldriver "github.com/go-sql-driver/mysql"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/logic/datascope"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/audit"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Database identity excludes credentials and query options. Operators must set
// the AI service's effective DSN, rather than inferring it from local table names.
func sameResourceDatabase(applet, ai string) bool {
	if applet == "" || ai == "" {
		return false
	}
	a, err := mysqldriver.ParseDSN(applet)
	if err != nil {
		return false
	}
	b, err := mysqldriver.ParseDSN(ai)
	if err != nil {
		return false
	}
	return a.Net == b.Net && a.Addr == b.Addr && a.DBName != "" && a.DBName == b.DBName
}

func aiResourceAvailability(db *gorm.DB, applet, ai string) (bool, string) {
	if ai == "" {
		return false, "未配置AI实际数据库，AI历史保留且不可移交"
	}
	if !sameResourceDatabase(applet, ai) {
		return false, "AI数据库与业务库不同，暂不支持跨库预览或移交"
	}
	for _, resource := range []any{&agentjobs.Conversation{}, &agentjobs.Message{}, &agentjobs.Run{}} {
		if !db.Migrator().HasTable(resource) {
			return false, "AI历史表不完整，请先完成AI数据库迁移"
		}
	}
	return true, ""
}

func scopedLifecycleUser(ctx context.Context, db *gorm.DB, actor *pb.SessionRequest, id int64, lock bool) (*model.SysUser, error) {
	query, err := datascope.Apply(ctx, db.Model(&model.SysUser{}).Joins("LEFT JOIN sys_user_departments lifecycle_department ON lifecycle_department.user_id = sys_users.id"), actor, "sys_users.id", "lifecycle_department.department_id")
	if err != nil {
		return nil, err
	}
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var user model.SysUser
	err = query.Select("sys_users.id", "sys_users.username", "sys_users.enable", "sys_users.authority_id", "sys_users.session_version").Where("sys_users.id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, userError("用户不存在或不在可访问范围")
	}
	return &user, err
}

func ownedLifecycleFiles(ctx context.Context, db *gorm.DB, actor *pb.SessionRequest, ownerID int64, lock bool) ([]model.SysFileResource, error) {
	query, err := datascope.Apply(ctx, db.Model(&model.SysFileResource{}), actor, "owner_id", "department_id")
	if err != nil {
		return nil, err
	}
	query = query.Where("owner_id = ? AND status <> ?", ownerID, "deleted")
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var rows []model.SysFileResource
	if err := query.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	// Refuse a partial transfer/preview: the UI must not present an incomplete
	// ownership inventory as the full set that will survive account deletion.
	var total int64
	if err := db.Model(&model.SysFileResource{}).Where("owner_id = ? AND status <> ?", ownerID, "deleted").Count(&total).Error; err != nil {
		return nil, err
	}
	if total != int64(len(rows)) {
		return nil, userError("该用户部分文件不在操作者数据范围，无法预览或移交全部资源")
	}
	return rows, nil
}

func previewUserResources(ctx context.Context, db *gorm.DB, in *pb.UserResourcePreviewRequest, appletDSN, aiDSN string) (*pb.UserResourcePreviewResponse, error) {
	if in == nil || in.UserID <= 0 {
		return nil, userError("用户ID无效")
	}
	response := &pb.UserResourcePreviewResponse{TransferMode: "files_only"}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Align with organization/data-scope writes so the inventory and its
		// permission checks use one stable authorized transaction.
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		user, err := scopedLifecycleUser(ctx, tx, in.Actor, in.UserID, false)
		if err != nil {
			return err
		}
		files, err := ownedLifecycleFiles(ctx, tx, in.Actor, in.UserID, false)
		if err != nil {
			return err
		}
		response.UserID, response.Username, response.Files = user.ID, user.Username, int64(len(files))
		response.AIAvailable, response.AIReason = aiResourceAvailability(tx, appletDSN, aiDSN)
		if !response.AIAvailable {
			return nil
		}
		response.TransferMode = "same_database"
		for _, item := range []struct {
			resource any
			count    *int64
		}{{&agentjobs.Conversation{}, &response.AIConversations}, {&agentjobs.Message{}, &response.AIMessages}, {&agentjobs.Run{}, &response.AIRuns}} {
			if err := tx.Model(item.resource).Where("owner_id = ?", in.UserID).Count(item.count).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func transferUserResources(ctx context.Context, db *gorm.DB, in *pb.TransferUserResourcesRequest, appletDSN, aiDSN string) (*pb.TransferUserResourcesResponse, error) {
	if in == nil || in.UserID <= 0 || in.TargetUserID <= 0 || in.UserID == in.TargetUserID || (!in.IncludeFiles && !in.IncludeAI) {
		return nil, userError("请选择不同的有效目标用户及需要移交的资源")
	}
	response := &pb.TransferUserResourcesResponse{TransactionMode: "same_database"}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		// Sorted user locks serialize AI submissions (which lock the owner) and
		// avoid opposite transfers acquiring the two owner rows in reverse order.
		ids := []int64{in.UserID, in.TargetUserID}
		if ids[0] > ids[1] {
			ids[0], ids[1] = ids[1], ids[0]
		}
		for _, id := range ids {
			user, err := scopedLifecycleUser(ctx, tx, in.Actor, id, true)
			if err != nil {
				return err
			}
			if id == in.TargetUserID && user.Enable != 1 {
				return userError("目标用户必须处于启用状态")
			}
			if id == in.TargetUserID {
				var roleIDs []int64
				if err := tx.Model(&model.SysUserAuthority{}).Where("sys_user_id = ?", id).Pluck("sys_authority_authority_id", &roleIDs).Error; err != nil {
					return err
				}
				if _, err := validateUserAuthorities(tx, roleIDs, user.AuthorityId); err != nil {
					return userError("目标用户角色配置无效，请先修复账号角色")
				}
			}
		}
		var department model.SysUserDepartment
		err := tx.Where("user_id = ?", in.TargetUserID).First(&department).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			var validDepartment model.SysDepartment
			if err := tx.Where("id = ? AND status = 1", department.DepartmentID).First(&validDepartment).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return userError("目标用户所属部门不存在或已停用")
				}
				return err
			}
		}
		if in.IncludeAI {
			if available, reason := aiResourceAvailability(tx, appletDSN, aiDSN); !available {
				return userError(reason)
			}
			// Never change owners while a worker is queued/running. Locking reads
			// observe current state after waiting, even under MySQL REPEATABLE READ.
			var runs []agentjobs.Run
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "status").Where("owner_id = ?", in.UserID).Order("id").Find(&runs).Error; err != nil {
				return err
			}
			for _, run := range runs {
				if !agentjobs.IsTerminal(run.Status) {
					return userError("AI任务仍在执行或排队，请结束后再移交")
				}
			}
		}
		if in.IncludeFiles {
			files, err := ownedLifecycleFiles(ctx, tx, in.Actor, in.UserID, true)
			if err != nil {
				return err
			}
			ids := make([]int64, 0, len(files))
			for _, file := range files {
				if file.Status != "active" {
					return userError("文件正在删除，无法移交")
				}
				ids = append(ids, file.ID)
			}
			if len(ids) > 0 {
				operation := tx.Model(&model.SysFileResource{}).Where("id IN ? AND owner_id = ? AND status = ?", ids, in.UserID, "active").Updates(map[string]any{"owner_id": in.TargetUserID, "department_id": department.DepartmentID})
				if operation.Error != nil {
					return operation.Error
				}
				response.Files = operation.RowsAffected
			}
		}
		if in.IncludeAI {
			for _, item := range []struct {
				resource any
				count    *int64
			}{{&agentjobs.Conversation{}, &response.AIConversations}, {&agentjobs.Message{}, &response.AIMessages}, {&agentjobs.Run{}, &response.AIRuns}} {
				operation := tx.Model(item.resource).Where("owner_id = ?", in.UserID).Update("owner_id", in.TargetUserID)
				if operation.Error != nil {
					return operation.Error
				}
				*item.count = operation.RowsAffected
			}
		}
		var actor model.SysUser
		if err := tx.Select("id", "username").First(&actor, in.Actor.UserID).Error; err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Event{ActorID: in.Actor.UserID, ActorName: actor.Username, AuthorityID: in.Actor.AuthorityId, Module: "user", Action: "transferUserResources", Object: strconv.FormatInt(in.UserID, 10) + "->" + strconv.FormatInt(in.TargetUserID, 10)})
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}
