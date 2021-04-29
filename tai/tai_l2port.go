package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/ebay/libovsdb"
)

// L2portObj attr list
const (
	L2portAttrVlanTag = "l2port_vlantag"
)

// L2portObj ...
type L2portObj struct {
	Name               string
	BridgeName         string
	PhysicalParentPort string
}

func rowToL2portObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableL2port := vtepdb.ConvertRowToL2port(libovsdb.ResultRow(row.Fields))

	obj := L2portObj{
		Name:               tableL2port.Name,
		BridgeName:         tableL2port.Bd,
		PhysicalParentPort: tableL2port.PhyparentPort,
	}
	attrs := map[interface{}]interface{}{
		L2portAttrVlanTag: tableL2port.Vlantag,
	}
	return obj, attrs
}

func rowToL2portAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.L2portFieldVlantag:
			if vlanTag, ok := value.(int); ok {
				attrs[L2portAttrVlanTag] = vlanTag
			}
		}
	}

	return attrs
}
