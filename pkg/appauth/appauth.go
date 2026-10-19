package appauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const Issuer = "go-zero-admin/app"
const Audience = "app-client"

type Config struct {
	EmailVerificationRequired bool `json:",default=false"`
	AccessSecret              string
	AccessExpire              int64 `json:",default=900"`
	RefreshExpire             int64 `json:",default=2592000"`
}

func (c Config) Validate() error {
	if len(c.AccessSecret) < 32 || c.AccessExpire <= 0 || c.RefreshExpire <= c.AccessExpire {
		return errors.New("AppJwtAuth requires a separate secret of at least 32 bytes and valid token lifetimes")
	}
	return nil
}

type Claims struct {
	UserID         int64  `json:"uid"`
	SessionID      string `json:"sid"`
	SessionVersion int64  `json:"sv"`
	Generation     int64  `json:"gen"`
	Kind           string `json:"kind"`
	jwt.RegisteredClaims
}

func Issue(c Config, userID int64, sessionID string, version, generation int64, kind string, expires time.Time) (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	claims := Claims{UserID: userID, SessionID: sessionID, SessionVersion: version, Generation: generation, Kind: kind, RegisteredClaims: jwt.RegisteredClaims{Issuer: Issuer, Audience: jwt.ClaimStrings{Audience}, Subject: strconv.FormatInt(userID, 10), ID: hex.EncodeToString(nonce), IssuedAt: jwt.NewNumericDate(time.Now().UTC()), ExpiresAt: jwt.NewNumericDate(expires)}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(c.AccessSecret))
}
func Parse(c Config, raw, kind string) (*Claims, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if len(raw) == 0 || len(raw) > 4096 {
		return nil, errors.New("invalid App token")
	}
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing algorithm")
		}
		return []byte(c.AccessSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !parsed.Valid || claims.Issuer != Issuer || !claims.VerifyAudience(Audience, true) || claims.Kind != kind || claims.ExpiresAt == nil || claims.IssuedAt == nil || claims.ID == "" || claims.UserID <= 0 || claims.SessionID == "" || claims.SessionVersion <= 0 || claims.Generation <= 0 || claims.Subject != strconv.FormatInt(claims.UserID, 10) {
		return nil, errors.New("invalid App token")
	}
	return claims, nil
}
func Digest(raw string) string {
	value := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(value[:])
}

var mainlandPhone = regexp.MustCompile(`^1[3-9][0-9]{9}$`)
var e164Phone = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

func NormalizeAccount(raw string) (kind, identifier string, err error) {
	value := strings.TrimSpace(raw)
	if strings.Contains(value, "@") {
		value = strings.ToLower(value)
		address, e := mail.ParseAddress(value)
		parts := strings.Split(value, "@")
		if e != nil || address.Address != value || len(value) > 254 || len(parts) != 2 || !strings.Contains(parts[1], ".") || strings.ContainsAny(value, " <>\t\r\n") {
			return "", "", errors.New("请输入有效邮箱或手机号")
		}
		return "email", value, nil
	}
	if mainlandPhone.MatchString(value) {
		return "phone", "+86" + value, nil
	}
	if e164Phone.MatchString(value) {
		return "phone", value, nil
	}
	return "", "", errors.New("请输入有效邮箱或手机号")
}
