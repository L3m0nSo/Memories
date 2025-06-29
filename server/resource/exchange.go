package resource

import (
	"github.com/L3m0nSo/Memories/server/auth"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	//"bytes"
	"bytes"
)

type ExchangeInterface interface {
	Update(target string, data []map[string]interface{}) error
}

type ExternalExchange interface {
	ExecuteTarget(row map[string]interface{}, transaction *sqlx.Tx) (map[string]interface{}, error)
}

type ColumnMap struct {
	SourceColumn     string
	SourceColumnType string
	TargetColumn     string
	TargetColumnType string
}

type ColumnMapping []ColumnMap

type ExchangeContract struct {
	Name             string
	SourceAttributes map[string]interface{} `db:"source_attributes"`
	Attributes       map[string]interface{} `db:"attributes"`
	SourceType       string                 `db:"source_type"`
	TargetAttributes map[string]interface{} `db:"target_attributes"`
	TargetType       string                 `db:"target_type"`
	User             auth.SessionUser
	Options          map[string]interface{}
	ReferenceId      string `db:"reference_id"`
	AsUserId         int64
}

var objectSuffix = []byte("{")
var arraySuffix = []byte("[")
var stringSuffix = []byte(`"`)

func (c *ColumnMapping) UnmarshalJSON(payload []byte) error {
	if bytes.HasPrefix(payload, objectSuffix) {
		return json.Unmarshal(payload, &c)
	}

	if bytes.HasPrefix(payload, arraySuffix) {
		return json.Unmarshal(payload, &c)
	}

	return errors.New("expected a JSON encoded object or array")
}

type ExchangeExecution struct {
	ExchangeContract ExchangeContract
	cruds            *map[string]*DbResource
}

func (exchangeExecution *ExchangeExecution) Execute(data []map[string]interface{}, transaction *sqlx.Tx) (result map[string]interface{}, err error) {

	var handler ExternalExchange

	switch exchangeExecution.ExchangeContract.TargetType {
	case "action":
		handler = NewActionExchangeHandler(exchangeExecution.ExchangeContract, *exchangeExecution.cruds)
		break
	case "rest":
		handler, err = NewRestExchangeHandler(exchangeExecution.ExchangeContract)
		if err != nil {
			return nil, err
		}
		break
	default:
		log.Errorf("exchange contract: target: 'self' is not yet implemented")
		return nil, errors.New("unknown target in exchange, not yet implemented")
	}

	//targetAttrs := exchangeExecution.ExchangeContract.TargetAttributes
	//
	//for k, v := range targetAttrs {
	//	inFields[k] = v
	//}

	for _, row := range data {
		result, err = handler.ExecuteTarget(row, transaction)
		if err != nil {
			log.Errorf("Failed to execute target for [%v]: %v", row["__type"], err)
		}
	}

	return result, err
}

func NewExchangeExecution(exchange ExchangeContract, cruds *map[string]*DbResource) *ExchangeExecution {

	return &ExchangeExecution{
		ExchangeContract: exchange,
		cruds:            cruds,
	}
}
