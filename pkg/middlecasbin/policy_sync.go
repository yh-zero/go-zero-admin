package middlecasbin

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const policyChannel = "/casbin"

// PolicyVersion is changed in the same transaction as permission rules. Redis
// notifications merely shorten the time before every instance checks this row.
type PolicyVersion struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement:false"`
	Version   uint64 `gorm:"not null"`
	UpdatedAt time.Time
}

func (PolicyVersion) TableName() string { return "sys_policy_versions" }

func BumpPolicyVersion(tx *gorm.DB) error {
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&PolicyVersion{ID: 1}).Error; err != nil {
		return err
	}
	result := tx.Model(&PolicyVersion{}).Where("id = ?", 1).Updates(map[string]interface{}{"version": gorm.Expr("version + 1"), "updated_at": time.Now()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("policy version row was not updated")
	}
	return nil
}

type PolicySynchronizer struct {
	db        *gorm.DB
	enforcer  *casbin.SyncedCachedEnforcer
	adapter   *boundedPolicyAdapter
	redis     redisclient.UniversalClient
	pubsub    *redisclient.PubSub
	root      context.Context
	cancel    context.CancelFunc
	timeout   time.Duration
	gate      chan struct{}
	wake      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	version   uint64 // guarded by gate
	loaded    bool   // guarded by gate
}

// NewPolicySynchronizer verifies the migration and initial policy load before
// serving requests. All DB work is bounded; Close stops the worker and watcher.
func (c CasbinConf) NewPolicySynchronizer(db *gorm.DB, redisConf redis.RedisConf) (*casbin.SyncedCachedEnforcer, *PolicySynchronizer, error) {
	return c.newPolicySynchronizer(db, redisConf, 2*time.Second, 3*time.Second)
}

func (c CasbinConf) newPolicySynchronizer(db *gorm.DB, redisConf redis.RedisConf, interval, timeout time.Duration) (*casbin.SyncedCachedEnforcer, *PolicySynchronizer, error) {
	root, cancel := context.WithCancel(context.Background())
	s := &PolicySynchronizer{db: db, root: root, cancel: cancel, timeout: timeout, gate: make(chan struct{}, 1), wake: make(chan struct{}, 1), done: make(chan struct{})}
	text := c.ModelText
	if text == "" {
		text = defaultModelText
	}
	m, err := model.NewModelFromString(text)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	adapter := &boundedPolicyAdapter{db: db, root: root, timeout: timeout}
	s.adapter = adapter
	enforcer, err := casbin.NewSyncedCachedEnforcer(m, adapter)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	enforcer.EnableCache(false)
	enforcer.EnableAutoSave(true)
	s.enforcer = enforcer
	if err := s.Sync(root); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("initialize policy synchronization (run database migrations first): %w", err)
	}
	if redisConf.Host != "" {
		options := &redisclient.UniversalOptions{Addrs: strings.Split(redisConf.Host, ","), Username: redisConf.User, Password: redisConf.Pass, DialTimeout: timeout, ReadTimeout: timeout, WriteTimeout: timeout}
		if redisConf.Tls {
			options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		s.redis = redisclient.NewUniversalClient(options)
		s.pubsub = s.redis.Subscribe(root, policyChannel)
	}
	go s.run(interval)
	return enforcer, s, nil
}

// Sync checks durable state even if notifications have been missed. It also
// serves as the enforcement request guard: DB failures fail closed.
func (s *PolicySynchronizer) Sync(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	select {
	case s.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	case <-s.root.Done():
		return s.root.Err()
	}
	defer func() { <-s.gate }()
	for attempt := 0; attempt < 3; attempt++ {
		var before PolicyVersion
		if err := s.db.WithContext(ctx).First(&before, "id = ?", 1).Error; err != nil {
			return err
		}
		if s.loaded && s.version == before.Version {
			return nil
		}
		s.adapter.setContext(ctx)
		loadErr := s.enforcer.LoadPolicy()
		s.adapter.setContext(nil)
		if loadErr != nil {
			return loadErr
		}
		// A commit racing with LoadPolicy must not make a stale snapshot appear
		// current. Store the earlier version, then retry if the row advanced.
		s.version, s.loaded = before.Version, true
		var after PolicyVersion
		if err := s.db.WithContext(ctx).First(&after, "id = ?", 1).Error; err != nil {
			return err
		}
		if after.Version == before.Version {
			return nil
		}
	}
	return errors.New("policy changed repeatedly during reload; retry enforcement")
}

