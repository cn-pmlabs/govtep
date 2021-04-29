package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/ebay/libovsdb"
)

// vrf attr list
const (
	VrfAttrL3vni  = "vrf_l3vni"
	VrfAttrTunnel = "vrf_tunnel"
)

// VrfObj ...
type VrfObj struct {
	Name string //"Vrf"+vni
}

func rowToVrfObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableVrf := vtepdb.ConvertRowToVrf(libovsdb.ResultRow(row.Fields))

	obj := VrfObj{
		Name: tableVrf.Name,
	}
	attrs := map[interface{}]interface{}{
		VrfAttrL3vni:  tableVrf.L3vni,
		VrfAttrTunnel: LocalPhsicalSwitchTunnelName,
	}

	return obj, attrs
}

func rowToVrfAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.VrfFieldL3vni:
			if l3Vni, ok := value.(string); ok {
				attrs[VrfAttrL3vni] = l3Vni
			}
		}
	}

	return attrs
}
