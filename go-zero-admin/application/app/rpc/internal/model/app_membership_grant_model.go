package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AppMembershipGrantModel = (*customAppMembershipGrantModel)(nil)

type (
	// AppMembershipGrantModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAppMembershipGrantModel.
	AppMembershipGrantModel interface {
		appMembershipGrantModel
	}

	customAppMembershipGrantModel struct {
		*defaultAppMembershipGrantModel
	}
)

// NewAppMembershipGrantModel returns a model for the database table.
func NewAppMembershipGrantModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AppMembershipGrantModel {
	return &customAppMembershipGrantModel{
		defaultAppMembershipGrantModel: newAppMembershipGrantModel(conn, c, opts...),
	}
}
