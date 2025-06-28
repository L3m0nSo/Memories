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
type makeResponsePerformer struct {
}

// Name of the action
func (d *makeResponsePerformer) Name() string {
	return "response.create"
}

// Perform action and try to make the current user the admin of the system
// Checks CanBecomeAdmin and then invokes BecomeAdmin if true
func (d *makeResponsePerformer) DoAction(request actionresponse.Outcome, inFieldMap map[string]interface{}, transaction *sqlx.Tx) (api2go.Responder, []actionresponse.ActionResponse, []error) {
	responseType, ok := inFieldMap["response_type"]
	if !ok {
		responseType = request.Type
	}
	return nil, []actionresponse.ActionResponse{
		resource.NewActionResponse(responseType.(string), inFieldMap),
	}, nil
}

// Create a new action performer for becoming administrator action
func NewMakeResponsePerformer() (actionresponse.ActionPerformerInterface, error) {

	handler := makeResponsePerformer{}

	return &handler, nil

}
