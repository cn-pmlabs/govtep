package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/ebay/libovsdb"
)

// mcastfdb attr list
const (
	McastFdbAttrLocators = "mcastfdb_locators"
)

// McastFdbObj ...
type McastFdbObj struct {
	BridgeName     string
	IsolationGroup string
	Mac            string
}

func rowToMcastfdbObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableMcastFdb := vtepdb.ConvertRowToMcastMacsRemote(libovsdb.ResultRow(row.Fields))

	obj := McastFdbObj{
		BridgeName: tableMcastFdb.Bridge,
		Mac:        tableMcastFdb.Mac,
		//IsolationGroup: ,
	}
	attrs := map[interface{}]interface{}{
		McastFdbAttrLocators: tableMcastFdb.Locators,
	}
	return obj, attrs
}

func rowToMcastfdbAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.McastMacsRemoteFieldLocators:
			if locator, ok := value.(string); ok {
				attrs[McastFdbAttrLocators] = locator
			}
		}
	}

	return attrs
}

func taiUpdateObjMcastFdb(objID ObjID, newrow libovsdb.Row, oldrow libovsdb.Row) {

}
