package fakerservice

import (
	"testing"

	"github.com/L3m0nSo/Memories/server/resource"
	"github.com/L3m0nSo/Memories/server/table_info"
	"github.com/artpar/api2go/v2"
	log "github.com/sirupsen/logrus"
)

func TestNewFakeInstance(t *testing.T) {

	resource.InitialiseColumnManager()
	table := &table_info.TableInfo{
		TableName: "test",
		Columns:   []api2go.ColumnInfo{},
	}

	for _, ty := range resource.ColumnTypes {
		table.Columns = append(table.Columns, api2go.ColumnInfo{
			ColumnName: ty.Name,
			ColumnType: ty.Name,
		})
	}

	fi := NewFakeInstance(table.Columns)
	for _, ty := range resource.ColumnTypes {
		if ty.Name == "id" {
			continue
		}
		if fi[ty.Name] == nil {
			t.Errorf("No fake value generated for %v", ty.Name)
		}
		log.Printf(" [%v] value : %v", ty.Name, fi[ty.Name])
	}

}
