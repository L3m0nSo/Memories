package actions

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/L3m0nSo/Memories/server/actionresponse"
	fieldtypes "github.com/L3m0nSo/Memories/server/columntypes"
	"github.com/L3m0nSo/Memories/server/resource"
	"github.com/L3m0nSo/Memories/server/rootpojo"
	"github.com/L3m0nSo/Memories/server/table_info"
	"github.com/artpar/api2go/v2"
	"github.com/artpar/xlsx/v2"
	"github.com/jmoiron/sqlx"
	"github.com/sadlil/go-trigger"
	log "github.com/sirupsen/logrus"
)

type uploadXlsFileToEntityPerformer struct {
	responseAttrs map[string]interface{}
	cruds         map[string]*resource.DbResource
	cmsConfig     *resource.CmsConfig
}

func (d *uploadXlsFileToEntityPerformer) Name() string {
	return "__upload_xlsx_file_to_entity"
}

var entityTypeToDataTypeMap = map[fieldtypes.EntityType]string{
	fieldtypes.DateTime:    "datetime",
	fieldtypes.Id:          "varchar(100)",
	fieldtypes.Time:        "time",
	fieldtypes.Date:        "date",
	fieldtypes.Ipaddress:   "varchar(100)",
	fieldtypes.Money:       "float(11)",
	fieldtypes.Rating5:     "int(4)",
	fieldtypes.Rating10:    "int(4)",
	fieldtypes.Rating100:   "int(4)",
	fieldtypes.Timestamp:   "timestamp",
	fieldtypes.NumberInt:   "int(5)",
	fieldtypes.NumberFloat: "float(11)",
	fieldtypes.Boolean:     "bool",
	fieldtypes.Latitude:    "float(11)",
	fieldtypes.Longitude:   "float(11)",
	fieldtypes.City:        "varchar(100)",
	fieldtypes.Country:     "varchar(100)",
	fieldtypes.Continent:   "varchar(100)",
	fieldtypes.State:       "varchar(100)",
	fieldtypes.Pincode:     "varchar(20)",
	fieldtypes.None:        "varchar(100)",
	fieldtypes.Label:       "varchar(100)",
	fieldtypes.Name:        "varchar(100)",
	fieldtypes.Email:       "varchar(100)",
	fieldtypes.Content:     "text",
	fieldtypes.Json:        "text",
	fieldtypes.Color:       "varchar(10)",
	fieldtypes.Alias:       "varchar(100)",
	fieldtypes.Namespace:   "varchar(100)",
}

var EntityTypeToColumnTypeMap = map[fieldtypes.EntityType]string{
	fieldtypes.DateTime:    "datetime",
	fieldtypes.Id:          "label",
	fieldtypes.Time:        "time",
	fieldtypes.Date:        "date",
	fieldtypes.Ipaddress:   "label",
	fieldtypes.Money:       "measurement",
	fieldtypes.Rating5:     "measurement",
	fieldtypes.Rating10:    "measurement",
	fieldtypes.Rating100:   "measurement",
	fieldtypes.Timestamp:   "timestamp",
	fieldtypes.NumberInt:   "measurement",
	fieldtypes.NumberFloat: "measurement",
	fieldtypes.Boolean:     "truefalse",
	fieldtypes.Latitude:    "location.latitude",
	fieldtypes.Longitude:   "location.longitude",
	fieldtypes.City:        "label",
	fieldtypes.Country:     "label",
	fieldtypes.Continent:   "label",
	fieldtypes.State:       "label",
	fieldtypes.Pincode:     "label",
	fieldtypes.None:        "content",
	fieldtypes.Label:       "label",
	fieldtypes.Name:        "name",
	fieldtypes.Email:       "email",
	fieldtypes.Content:     "content",
	fieldtypes.Json:        "json",
	fieldtypes.Color:       "color",
	fieldtypes.Alias:       "alias",
	fieldtypes.Namespace:   "namespace",
}

