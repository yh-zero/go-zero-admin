package middlecasbin

import (
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

// all  *casbin.Enforcer --> *casbin.SyncedCachedEnforcer
func (l CasbinConf) MustNewCasbin(dsn string) *casbin.SyncedCachedEnforcer {
	csb, err := l.NewCasbin(dsn)
	if err != nil {
		logx.Errorw("initialize Casbin failed", logx.Field("detail", err.Error()))
		return nil
	}

	return csb
}

func (l CasbinConf) NewCasbin(dsn string) (*casbin.SyncedCachedEnforcer, error) {
	adapter, err := gormadapter.NewAdapter("mysql", dsn, true)
	logx.Must(err)

	var text string
	if l.ModelText == "" {
		text = `
		[request_definition]
		r = sub, obj, act
		
		[policy_definition]
		p = sub, obj, act
		
		[role_definition]
		g = _, _
		
		[policy_effect]
		e = some(where (p.eft == allow))
		
		[matchers]
		m = r.sub == p.sub && keyMatch2(r.obj,p.obj) && r.act == p.act
		`
	} else {
		text = l.ModelText
	}

	m, err := model.NewModelFromString(text)
	if err != nil {
		logx.Errorf("字符串加载模型失败! err:%v", err)
		logx.Must(err)
	}

	// NewSyncedCachedEnforcer 在 NewEnforcer 的基础上增加了同步缓存的功能，提供了更好的性能和并发安全性，适用于对性能要求较高的场景
	//enforcer, err := casbin.NewEnforcer(m, adapter)

	enforcer, err := casbin.NewSyncedCachedEnforcer(m, adapter)
	enforcer.SetExpireTime(60 * 60)

	err = enforcer.LoadPolicy()
	if err != nil {
		logx.Errorf("创建 Enforcer（访问控制器）的函数 失败! err:%v", err)
		logx.Must(err)
	}

	return enforcer, nil
}

func (l CasbinConf) MustNewCasbinWithRedisWatcher(dsn string, c redis.RedisConf) *casbin.SyncedCachedEnforcer {
	cbn := l.MustNewCasbin(dsn)
	w := l.MustNewRedisWatcher(c, func(data string) {
		rediswatcher.DefaultUpdateCallback(cbn)(data)
	})
	err := cbn.SetWatcher(w)
	logx.Must(err)
	err = cbn.SavePolicy()
	logx.Must(err)
	return cbn
}

func (l CasbinConf) MustNewRedisWatcher(c redis.RedisConf, f func(string2 string)) persist.Watcher {
	w, err := rediswatcher.NewWatcher(c.Host, rediswatcher.WatcherOptions{
		Options: redis2.Options{
			Network:  "tcp",
			Password: c.Pass,
		},
		Channel:    "/casbin",
		IgnoreSelf: false,
	})
	logx.Must(err)

	err = w.SetUpdateCallback(f)
	logx.Must(err)

	return w
}
