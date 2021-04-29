package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// route attr list
const (
	PBRAttrNexthopGroup = "pbr_nexthop_group"
)

// PBRObj ...
type PBRObj struct {
	Type     string
	Vrf      string
	IP       string
	Port     int
	Protocol string
}

func rowToPBRObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tablePBR := vtepdb.ConvertRowToPolicyBasedRoute(libovsdb.ResultRow(row.Fields))

	if len(tablePBR.Port) != 1 || len(tablePBR.Protocol) != 1 {
		log.Warning("PBR %+v port or protocol invalid\n", tablePBR)
		return nil, nil
	}

	obj := PBRObj{
		Type:     tablePBR.Type,
		Vrf:      tablePBR.Vrf,
		IP:       tablePBR.IP,
		Port:     tablePBR.Port[0],
		Protocol: tablePBR.Protocol[0],
	}
	attrs := map[interface{}]interface{}{
		PBRAttrNexthopGroup: tablePBR.NhGroup,
	}

	return obj, attrs
}

func rowToPBRAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.PolicyBasedRouteFieldNhGroup:
			switch value.(type) {
			case string:
				attrs[PBRAttrNexthopGroup] = []string{value.(string)}
			case libovsdb.OvsSet:
				attrs[PBRAttrNexthopGroup] = odbc.
					ConvertGoSetToStringArray(value.(libovsdb.OvsSet))
			}
		}
	}

	return attrs
}
