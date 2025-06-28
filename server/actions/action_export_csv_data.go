package actions

import (
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/L3m0nSo/Memories/server/actionresponse"
	"github.com/L3m0nSo/Memories/server/resource"
	"github.com/artpar/api2go/v2"
	"github.com/gocarina/gocsv"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

type exportCsvDataPerformer struct {
	cmsConfig *resource.CmsConfig
	cruds     map[string]*resource.DbResource
}

func (d *exportCsvDataPerformer) Name() string {
	return "__csv_data_export"
}

func (d *exportCsvDataPerformer) DoAction(request actionresponse.Outcome, inFields map[string]interface{}, transaction *sqlx.Tx) (api2go.Responder, []actionresponse.ActionResponse, []error) {

	responses := make([]actionresponse.ActionResponse, 0)

	tableName, ok := inFields["table_name"]

	finalName := "complete"

	result := make(map[string]interface{})

	if ok && tableName != nil {

		tableNameStr := tableName.(string)
		log.Printf("Export data for table: %v", tableNameStr)

		objects, err := d.cruds[tableNameStr].GetAllRawObjectsWithTransaction(tableNameStr, transaction)
		if err != nil {
			log.Errorf("Failed to get all objects of type [%v] : %v", tableNameStr, err)
		}

		result[tableNameStr] = objects
		finalName = tableNameStr
	} else {

		for _, tableInfo := range d.cmsConfig.Tables {
			data, err := d.cruds[tableInfo.TableName].GetAllRawObjectsWithTransaction(tableInfo.TableName, transaction)
			if err != nil {
				log.Errorf("Failed to export objects of type [%v]: %v", tableInfo.TableName, err)
				continue
			}
			result[tableInfo.TableName] = data
		}

	}

	currentDate := time.Now()
	prefix := currentDate.Format("2006-01-02-15-04-05")
	csvFile, err := os.CreateTemp("", prefix)
	defer csvFile.Close()

	for outTableName, contents := range result {

		if tableName != nil {
			csvFile.WriteString(outTableName)
		}
		csvFileWriter := csv.NewWriter(csvFile)
		contentArray := contents.([]map[string]interface{})

		if len(contentArray) == 0 {
			csvFile.WriteString("No data\n")
		}

		var columnKeys []string
		csvWriter := gocsv.NewSafeCSVWriter(csvFileWriter)
		firstRow := contentArray[0]

		for colName := range firstRow {
			columnKeys = append(columnKeys, colName)
		}

		csvWriter.Write(columnKeys)

		for _, row := range contentArray {
			var dataRow []string
			for _, colName := range columnKeys {
				dataRow = append(dataRow, fmt.Sprintf("%v", row[colName]))
			}
			csvWriter.Write(dataRow)
		}
		csvFile.WriteString("\n")
	}

	csvFileName := csvFile.Name()
	csvFileContents, err := os.ReadFile(csvFileName)
	if resource.InfoErr(err, "Failed to read csv file to download") {
		actionResponse := resource.NewActionResponse("client.notify", resource.NewClientNotification("error", "Failed to generate csv: "+err.Error(), "Failed"))
		responses = append(responses, actionResponse)
		return nil, responses, nil
	}

	responseAttrs := make(map[string]interface{})
	responseAttrs["content"] = base64.StdEncoding.EncodeToString(csvFileContents)
	responseAttrs["name"] = fmt.Sprintf("daptin_dump_%v.csv", finalName)
	responseAttrs["contentType"] = "application/csv"
	responseAttrs["message"] = "Downloading csv "

	actionResponse := resource.NewActionResponse("client.file.download", responseAttrs)

	responses = append(responses, actionResponse)

	return nil, responses, nil
}

func NewExportCsvDataPerformer(initConfig *resource.CmsConfig, cruds map[string]*resource.DbResource) (actionresponse.ActionPerformerInterface, error) {

	handler := exportCsvDataPerformer{
		cmsConfig: initConfig,
		cruds:     cruds,
	}

	return &handler, nil

}
