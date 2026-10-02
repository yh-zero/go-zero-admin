// Package audit stores a small, allowlisted audit trail. It never accepts raw
// request/response bodies, credentials, or authentication headers as event data.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

const MaxParamsBytes = 4096

// Event is append-only; there is deliberately no update/delete API or soft delete.
type Event struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	ActorID     int64     `gorm:"not null;default:0;index:idx_audit_actor_time,priority:1"`
	ActorName   string    `gorm:"size:64;not null;default:''"`
	AuthorityID int64     `gorm:"not null;default:0"`
	EventType   string    `gorm:"size:16;not null;index:idx_audit_type_time,priority:1"`
	Module      string    `gorm:"size:64;not null;index:idx_audit_module_time,priority:1"`
	Action      string    `gorm:"size:64;not null"`
	Object      string    `gorm:"size:256;not null;default:''"`
	Path        string    `gorm:"size:256;not null;default:''"`
	Method      string    `gorm:"size:16;not null;default:''"`
	Result      string    `gorm:"size:16;not null;index:idx_audit_result_time,priority:1"`
	StatusCode  int64     `gorm:"not null;default:0"`
	IP          string    `gorm:"size:64;not null;default:''"`
	TraceID     string    `gorm:"size:64;not null;default:''"`
	DurationMs  int64     `gorm:"not null;default:0"`
	Params      string    `gorm:"type:text;not null"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime;index:idx_audit_time;index:idx_audit_actor_time,priority:2;index:idx_audit_type_time,priority:2;index:idx_audit_module_time,priority:2;index:idx_audit_result_time,priority:2"`
}

func (Event) TableName() string { return "sys_audit_logs" }

// Record can use the business transaction, so failed audit inserts also roll back
// sensitive changes. IDs and event timestamps always come from this service.
func Record(ctx context.Context, db *gorm.DB, event Event) error {
	if db == nil {
		return errors.New("audit database unavailable")
	}
	actor := ActorFromContext(ctx)
	if event.ActorID == 0 && event.ActorName == "" {
		event.ActorID, event.ActorName, event.AuthorityID = actor.ID, actor.Name, actor.AuthorityID
	}
	request := RequestFromContext(ctx)
	if event.Path == "" {
		event.Path = request.Path
	}
	if event.Method == "" {
		event.Method = request.Method
	}
	if event.IP == "" {
		event.IP = request.IP
	}
	if event.TraceID == "" {
		event.TraceID = request.TraceID
	}
	if err := Normalize(&event); err != nil {
		return err
	}
	event.ID = 0
	event.CreatedAt = time.Now().UTC()
	return db.WithContext(ctx).Create(&event).Error
}

func Normalize(event *Event) error {
	if event == nil {
		return errors.New("audit event required")
	}
	if event.EventType == "" {
		event.EventType = "operation"
	}
	if event.EventType != "operation" && event.EventType != "login" {
		return errors.New("invalid audit event type")
	}
	if event.Result == "" {
		event.Result = "success"
	}
	if event.Result != "success" && event.Result != "failure" {
		return errors.New("invalid audit result")
	}
	if event.ActorID < 0 || event.AuthorityID < 0 || event.DurationMs < 0 || event.StatusCode < 0 || event.StatusCode > 599 {
		return errors.New("invalid audit numeric field")
	}
	event.Module, event.Action = clean(event.Module, 64), clean(event.Action, 64)
	if event.Module == "" || event.Action == "" {
		return errors.New("audit module and action required")
	}
	event.ActorName = clean(event.ActorName, 64)
	event.Object = clean(event.Object, 256)
	event.Path = clean(strings.SplitN(strings.SplitN(event.Path, "?", 2)[0], "#", 2)[0], 256)
	event.Method = clean(strings.ToUpper(event.Method), 16)
	event.IP, event.TraceID = clean(event.IP, 64), clean(event.TraceID, 64)
	// Sanitize on the persistence side as well, so an RPC caller cannot bypass
	// the HTTP middleware allowlist by passing arbitrary JSON in Params.
	event.Params = SanitizeJSON([]byte(event.Params))
	return nil
}

func clean(value string, max int) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
	runes := []rune(value)
	if len(runes) > max {
		return string(runes[:max])
	}
	return value
}

// These values help identify the modified resource while excluding names,
// descriptions, phone/email, passwords, tokens and captcha/OTP values.
var allowedKeys = map[string]bool{
	"id": true, "ids": true, "userid": true, "authorityid": true,
	"authorityids": true, "menuids": true, "menubtnids": true,
	"parentid": true, "sysbasemenuid": true, "sysdictionaryid": true,
	"status": true, "enable": true, "method": true,
	"pageno": true, "pagesize": true,
}

func SanitizeJSON(data []byte) string {
	if len(data) == 0 || len(data) > 16*1024 || !utf8.Valid(data) {
		return "{}"
	}
	var input map[string]json.RawMessage
	if json.Unmarshal(data, &input) != nil {
		return "{}"
	}
	output := map[string]any{}
	for key, raw := range input {
		key = strings.ToLower(key)
		if !allowedKeys[key] {
			continue
		}
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if decoder.Decode(&value) != nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if key == "method" {
				switch strings.ToUpper(typed) {
				case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
					output[key] = strings.ToUpper(typed)
				}
				continue
			}
			// Legacy menuIds is a comma-separated string. Other arbitrary
			// strings are dropped even when placed under an allowlisted key.
			if len(typed) <= 256 && numericIDs(typed) {
				output[key] = typed
			}
		case json.Number:
			if number, err := typed.Int64(); err == nil && number >= 0 {
				output[key] = typed
			}
		case bool:
			if key == "status" || key == "enable" {
				output[key] = typed
			}
		case []any:
			if len(typed) > 100 {
				continue
			}
			valid := true
			for _, item := range typed {
				number, ok := item.(json.Number)
				if numberValue, err := number.Int64(); !ok || err != nil || numberValue < 0 {
					valid = false
				}
			}
			if valid {
				output[key] = typed
			}
		}
	}
	encoded, err := json.Marshal(output)
	if err != nil || len(encoded) > MaxParamsBytes {
		return "{}"
	}
	return string(encoded)
}

func numericIDs(value string) bool {
	if value == "" {
		return false
	}
	for _, part := range strings.Split(value, ",") {
		if number, err := strconv.ParseInt(part, 10, 64); err != nil || number < 0 {
			return false
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return false
			}
		}
	}
	return true
}
