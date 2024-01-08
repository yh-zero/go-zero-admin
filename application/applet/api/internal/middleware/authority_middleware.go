package middleware

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/pkg/ctxJwt"
	"net/http"
	"strconv"
)

// all  *casbin.Enforcer --> *casbin.SyncedCachedEnforce
type AuthorityMiddleware struct {
	CasB        *casbin.SyncedCachedEnforcer
	Rds         *redis.Redis
	BanRoleData map[string]bool
}

func NewAuthorityMiddleware(CasB *casbin.SyncedCachedEnforcer, rds *redis.Redis, banRoleData map[string]bool) *AuthorityMiddleware {
	return &AuthorityMiddleware{
		CasB:        CasB,
		Rds:         rds,
		BanRoleData: banRoleData,
	}
}

func (m *AuthorityMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("==========  AuthorityMiddleware:Handle ===========")
		path := r.URL.Path
		method := r.Method
		authorityId := ctxJwt.GetJwtDataAuthorityId(r.Context())
		fmt.Println("authorityId", authorityId)
		result := batchCheck(m.CasB, strconv.FormatInt(authorityId, 10), path, method)
		if !result {
			logx.Errorf("---------- batchCheck: 不通过 ------------")
			//fmt.Println("---------- batchCheck: 不通过 ------------")
			return
		}
		next(w, r)
	}
}

func batchCheck(CasB *casbin.SyncedCachedEnforcer, authorityId, path, method string) bool {
	//fmt.Println("==== authorityId", authorityId)
	//fmt.Println("==== path", path)
	//fmt.Println("==== method", method)
	ok, _ := CasB.Enforce(authorityId, path, method)
	if !ok {
		_, _ = CasB.AddPolicy(authorityId, path, method) // 临时添加保存通过  方便开发

		logx.Errorf("---------- 权限不足 ------------")
		//fmt.Println("---------- 权限不足 ------------")
		return false
	}
	return true
}

//var (
//	syncedCachedEnforcer *casbin.SyncedCachedEnforcer
//	once                 sync.Once
//	gormDB               *gorm.DB
//)
//
//func Casbin() *casbin.SyncedCachedEnforcer {
//	once.Do(func() {
//		a, err := gormadapter.NewAdapterByDB(gormDB)
//		if err != nil {
//			logx.Errorf("适配数据库失败请检查casbin表是否为InnoDB引擎! err:%v", err)
//			return
//		}
//		text := svc.ServiceContext.Config.CasbinConf.ModelText
//		m, err := model.NewModelFromString(text)
//		if err != nil {
//			logx.Errorf("字符串加载模型失败! err:%v", err)
//			return
//		}
//		syncedCachedEnforcer, _ = casbin.NewSyncedCachedEnforcer(m, a)
//		syncedCachedEnforcer.SetExpireTime(60 * 60)
//		_ = syncedCachedEnforcer.LoadPolicy()
//	})
//	return syncedCachedEnforcer
//}
