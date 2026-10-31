package tenantservicelogic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/tenant/rpc/internal/model"
	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/objectstorage"
	"go-zero-admin/pkg/xerr"
	"os"
	"strings"
	"time"
)

type DeleteTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteTenantLogic(ctx context.Context, s *svc.ServiceContext) *DeleteTenantLogic {
	return &DeleteTenantLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *DeleteTenantLogic) DeleteTenant(in *tenantpb.DeleteTenantRequest) (*tenantpb.NoDataResponse, error) {
	tenant, release, err := managedTenant(l.ctx, l.svcCtx, in.Actor, in.TenantID)
	if err != nil {
		return nil, err
	}
	defer release()
	if in.ConfirmSlug != tenant.Slug {
		return nil, xerr.NewErrCodeMsg(400, "确认标识不匹配，删除已取消")
	}
	ctx, cancel := context.WithTimeout(l.ctx, 90*time.Second)
	defer cancel()
	if err = routeTenant(ctx, l.svcCtx, tenant, false); err != nil {
		return nil, tenantFailure(l.svcCtx, tenant, "删除租户路由失败")
	}
	if _, statErr := os.Stat(tenantEnvPath(l.svcCtx, tenant)); statErr == nil {
		if _, err = compose(ctx, l.svcCtx, tenant, nil, "down", "--volumes", "--remove-orphans", "--timeout", "10"); err != nil {
			return nil, tenantFailure(l.svcCtx, tenant, "删除租户容器失败")
		}
	} else if !os.IsNotExist(statErr) || !noTenantResources(ctx, tenant) {
		return nil, tenantFailure(l.svcCtx, tenant, "租户运行配置缺失，不能安全删除资源")
	}
	storage, err := tenantStorage(l.svcCtx, tenant)
	if err != nil {
		return nil, tenantFailure(l.svcCtx, tenant, "租户存储配置不可用")
	}
	if err = deleteTenantBuckets(ctx, storage, tenant); err != nil {
		return nil, tenantFailure(l.svcCtx, tenant, "删除租户存储空间失败")
	}
	if err = os.Remove(tenantEnvPath(l.svcCtx, tenant)); err != nil && !os.IsNotExist(err) {
		return nil, tenantFailure(l.svcCtx, tenant, "删除租户运行配置失败")
	}
	tenant.DeletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	tenant.Status, tenant.ErrorMsg, tenant.AdminAccountReady = "stopped", "", 0
	if err = l.svcCtx.TenantRegistryModel.Update(ctx, tenant); err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &tenantpb.NoDataResponse{}, nil
}
func noTenantResources(ctx context.Context, t *model.TenantRegistry) bool {
	for _, args := range [][]string{{"ps", "-aq", "--filter", "label=com.docker.compose.project=" + t.ComposeProject}, {"volume", "ls", "-q", "--filter", "label=com.docker.compose.project=" + t.ComposeProject}, {"network", "ls", "-q", "--filter", "label=com.docker.compose.project=" + t.ComposeProject}} {
		output, err := dockerCommand(ctx, nil, args...)
		if err != nil || strings.TrimSpace(string(output)) != "" {
			return false
		}
	}
	return true
}
func deleteTenantBuckets(ctx context.Context, config objectstorage.Config, t *model.TenantRegistry) error {
	client := tenantS3(config)
	for _, bucket := range []string{config.PublicBucket, config.PrivateBucket} {
		tags, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: aws.String(bucket)})
		if missingBucket(err) {
			continue
		}
		if err != nil {
			return errors.New("cannot verify tenant bucket owner")
		}
		project, id := "", ""
		for _, tag := range tags.TagSet {
			if aws.ToString(tag.Key) == "TenantProject" {
				project = aws.ToString(tag.Value)
			}
			if aws.ToString(tag.Key) == "TenantRegistryId" {
				id = aws.ToString(tag.Value)
			}
		}
		// 相同名字但没有当前登记的所有权标签，绝不删除。
		if project != t.ComposeProject || id != fmt.Sprint(t.Id) {
			continue
		}
		for {
			objects, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(bucket), MaxKeys: aws.Int32(1000)})
			if err != nil {
				return errors.New("cannot list tenant objects")
			}
			if len(objects.Contents) == 0 {
				break
			}
			for _, object := range objects.Contents {
				if _, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: object.Key}); err != nil {
					return errors.New("cannot delete tenant object")
				}
			}
		}
		if _, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)}); err != nil {
			return errors.New("cannot delete tenant bucket")
		}
	}
	return nil
}