// Notify queues a coalesced local check and publication for the single worker.
// It never performs network work on a committed mutation's request path.
func (s *PolicySynchronizer) Notify() {
	select {
	case <-s.root.Done():
		return
	case s.wake <- struct{}{}:
	default:
	}
}

// Publication is only an optimization; durable polling and enforcement guards
// remain authoritative if Redis is slow or unavailable.
func (s *PolicySynchronizer) publish() {
	if s.redis != nil {
		ctx, cancel := context.WithTimeout(s.root, s.timeout)
		defer cancel()
		if err := s.redis.Publish(ctx, policyChannel, `{"Method":"Update","ID":"policy-version"}`).Err(); err != nil && s.root.Err() == nil {
			logx.Error("policy notification unavailable; durable version polling will retry")
		}
	}
}

func (s *PolicySynchronizer) run(interval time.Duration) {
	defer close(s.done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var messages <-chan interface{}
	if s.pubsub != nil {
		messages = s.pubsub.ChannelWithSubscriptions()
	}
	for {
		publish := false
		select {
		case <-s.root.Done():
			return
		case <-ticker.C:
		case <-s.wake:
			publish = true
		case _, ok := <-messages:
			if !ok {
				messages = nil
				continue
			}
		}
		if err := s.Sync(s.root); err != nil && s.root.Err() == nil {
			logx.Error("policy version synchronization failed; next check will retry")
		}
		if publish && s.root.Err() == nil {
			s.publish()
		}
	}
}

func (s *PolicySynchronizer) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		s.cancel()
		if s.pubsub != nil {
			_ = s.pubsub.Close()
		}
		if s.redis != nil {
			_ = s.redis.Close()
		}
		<-s.done
	})
}

// A read-only adapter prevents accidental direct writes from bypassing version
// increments, and adds cancellation/timeout to Casbin's LoadPolicy interface.
type boundedPolicyAdapter struct {
	db        *gorm.DB
	root      context.Context
	timeout   time.Duration
	contextMu sync.Mutex
	current   context.Context
}

func (a *boundedPolicyAdapter) setContext(ctx context.Context) {
	a.contextMu.Lock()
	a.current = ctx
	a.contextMu.Unlock()
}

func (a *boundedPolicyAdapter) LoadPolicy(m model.Model) error {
	a.contextMu.Lock()
	parent := a.current
	a.contextMu.Unlock()
	if parent == nil {
		parent = a.root
	}
	ctx, cancel := context.WithTimeout(parent, a.timeout)
	defer cancel()
	stop := context.AfterFunc(a.root, cancel)
	defer stop()
	var lines []gormadapter.CasbinRule
	if err := a.db.WithContext(ctx).Order("id").Find(&lines).Error; err != nil {
		return err
	}
	for _, line := range lines {
		if line.Ptype == "" {
			continue
		}
		values := []string{line.Ptype, line.V0, line.V1, line.V2, line.V3, line.V4, line.V5}
		for len(values) > 1 && values[len(values)-1] == "" {
			values = values[:len(values)-1]
		}
		assertion, exists := m[line.Ptype[:1]][line.Ptype]
		if !exists || len(values)-1 != len(assertion.Tokens) {
			return errors.New("database policy does not match the configured model")
		}
		if err := persist.LoadPolicyArray(values, m); err != nil {
			return err
		}
	}
	return nil
}

var errDirectPolicyWrite = errors.New("policy writes must use PolicyTransaction")

func (*boundedPolicyAdapter) SavePolicy(model.Model) error             { return errDirectPolicyWrite }
func (*boundedPolicyAdapter) AddPolicy(string, string, []string) error { return errDirectPolicyWrite }
func (*boundedPolicyAdapter) RemovePolicy(string, string, []string) error {
	return errDirectPolicyWrite
}
func (*boundedPolicyAdapter) RemoveFilteredPolicy(string, string, int, ...string) error {
	return errDirectPolicyWrite
}
