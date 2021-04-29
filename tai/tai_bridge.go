package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/ebay/libovsdb"
)

// TAI bridge attr list
const (
	BridgeAttrL2vni       = "bridge_l2vni"
	BridgeAttrVxlanTunnel = "bridge_vxlan_tunnel"
)

// BridgeObj ...
type BridgeObj struct {
	Name string //"Bd"+str(vni)
	Vni  int
}

func rowToBridgeObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableBridgeDomain := vtepdb.ConvertRowToBridgeDomain(libovsdb.ResultRow(row.Fields))

	obj := BridgeObj{
		Name: tableBridgeDomain.Name,
		Vni:  tableBridgeDomain.L2vni,
	}
	attrs := map[interface{}]interface{}{
		BridgeAttrVxlanTunnel: LocalPhsicalSwitchTunnelName,
	}
	return obj, attrs
}

func rowToBridgeAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.BridgeDomainFieldL2vni:
			// TODO: for old value is nil ovsset
			if l2Vni, ok := value.(string); ok {
				attrs[BridgeAttrL2vni] = l2Vni
			}
		}
	}

	return attrs
}
