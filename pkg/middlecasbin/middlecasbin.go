package middlecasbin

import (
	"errors"
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	redis2 "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type CasbinConf struct {
	ModelText string `json:"ModelText,optional,env=CASBIN_MODEL_TEXT"`
}

// defaultModelText 默认模型 与各服务yaml中的CasbinConf.ModelText保持一致
// 路径匹配统一使用casbin内置keyMatch2 支持 /api/:id 风格的通配符
const defaultModelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && r.act == p.act
`

var (
	syncedCachedEnforcer *casbin.SyncedCachedEnforcer
	once                 sync.Once // 进程内单例 同一进程只创建一个enforcer
)

// MustNewCasbin 创建enforcer 失败直接panic快速失败 避免服务带着nil enforcer启动后请求panic
func (l CasbinConf) MustNewCasbin(dsn string) *casbin.SyncedCachedEnforcer {
	csb, err := l.NewCasbin(dsn)
	if err != nil {
		logx.Errorw("initialize Casbin failed", logx.Field("detail", err.Error()))
		panic(err)
	}

	return csb
}

// NewCasbin 创建SyncedCachedEnforcer 在NewEnforcer的基础上增加了同步缓存的功能
func (l CasbinConf) NewCasbin(dsn string) (*casbin.SyncedCachedEnforcer, error) {
	var initErr error
	once.Do(func() {
		adapter, err := gormadapter.NewAdapter("mysql", dsn, true)
		if err != nil {
			initErr = err
			return
		}

		text := l.ModelText
		if text == "" {
			text = defaultModelText
		}

		m, err := model.NewModelFromString(text)
		if err != nil {
			initErr = err
			return
		}

		enforcer, err := casbin.NewSyncedCachedEnforcer(m, adapter)
		if err != nil {
			initErr = err
			return
		}
		// Policies already live in memory. Avoid caching allow/deny decisions so a
		// concurrent policy reload cannot retain a result from the old permissions.
		enforcer.EnableCache(false)

		if err = enforcer.LoadPolicy(); err != nil {
			initErr = err
			return
		}
		syncedCachedEnforcer = enforcer
	})

	// once已消耗但enforcer仍为nil 说明此前初始化失败过(initErr是局部变量此时为nil)
	// 补充错误 避免调用方拿到(nil, nil)后在运行时panic
	if syncedCachedEnforcer == nil && initErr == nil {
		initErr = errors.New("casbin enforcer not initialized, previous init failed")
	}

	return syncedCachedEnforcer, initErr
}

// MustNewCasbinWithRedisWatcher 创建带redis watcher的enforcer 用于多实例间策略同步
// 注意: EnableAutoSave后写操作会自动落库并广播 不要调用SavePolicy全量写回(有清空线上策略表的风险)
func (l CasbinConf) MustNewCasbinWithRedisWatcher(dsn string, c redis.RedisConf) *casbin.SyncedCachedEnforcer {
	cbn := l.MustNewCasbin(dsn)
	w := l.MustNewRedisWatcher(c, func(data string) {
		rediswatcher.DefaultUpdateCallback(cbn)(data)
	})
	err := cbn.SetWatcher(w)
	logx.Must(err)
	cbn.EnableAutoSave(true)
	return cbn
}

// MustNewRedisWatcher 创建redis watcher IgnoreSelf=true 过滤自己发出的通知 避免重复LoadPolicy
func (l CasbinConf) MustNewRedisWatcher(c redis.RedisConf, f func(string2 string)) persist.Watcher {
	w, err := rediswatcher.NewWatcher(c.Host, rediswatcher.WatcherOptions{
		Options: redis2.Options{
			Network:  "tcp",
			Password: c.Pass,
		},
		Channel:    "/casbin",
		IgnoreSelf: true,
	})
	logx.Must(err)

	err = w.SetUpdateCallback(f)
	logx.Must(err)

	return w
}
