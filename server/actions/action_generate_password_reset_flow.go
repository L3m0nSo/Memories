package actions

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/L3m0nSo/Memories/server/actionresponse"
	daptinid "github.com/L3m0nSo/Memories/server/id"
	"github.com/L3m0nSo/Memories/server/resource"
	"github.com/artpar/api2go/v2"
	"github.com/artpar/go-guerrilla/backends"
	"github.com/artpar/go-guerrilla/mail"
	"github.com/doug-martin/goqu/v9"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

type generatePasswordResetActionPerformer struct {
	cruds                  map[string]*resource.DbResource
	secret                 []byte
	tokenLifeTime          int
	jwtTokenIssuer         string
	passwordResetEmailFrom string
}

func (d *generatePasswordResetActionPerformer) Name() string {
	return "password.reset.begin"
}

func (d *generatePasswordResetActionPerformer) DoAction(request actionresponse.Outcome, inFieldMap map[string]interface{}, transaction *sqlx.Tx) (api2go.Responder, []actionresponse.ActionResponse, []error) {

	responses := make([]actionresponse.ActionResponse, 0)

	email := inFieldMap["email"]

	existingUsers, _, err := d.cruds[resource.USER_ACCOUNT_TABLE_NAME].GetRowsByWhereClause("user_account", nil, transaction, goqu.Ex{"email": email})

	responseAttrs := make(map[string]interface{})
	if err != nil || len(existingUsers) < 1 {
		responseAttrs["type"] = "error"
		responseAttrs["message"] = "No Such account"
		responseAttrs["title"] = "Failed"
		actionResponse := resource.NewActionResponse("client.notify", responseAttrs)
		responses = append(responses, actionResponse)
	} else {
		existingUser := existingUsers[0]

		// Create a new token object, specifying signing method and the claims
		// you would like it to contain.
		u, _ := uuid.NewV7()
		email := existingUser["email"].(string)
		timeNow := time.Now().UTC()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"email": email,
			"name":  existingUser["name"],
			"nbf":   timeNow.Unix(),
			"exp":   timeNow.Add(30 * time.Minute).Unix(),
			"sub":   daptinid.InterfaceToDIR(existingUser["reference_id"]).String(),
			"iss":   d.jwtTokenIssuer,
			"iat":   timeNow.Unix(),
			"jti":   u.String(),
		})

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := token.SignedString(d.secret)
		tokenStringBase64 := base64.StdEncoding.EncodeToString([]byte(tokenString))
		fmt.Printf("%v %v", tokenStringBase64, err)
		if err != nil {
			log.Errorf("Failed to sign string: %v", err)
			return nil, nil, []error{err}
		}

		mailBody := "Reset your password by clicking on this link: " + tokenStringBase64

		bodyDaya := bytes.NewBuffer([]byte(mailBody))
		mailEnvelop := mail.Envelope{
			Subject: "Reset password for account " + email,
			RcptTo: []mail.Address{
				{
					User: strings.Split(email, "@")[0],
					Host: strings.Split(email, "@")[1],
				},
			},
			MailFrom: mail.Address{
				User: strings.Split(d.passwordResetEmailFrom, "@")[0],
				Host: strings.Split(d.passwordResetEmailFrom, "@")[1],
			},
			Header: textproto.MIMEHeader{
				"Date": []string{timeNow.String()},
			},
			Data: *bodyDaya,
		}

		mailResult, err := d.cruds["mail"].MailSender(&mailEnvelop, backends.TaskSaveMail)
		if mailResult != nil {
			log.Printf("Password reset mail result:  %s", mailResult.String())
			notificationAttrs := make(map[string]string)
			notificationAttrs["message"] = "Password reset mail sent"
			notificationAttrs["title"] = "Success"
			notificationAttrs["type"] = "success"
			responses = append(responses, resource.NewActionResponse("client.notify", notificationAttrs))
		} else {
			log.Errorf("Failed to sent password reset email %s", err)
			notificationAttrs := make(map[string]string)
			notificationAttrs["message"] = "Failed to send password reset mail"
			notificationAttrs["title"] = "Failed"
			notificationAttrs["type"] = "failed"
			responses = append(responses, resource.NewActionResponse("client.notify", notificationAttrs))
		}

	}

	return nil, responses, nil
}

func NewGeneratePasswordResetActionPerformer(configStore *resource.ConfigStore, cruds map[string]*resource.DbResource) (actionresponse.ActionPerformerInterface, error) {
	transaction, err := cruds["world"].Connection().Beginx()
	if err != nil {
		resource.CheckErr(err, "Failed to begin transaction [120]")
		return nil, err
	}

	defer transaction.Commit()
	secret, _ := configStore.GetConfigValueFor("jwt.secret", "backend", transaction)

	tokenLifeTimeHours, err := configStore.GetConfigIntValueFor("jwt.token.life.hours", "backend", transaction)
	resource.CheckErr(err, "No default jwt token life time set in configuration")
	if err != nil {
		err = configStore.SetConfigIntValueFor("jwt.token.life.hours", 24*3, "backend", transaction)
		resource.CheckErr(err, "Failed to store default jwt token life time")
		tokenLifeTimeHours = 24 * 3 // 3 days
	}

	jwtTokenIssuer, err := configStore.GetConfigValueFor("jwt.token.issuer", "backend", transaction)
	resource.CheckErr(err, "No default jwt token issuer set")
	if err != nil {
		uid, _ := uuid.NewV7()
		jwtTokenIssuer = "daptin-" + uid.String()[0:6]
		err = configStore.SetConfigValueFor("jwt.token.issuer", jwtTokenIssuer, "backend", transaction)
	}

	passwordResetEmailFrom, err := configStore.GetConfigValueFor("password.reset.email.from", "backend", transaction)
	resource.CheckErr(err, "No default password reset email from set")
	if err != nil {
		hostname, err := configStore.GetConfigValueFor("hostname", "backend", transaction)
		if err != nil {
			hostname, err = os.Hostname()
		}
		jwtTokenIssuer = "no-reply@" + hostname
		err = configStore.SetConfigValueFor("password.reset.email.from", hostname, "backend", transaction)
	}

	handler := generatePasswordResetActionPerformer{
		cruds:                  cruds,
		secret:                 []byte(secret),
		tokenLifeTime:          tokenLifeTimeHours,
		passwordResetEmailFrom: passwordResetEmailFrom,

		jwtTokenIssuer: jwtTokenIssuer,
	}

	return &handler, nil

}
