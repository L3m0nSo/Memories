package dbresourceinterface

import (
	"github.com/L3m0nSo/Memories/server/actionresponse"
	"github.com/L3m0nSo/Memories/server/assetcachepojo"
	"github.com/L3m0nSo/Memories/server/database"
	daptinid "github.com/L3m0nSo/Memories/server/id"
	"github.com/L3m0nSo/Memories/server/permission"
	"github.com/L3m0nSo/Memories/server/rootpojo"
	"github.com/L3m0nSo/Memories/server/table_info"
	"github.com/artpar/api2go/v2"
	"github.com/jmoiron/sqlx"
)

type DbResourceInterface interface {
	GetAllObjects(name string, transaction *sqlx.Tx) ([]map[string]interface{}, error)
	GetObjectPermissionByReferenceId(name string, ref daptinid.DaptinReferenceId, tx *sqlx.Tx) permission.PermissionInstance
	TableInfo() *table_info.TableInfo
	GetAdminEmailId(transaction *sqlx.Tx) string
	Connection() database.DatabaseConnection
	HandleActionRequest(request actionresponse.ActionRequest, data api2go.Request, transaction1 *sqlx.Tx) ([]actionresponse.ActionResponse, error)
	GetActionHandler(name string) actionresponse.ActionPerformerInterface
	GetCredentialByName(credentialName string, transaction *sqlx.Tx) (*Credential, error)
	SubsiteFolderCache(id daptinid.DaptinReferenceId) (*assetcachepojo.AssetFolderCache, bool)
	SyncStorageToPath(store rootpojo.CloudStore, name string, path string, transaction *sqlx.Tx) error
}
