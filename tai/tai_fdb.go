package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"

	"github.com/ebay/libovsdb"
)

// FDB attr list
const (
	FdbAttrLocator    = "fdb_locator"
	FdbAttrTunnelName = "fdb_tunnel_name"
	FdbAttrRemoteIP   = "fdb_remote_ip"
	FdbAttrPort       = "fdb_port"
)

// FdbObj ...
type FdbObj struct {
	Bridge string //Bridge uuid
	Mac    string
}

func rowToFdbObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableFdb := vtepdb.ConvertRowToRemoteFdb(libovsdb.ResultRow(row.Fields))

	obj := FdbObj{
		Bridge: tableFdb.Bridge,
		Mac:    tableFdb.Mac,
	}
	attrs := make(map[interface{}]interface{})

	tableLocator, err := vtepdb.LocatorGetByUUID(tableFdb.RemoteLocator)
	if err != nil {
		log.Warning("[TAI] fdb %+v remote ip get failed", obj)
		return obj, attrs
	}

	attrs[FdbAttrRemoteIP] = tableLocator.Ipaddr[0]
	attrs[FdbAttrTunnelName] = LocalPhsicalSwitchTunnelName

	return obj, attrs
}

func rowToFdbAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.RemoteFdbFieldRemoteLocator:
			if locator, ok := value.(string); ok {
				attrs[FdbAttrLocator] = locator
			}
		}
	}

	return attrs
}