func (d *uploadXlsFileToEntityPerformer) DoAction(request actionresponse.Outcome, inFields map[string]interface{}, transaction *sqlx.Tx) (api2go.Responder, []actionresponse.ActionResponse, []error) {

	//actions := make([]actionresponse.ActionResponse, 0)
	log.Printf("Do action: %v", d.Name())
	schemaFolderDefinedByEnv, _ := os.LookupEnv("DAPTIN_SCHEMA_FOLDER")

	files := inFields["data_xls_file"].([]interface{})

	entityName := inFields["entity_name"].(string)
	create_if_not_exists, _ := inFields["create_if_not_exists"].(bool)
	add_missing_columns, _ := inFields["add_missing_columns"].(bool)

	table := table_info.TableInfo{}
	table.TableName = entityName

	columns := make([]api2go.ColumnInfo, 0)

	allSt := make(map[string]interface{})

	sources := make([]rootpojo.DataFileImport, 0)

	completed := false

	var existingEntity *table_info.TableInfo
	if !create_if_not_exists {
		var ok bool
		dbr, ok := d.cruds[entityName]
		if !ok {
			return nil, nil, []error{fmt.Errorf("no such entity: %v", entityName)}
		}
		existingEntity = dbr.TableInfo()
	}

nextFile:
	for _, fileInterface := range files {
		file := fileInterface.(map[string]interface{})
		fileName := "_uploaded_" + file["name"].(string)
		fileContentsBase64 := file["file"].(string)
		fileBytes, err := base64.StdEncoding.DecodeString(strings.Split(fileContentsBase64, ",")[1])
		log.Printf("Processing file: %v", fileName)

		xlsFile, err := xlsx.OpenBinary(fileBytes)
		resource.CheckErr(err, "Uploaded file is not a valid xls file")
		if err != nil {
			return nil, nil, []error{fmt.Errorf("Failed to read file: %v", err)}
		}
		log.Printf("File has %d sheets", len(xlsFile.Sheets))
		err = os.WriteFile(schemaFolderDefinedByEnv+string(os.PathSeparator)+fileName, fileBytes, 0644)
		if err != nil {
			log.Errorf("Failed to write xls file to disk: %v", err)
		}

		for _, sheet := range xlsFile.Sheets {

			data, columnNames, err := GetDataArray(sheet)
			recordCount := len(data)

			if err != nil {
				log.Errorf("Failed to get data from sheet [%s]: %v", sheet.Name, err)
				return nil, nil, []error{fmt.Errorf("Failed to get data from sheet [%s]: %v", sheet.Name, err)}
			}

			// identify data type of each column
			for _, colName := range columnNames {

				if colName == "" {
					continue
				}

				var column api2go.ColumnInfo

				if add_missing_columns && existingEntity != nil {
					_, ok := existingEntity.GetColumnByName(colName)
					if !ok {
						// ignore column if it doesn't exists
						continue
					}
				}

				dataMap := map[string]bool{}
				datas := make([]string, 0)

				isNullable := false
				count := 100000
				maxLen := 100
				for _, d := range data {
					if count < 0 {
						break
					}
					i := d[colName]
					var strVal string
					if i == nil {
						strVal = ""
						isNullable = true
						continue
					} else {
						strVal = i.(string)
					}
					if dataMap[strVal] {
						continue
					}
					dataMap[strVal] = true
					datas = append(datas, strVal)
					if maxLen < len(strVal) {
						maxLen = len(strVal)
					}
					count -= 1
				}

				eType, _, err := fieldtypes.DetectType(datas)
				if err != nil {
					log.Printf("Unable to identify column type for %v", colName)
					column.ColumnType = "label"
					column.DataType = fmt.Sprintf("varchar(%v)", maxLen)
				} else {
					log.Printf("Column %v was identified as %v", colName, eType)
					column.ColumnType = EntityTypeToColumnTypeMap[eType]

					dbDataType := entityTypeToDataTypeMap[eType]
					if strings.Index(dbDataType, "varchar") == 0 {
						dbDataType = fmt.Sprintf("varchar(%v)", maxLen+100)
					}
					column.DataType = dbDataType

				}

				if len(datas) > (recordCount / 10) {
					column.IsIndexed = true
				}

				if len(datas) == recordCount {
					column.IsUnique = true
				}

				column.IsNullable = isNullable
				column.Name = colName
				column.ColumnName = SmallSnakeCaseText(colName)

				columns = append(columns, column)
			}

			table.Columns = columns
			completed = true
			sources = append(sources, rootpojo.DataFileImport{FilePath: fileName, Entity: table.TableName, FileType: "xlsx"})

			break nextFile
		}
	}

	if completed {

		if create_if_not_exists {
			allSt["tables"] = []table_info.TableInfo{table}
		}

		allSt["imports"] = sources

		jsonStr, err := json.Marshal(allSt)
		if err != nil {
			resource.InfoErr(err, "Failed to convert object to json")
			return nil, nil, []error{err}
		}

		jsonFileName := fmt.Sprintf("schema_uploaded_%v_daptin.json", entityName)
		err = os.WriteFile(schemaFolderDefinedByEnv+string(os.PathSeparator)+jsonFileName, jsonStr, 0644)
		resource.CheckErr(err, "Failed to write json to schema file [%v]", jsonFileName)
		log.Printf("File %v written to disk for upload", jsonFileName)

		if create_if_not_exists || add_missing_columns {
			//go Restart()
		} else {
			resource.ImportDataFiles(sources, transaction, d.cruds)
		}

		trigger.Fire("clean_up_uploaded_files")

		return nil, successResponses, nil
	} else {
		return nil, failedResponses, nil
	}

}

var successResponses = []actionresponse.ActionResponse{
	resource.NewActionResponse("client.notify", map[string]interface{}{
		"type":    "success",
		"message": "Initiating system update.",
		"title":   "Success",
	}),
	resource.NewActionResponse("client.redirect", map[string]interface{}{
		"location": "/",
		"window":   "self",
		"delay":    15000,
	}),
}

var failedResponses = []actionresponse.ActionResponse{
	resource.NewActionResponse("client.notify", map[string]interface{}{
		"type":    "error",
		"message": "Failed to import xls",
		"title":   "Failed",
	}),
}

func NewUploadFileToEntityPerformer(initConfig *resource.CmsConfig, cruds map[string]*resource.DbResource) (actionresponse.ActionPerformerInterface, error) {

	handler := uploadXlsFileToEntityPerformer{
		cruds:     cruds,
		cmsConfig: initConfig,
	}

	return &handler, nil

}
