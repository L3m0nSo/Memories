package actions

import (
	"fmt"

	"github.com/L3m0nSo/Memories/server/actionresponse"
	"github.com/L3m0nSo/Memories/server/resource"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	//"golang.org/x/oauth2"
	"time"

	"github.com/artpar/api2go/v2"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/oauth2"
)

type oauthLoginBeginActionPerformer struct {
	responseAttrs map[string]interface{}
	cruds         map[string]*resource.DbResource
	configStore   *resource.ConfigStore
	otpKey        string
}

func (d *oauthLoginBeginActionPerformer) Name() string {
	return "oauth.client.redirect"
}

func (d *oauthLoginBeginActionPerformer) DoAction(request actionresponse.Outcome, inFieldMap map[string]interface{}, transaction *sqlx.Tx) (api2go.Responder, []actionresponse.ActionResponse, []error) {

	state, err := totp.GenerateCodeCustom(d.otpKey, time.Now(), totp.ValidateOpts{
		Period:    300,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		log.Errorf("Failed to generate code: %v", err)
		return nil, nil, []error{err}
	}

	authConnectorData := inFieldMap["authenticator"].(string)

	//redirectUri := authConnectorData["redirect_uri"].(string)
	//
	//if strings.Index(redirectUri, "?") > -1 {
	//	redirectUri = redirectUri + "&authenticator=" + authConnectorData["name"].(string)
	//} else {
	//	redirectUri = redirectUri + "?authenticator=" + authConnectorData["name"].(string)
	//}

	conf, _, err := GetOauthConnectionDescription(authConnectorData, d.cruds["oauth_connect"], transaction)
	resource.CheckErr(err, "Failed to get oauth.conf from authenticator name")

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	var url string
	if len(conf.Scopes) > 1 {
		url = conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
	} else {
		url = conf.AuthCodeURL(state)

	}
	fmt.Printf("Visit the URL for the auth dialog: %v", url)

	responseAttrs := make(map[string]interface{})

	responseAttrs["location"] = url
	responseAttrs["window"] = "self"
	responseAttrs["delay"] = 0

	setStateResponse := resource.NewActionResponse("client.store.set", map[string]interface{}{
		"key":   "secret",
		"value": state,
	})
	actionResponse := resource.NewActionResponse("client.redirect", responseAttrs)

	return nil, []actionresponse.ActionResponse{setStateResponse, actionResponse}, nil
}

func NewOauthLoginBeginActionPerformer(initConfig *resource.CmsConfig, cruds map[string]*resource.DbResource, configStore *resource.ConfigStore, transaction *sqlx.Tx) (actionresponse.ActionPerformerInterface, error) {

	secret, err := configStore.GetConfigValueFor("totp.secret", "backend", transaction)

	if err != nil {
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "site.daptin.com",
			AccountName: "dummy@site.daptin.com",
			Period:      300,
			SecretSize:  10,
		})

		if err != nil {
			log.Errorf("Failed to generate code: %v", err)
			return nil, err
		}
		configStore.SetConfigValueFor("totp.secret", key.Secret(), "backend", transaction)
		secret = key.Secret()
	}

	handler := oauthLoginBeginActionPerformer{
		cruds:       cruds,
		otpKey:      secret,
		configStore: configStore,
	}

	return &handler, nil

}
