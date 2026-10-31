package jwtx

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var ErrSessionInvalid = errors.New("session is invalid or expired")

// Session 只在服务端保存，RefreshHash 不得返回客户端。
type Session struct {
	Id          string `json:"id"`
	UserId      int64  `json:"user_id,string"`
	Username    string `json:"username"`
	DeptId      int64  `json:"dept_id,string"`
	AccessId    string `json:"access_id"`
	RefreshHash string `json:"refresh_hash"`
	CreatedAt   int64  `json:"created_at"`
	LastSeenAt  int64  `json:"last_seen_at"`
	ExpiresAt   int64  `json:"expires_at"`
	Ip          string `json:"ip"`
	UserAgent   string `json:"user_agent"`
	DeviceName  string `json:"device_name"`
}

type SessionStore struct {
	redis  *goredis.Client
	prefix string
}

func NewSessionStore(client *goredis.Client, namespace string) *SessionStore {
	return &SessionStore{redis: client, prefix: "auth:{" + namespace + "}:"}
}

func RandomToken() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func tokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
func (s *SessionStore) sessionKey(id string) string { return s.prefix + "session:" + id }
func (s *SessionStore) userKey(id int64) string     { return fmt.Sprintf("%suser:%d", s.prefix, id) }

// Create 同时创建会话及用户索引，每人最多保留 20 个设备；淘汰最旧设备。
func (s *SessionStore) Create(ctx context.Context, userId int64, username string, deptId int64, ip, ua string, lifetime time.Duration) (*Session, string, error) {
	if userId <= 0 || lifetime < time.Second {
		return nil, "", ErrSessionInvalid
	}
	id, err := RandomToken()
	if err != nil {
		return nil, "", err
	}
	accessId, err := RandomToken()
	if err != nil {
		return nil, "", err
	}
	secret, err := RandomToken()
	if err != nil {
		return nil, "", err
	}
	refresh := id + "." + secret
	now := time.Now().Unix()
	session := &Session{Id: id, UserId: userId, Username: username, DeptId: deptId, AccessId: accessId, RefreshHash: tokenHash(refresh), CreatedAt: now, LastSeenAt: now, ExpiresAt: now + int64(lifetime/time.Second), Ip: truncate(ip, 64), UserAgent: truncate(ua, 512), DeviceName: deviceName(ua)}
	data, err := json.Marshal(session)
	if err != nil {
		return nil, "", err
	}
	const script = `
redis.call('ZREMRANGEBYSCORE', KEYS[2], '-inf', ARGV[3])
local old = redis.call('ZRANGE', KEYS[2], 0, -20)
for _, id in ipairs(old) do redis.call('DEL', ARGV[5] .. id); redis.call('ZREM', KEYS[2], id) end
redis.call('SET', KEYS[1], ARGV[1], 'EXAT', ARGV[2])
redis.call('ZADD', KEYS[2], ARGV[2], ARGV[4])
local latest = redis.call('ZREVRANGE', KEYS[2], 0, 0, 'WITHSCORES')
redis.call('EXPIREAT', KEYS[2], latest[2])
return 1`
	_, err = s.redis.Eval(ctx, script, []string{s.sessionKey(id), s.userKey(userId)}, string(data), session.ExpiresAt, now, id, s.prefix+"session:").Result()
	if err != nil {
		return nil, "", err
	}
	return session, refresh, nil
}

func (s *SessionStore) Validate(ctx context.Context, id string, userId int64, accessId string) (*Session, error) {
	if id == "" || userId <= 0 || accessId == "" {
		return nil, ErrSessionInvalid
	}
	const script = `
local raw = redis.call('GET', KEYS[1]); if not raw then return '' end
local session = cjson.decode(raw)
if session.user_id ~= ARGV[1] or session.access_id ~= ARGV[2] or session.expires_at <= tonumber(ARGV[3]) then return '' end
if session.last_seen_at < tonumber(ARGV[3]) - 30 then
 session.last_seen_at = tonumber(ARGV[3]); raw = cjson.encode(session); redis.call('SET', KEYS[1], raw, 'KEEPTTL')
end
return raw`
	raw, err := s.redis.Eval(ctx, script, []string{s.sessionKey(id)}, fmt.Sprint(userId), accessId, time.Now().Unix()).Text()
	if err != nil {
		return nil, err
	}
	return decodeSession(raw)
}

func (s *SessionStore) GetByRefresh(ctx context.Context, refresh string) (*Session, error) {
	parts := strings.Split(refresh, ".")
	if len(parts) != 2 || len(parts[0]) != 64 || len(parts[1]) != 64 {
		return nil, ErrSessionInvalid
	}
	raw, err := s.redis.Get(ctx, s.sessionKey(parts[0])).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, err
	}
	session, err := decodeSession(raw)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(session.RefreshHash), []byte(tokenHash(refresh))) != 1 {
		return nil, ErrSessionInvalid
	}
	return session, nil
}

