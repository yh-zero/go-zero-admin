package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"go-zero-admin/pkg/ctxJwt"

	"github.com/casbin/casbin/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// all  *casbin.Enforcer --> *casbin.SyncedCachedEnforce
type AuthorityMiddleware struct {
	CasB *casbin.SyncedCachedEnforcer
	Rds  *redis.Redis
}

func NewAuthorityMiddleware(CasB *casbin.SyncedCachedEnforcer, rds *redis.Redis) *AuthorityMiddleware {
	return &AuthorityMiddleware{
		CasB: CasB,
		Rds:  rds,
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
	ok, _ := CasB.Enforce(authorityId, path, method)
	if !ok {
		//_, _ = CasB.AddPolicy(authorityId, path, method) // 如果权限数据不小心清了 把这个开启  然后api请求两次就会有权限  最后重新设置权限即可

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
