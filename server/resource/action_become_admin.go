package resource

import (
	"github.com/artpar/api2go"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// becomeAdminActionPerformer daptin action implementation
type becomeAdminActionPerformer struct {
	cruds map[string]*DbResource
}

// Name of the action
func (d *becomeAdminActionPerformer) Name() string {
	return "__become_admin"
}

// becomeAdminActionPerformer Perform action and try to make the current user the admin of the system
// Checks CanBecomeAdmin and then invokes BecomeAdmin if true
func (d *becomeAdminActionPerformer) DoAction(request Outcome, inFieldMap map[string]interface{}, transaction *sqlx.Tx) (api2go.Responder, []ActionResponse, []error) {

	if !d.cruds["world"].CanBecomeAdmin(transaction) {
		return nil, nil, []error{errors.New("Unauthorized")}
	}
	u := inFieldMap["user"]
	if u == nil {
		return nil, nil, []error{errors.New("Unauthorized")}
	}
	user := u.(map[string]interface{})

	responseAttrs := make(map[string]interface{})

	if d.cruds["world"].BecomeAdmin(user["id"].(int64), transaction) {
		commitError := transaction.Commit()
		CheckErr(commitError, "failed to rollback")
		responseAttrs["location"] = "/"
		responseAttrs["window"] = "self"
		responseAttrs["delay"] = 7000
	}
	rollbackError := transaction.Rollback()
	CheckErr(rollbackError, "failed to rollback")

	actionResponse := NewActionResponse("client.redirect", responseAttrs)
	_ = OlricCache.Destroy()

	go restart()

	return nil, []ActionResponse{actionResponse}, nil
}

// Create a new action performer for becoming administrator action
func NewBecomeAdminPerformer(initConfig *CmsConfig, cruds map[string]*DbResource) (ActionPerformerInterface, error) {

	handler := becomeAdminActionPerformer{
		cruds: cruds,
	}

	return &handler, nil

}
