package rootpojo

import (
	"time"

	daptinid "github.com/L3m0nSo/Memories/server/id"
	"github.com/L3m0nSo/Memories/server/permission"
)

type CloudStore struct {
	Id              int64
	RootPath        string
	StoreParameters map[string]interface{}
	UserId          daptinid.DaptinReferenceId
	CredentialName  string
	Name            string
	StoreType       string
	StoreProvider   string
	Version         int
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
	DeletedAt       *time.Time
	ReferenceId     daptinid.DaptinReferenceId
	Permission      permission.PermissionInstance
}
