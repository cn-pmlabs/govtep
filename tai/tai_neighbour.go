package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"

	"github.com/ebay/libovsdb"
)

// neighbour attr list
const (
	NeighbourAttrMacaddr  = "neighbour_macaddr"
	NeighbourAttrBridge   = "neighbour_bridge"
	NeighbourAttrOutPort  = "neighbour_outport"
	NeighbourAttrRemoteIP = "neighbour_remoteip"
	NeighbourAttrLocator  = "neighbour_locator"
)

// NeighbourObj ...
type NeighbourObj struct {
	Ipaddr string
}

func rowToNeighbourObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableNeighbour := vtepdb.ConvertRowToRemoteNeigh(libovsdb.ResultRow(row.Fields))

	obj := NeighbourObj{
		Ipaddr: tableNeighbour.Ipaddr,
	}

	attrs := map[interface{}]interface{}{
		NeighbourAttrMacaddr: tableNeighbour.Mac,
		NeighbourAttrOutPort: tableNeighbour.OutL3port,
	}

	tableLocator, err := vtepdb.LocatorGetByUUID(tableNeighbour.RemoteLocator)
	if err != nil {
		log.Warning("[TAI] Locator for neighbour %s not exist yet", tableNeighbour.Ipaddr)
	} else {
		attrs[NeighbourAttrRemoteIP] = tableLocator.Ipaddr[0]
	}

	return obj, attrs
}

func rowToNeighbourAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.RemoteNeighFieldMac:
			if mac, ok := value.(string); ok {
				attrs[NeighbourAttrMacaddr] = mac
			}
		case vtepdb.RemoteNeighFieldRemoteLocator:
			if locator, ok := value.(string); ok {
				attrs[NeighbourAttrLocator] = locator
			}
		}
	}

	return attrs
}
