package fakerservice

import (
	"github.com/L3m0nSo/Memories/server/resource"
	"github.com/artpar/api2go/v2"
)

func NewFakeInstance(columns []api2go.ColumnInfo) map[string]interface{} {

	newObject := make(map[string]interface{})

	for _, col := range columns {
		if col.IsForeignKey {
			continue
		}

		if col.ColumnName == "id" {
			continue
		}

		fakeData := resource.ColumnManager.GetFakeData(col.ColumnType)

		newObject[col.ColumnName] = fakeData

	}

	return newObject

}
