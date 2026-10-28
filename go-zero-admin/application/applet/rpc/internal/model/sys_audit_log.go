package model

import "go-zero-admin/pkg/audit"

// SysAuditLog shares the transaction-safe event model with business services.
type SysAuditLog = audit.Event
