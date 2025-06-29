package actions

import (
	"github.com/L3m0nSo/Memories/server/actionresponse"
	"github.com/L3m0nSo/Memories/server/resource"
	"github.com/artpar/api2go/v2"
	"github.com/jmoiron/sqlx"
)

/*
*

	Become administrator of daptin action implementation
*/
type graphqlEnableActionPerformer struct {
	cruds map[string]*resource.DbResource
}

// Name of the action
func (d *graphqlEnableActionPerformer) Name() string {
	return "__enable_graphql"
}

// Perform action and try to make the current user the admin of the system
// Checks CanGraphqlEnable and then invokes GraphqlEnable if true
func (d *graphqlEnableActionPerformer) DoAction(request actionresponse.Outcome, inFieldMap map[string]interface{}, transaction *sqlx.Tx) (api2go.Responder, []actionresponse.ActionResponse, []error) {

	err := d.cruds["world"].ConfigStore.SetConfigValueForWithTransaction("graphql.enable", "true", "backend", transaction)

	if err != nil {
		//go Restart()

		return nil, []actionresponse.ActionResponse{resource.NewActionResponse("client.notify",
			resource.NewClientNotification("message", "Restarting with graphql enabled", "Success"))}, nil
	} else {
		return nil, []actionresponse.ActionResponse{resource.NewActionResponse("client.notify",
			resource.NewClientNotification("message", "Failed to update config: "+err.Error(), "Failed"))}, nil
	}
}

// Create a new action performer for becoming administrator action
func NewGraphqlEnablePerformer(initConfig *resource.CmsConfig, cruds map[string]*resource.DbResource) (actionresponse.ActionPerformerInterface, error) {

	handler := graphqlEnableActionPerformer{
		cruds: cruds,
	}

	return &handler, nil

}
