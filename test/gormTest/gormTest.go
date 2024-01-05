package gormTest

//
//import (
//	"context"
//	"fmt"
//	"strconv"
//	"time"
//
//	"gorm.io/gorm"
//)
//
//type tenantModel interface {
//	Insert(ctx context.Context, data *Tenant) error
//	FindOne(ctx context.Context, id int64) (*Tenant, error)
//	FindOneByUserId(ctx context.Context, userId int64) (*Tenant, error)
//	Update(ctx context.Context, data *Tenant) error
//	Delete(ctx context.Context, id int64) error
//}
//
//type defaultTenantModel struct {
//	db          *gorm.DB
//	cache       cache.Cache
//	table       string
//	cachePrefix string
//}
//
//func NewTenantModel(db *gorm.DB, c cache.Cache) *defaultTenantModel {
//	return &defaultTenantModel{
//		db:          db,
//		cache:       c,
//		table:       "tenant",
//		cachePrefix: "tenant:",
//	}
//}
//
//func (m *defaultTenantModel) Insert(ctx context.Context, data *Tenant) error {
//	// 插入到数据库
//	err := m.db.WithContext(ctx).Create(data).Error
//	if err != nil {
//		return err
//	}
//
//	// 将数据存入缓存
//	cacheKey := m.cachePrefix + strconv.FormatInt(data.Id, 10)
//	if err := m.cache.Set(ctx, cacheKey, data, 24*time.Hour); err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func (m *defaultTenantModel) FindOne(ctx context.Context, id int64) (*Tenant, error) {
//	// 尝试从缓存获取
//	cacheKey := m.cachePrefix + strconv.FormatInt(id, 10)
//	var tenant Tenant
//	if err := m.cache.Get(ctx, cacheKey, &tenant); err == nil {
//		return &tenant, nil
//	}
//
//	// 从数据库获取
//	err := m.db.WithContext(ctx).Table(m.table).Where("id = ?", id).First(&tenant).Error
//	if err != nil {
//		return nil, err
//	}
//
//	// 将数据存入缓存
//	if err := m.cache.Set(ctx, cacheKey, &tenant, 24*time.Hour); err != nil {
//		// 如果缓存设置失败，这可能不是关键错误，可以记录日志并继续
//		fmt.Println("Failed to set cache for tenant with id:", id, "Error:", err)
//	}
//
//	return &tenant, nil
//}
//
//// 其他方法的实现与FindOne方法类似，即先从缓存尝试获取，如果未命中则从数据库获取，并在获取后存入缓存。