// Rotate 在 Redis 内原子比较旧刷新摘要并替换；并发刷新只有一个请求成功。
func (s *SessionStore) Rotate(ctx context.Context, current *Session, oldRefresh, ip, ua string) (*Session, string, error) {
	if current == nil {
		return nil, "", ErrSessionInvalid
	}
	accessId, err := RandomToken()
	if err != nil {
		return nil, "", err
	}
	secret, err := RandomToken()
	if err != nil {
		return nil, "", err
	}
	refresh := current.Id + "." + secret
	next := *current
	next.AccessId, next.RefreshHash, next.LastSeenAt = accessId, tokenHash(refresh), time.Now().Unix()
	next.Ip, next.UserAgent, next.DeviceName = truncate(ip, 64), truncate(ua, 512), deviceName(ua)
	data, err := json.Marshal(next)
	if err != nil {
		return nil, "", err
	}
	const script = `
local raw = redis.call('GET', KEYS[1]); if not raw then return 0 end
local session = cjson.decode(raw)
if session.user_id ~= ARGV[1] or session.refresh_hash ~= ARGV[2] or session.expires_at <= tonumber(ARGV[3]) then return 0 end
redis.call('SET', KEYS[1], ARGV[4], 'KEEPTTL'); return 1`
	result, err := s.redis.Eval(ctx, script, []string{s.sessionKey(current.Id)}, fmt.Sprint(current.UserId), tokenHash(oldRefresh), time.Now().Unix(), string(data)).Int()
	if err != nil {
		return nil, "", err
	}
	if result != 1 {
		return nil, "", ErrSessionInvalid
	}
	return &next, refresh, nil
}

func (s *SessionStore) List(ctx context.Context, userId int64) ([]*Session, error) {
	if userId <= 0 {
		return nil, ErrSessionInvalid
	}
	key := s.userKey(userId)
	if err := s.redis.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprint(time.Now().Unix())).Err(); err != nil {
		return nil, err
	}
	ids, err := s.redis.ZRevRange(ctx, key, 0, 19).Result()
	if err != nil {
		return nil, err
	}
	list := make([]*Session, 0, len(ids))
	for _, id := range ids {
		raw, err := s.redis.Get(ctx, s.sessionKey(id)).Result()
		if errors.Is(err, goredis.Nil) {
			continue
		}
		if err != nil {
			return nil, err
		}
		item, err := decodeSession(raw)
		if err != nil {
			continue
		}
		if item.UserId == userId {
			list = append(list, item)
		}
	}
	return list, nil
}

func (s *SessionStore) Revoke(ctx context.Context, userId int64, id string) error {
	if userId <= 0 || id == "" {
		return ErrSessionInvalid
	}
	const script = `
local raw = redis.call('GET', KEYS[1]); if not raw then redis.call('ZREM', KEYS[2], ARGV[2]); return 1 end
if cjson.decode(raw).user_id ~= ARGV[1] then return 0 end
redis.call('DEL', KEYS[1]); redis.call('ZREM', KEYS[2], ARGV[2]); return 1`
	result, err := s.redis.Eval(ctx, script, []string{s.sessionKey(id), s.userKey(userId)}, fmt.Sprint(userId), id).Int()
	if err != nil {
		return err
	}
	if result != 1 {
		return ErrSessionInvalid
	}
	return nil
}

func (s *SessionStore) RevokeAll(ctx context.Context, userId int64) error {
	if userId <= 0 {
		return ErrSessionInvalid
	}
	const script = `local ids = redis.call('ZRANGE', KEYS[1], 0, -1)
for _, id in ipairs(ids) do redis.call('DEL', ARGV[1] .. id) end
redis.call('DEL', KEYS[1]); return 1`
	return s.redis.Eval(ctx, script, []string{s.userKey(userId)}, s.prefix+"session:").Err()
}

func decodeSession(raw string) (*Session, error) {
	var session Session
	if raw == "" || json.Unmarshal([]byte(raw), &session) != nil || session.ExpiresAt <= time.Now().Unix() {
		return nil, ErrSessionInvalid
	}
	return &session, nil
}

func truncate(value string, max int) string {
	r := []rune(value)
	if len(r) > max {
		return string(r[:max])
	}
	return value
}
func deviceName(ua string) string {
	platform := "浏览器"
	for _, name := range []string{"Android", "iPhone", "iPad", "Windows", "Macintosh", "Linux"} {
		if strings.Contains(ua, name) {
			platform = name
			break
		}
	}
	browser := ""
	for _, pair := range [][2]string{{"Edg/", "Edge"}, {"Firefox/", "Firefox"}, {"Chrome/", "Chrome"}, {"Safari/", "Safari"}} {
		if strings.Contains(ua, pair[0]) {
			browser = pair[1]
			break
		}
	}
	return strings.TrimSpace(platform + " " + browser)
}
