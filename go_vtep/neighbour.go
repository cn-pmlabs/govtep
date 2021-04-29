package govtep

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"

	"github.com/ebay/libovsdb"
)

// RemoteNeigh ...
type RemoteNeigh struct {
	UUID          string
	OutL3Port     string
	Ipaddr        string
	Mac           string
	RemoteLocator string //VTEP DB Locator uuid
}

// LocalNeigh ...
type LocalNeigh struct {
	UUID      string
	OutL3Port string
	Ipaddr    string
	Mac       string
}

func macbindingNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate) {
	// todo

}

func remoteNeighCreate(rns []RemoteNeigh) error {
	for _, rn := range rns {
		neighIndex := vtepdb.RemoteNeighIndex{
			Ipaddr: rn.Ipaddr,
			Mac:    rn.Mac,
		}
		_, err := vtepdb.RemoteNeighGetByIndex(neighIndex)
		if err == nil {
			log.Info("neighbour %+v already exist", neighIndex)
			continue
		}

		tableNeigh := vtepdb.TableRemoteNeigh{
			Mac:           rn.Mac,
			Ipaddr:        rn.Ipaddr,
			OutL3port:     rn.OutL3Port,
			RemoteLocator: rn.RemoteLocator,
		}

		l3portIndex := vtepdb.L3portIndex1{
			Name: rn.OutL3Port,
		}
		tableL3port, err := vtepdb.L3portGetByIndex(l3portIndex)
		if err != nil {
			log.Info("Neighbour %s outL3port not exist", tableNeigh.Ipaddr)
			continue
		}

		err = vtepdb.L3portUpdateAddNeighbour(l3portIndex, tableNeigh)
		if err != nil {
			continue
		}

		// in sysmetry forwarding, the neighbour should add static host vxlan route
		vrfIndex := vtepdb.VrfIndex{
			Name: tableL3port.Vrf,
		}
		tableVrf, err := vtepdb.VrfGetByIndex(vrfIndex)
		if err != nil {
			log.Warning("Neighbour %s vrf %s not exist", tableNeigh.Ipaddr, vrfIndex.Name)
			continue
		}
		tableLocator, err := vtepdb.LocatorGetByUUID(rn.RemoteLocator)
		if err != nil {
			log.Warning("Locator for neighbour %s not exist yet", tableNeigh.Ipaddr)
			continue
		}

		rt := Route{
			IPPrefix:      rn.Ipaddr + "/32",
			Vrf:           tableVrf.Name,
			NhVrf:         tableVrf.Name,
			OutputPort:    tableL3port.Name,
			Policy:        RoutePolicyDefault,
			RemoteLocator: rn.RemoteLocator,
		}
		// locator first ip as route nexthop for now
		if len(tableLocator.Ipaddr) > 0 {
			rt.Nexthop = tableLocator.Ipaddr[0]
		}

		err = routeCreate(rt)
		if err != nil {
			log.Warning("Add host vxlan route for neighbour %s failed", tableNeigh.Ipaddr)
			continue
		}
	}
	return nil
}

func remoteNeighRemove(rns []RemoteNeigh) error {
	for _, rn := range rns {
		neighIndex := vtepdb.RemoteNeighIndex{
			Ipaddr: rn.Ipaddr,
			Mac:    rn.Mac,
		}
		tableNeigh, err := vtepdb.RemoteNeighGetByIndex(neighIndex)
		if err != nil {
			log.Warning("neighbour %+v not existed", neighIndex)
			continue
		}

		l3portIndex := vtepdb.L3portIndex1{
			Name: rn.OutL3Port,
		}
		tableL3port, err := vtepdb.L3portGetByIndex(l3portIndex)
		if err != nil {
			log.Warning("Neighbour %s outL3port not exist", neighIndex.Ipaddr)
			continue
		}

		err = vtepdb.L3portUpdateNeighbourDelvalue(l3portIndex, []libovsdb.UUID{{GoUUID: tableNeigh.UUID}})
		if err != nil {
			continue
		}

		// remove related vxlan static host route
		rt := Route{
			IPPrefix: rn.Ipaddr + "/32",
			Vrf:      tableL3port.Vrf,
		}

		err = routeRemove(rt)
		if err != nil {
			log.Warning("Remove host vxlan route for neighbour %s failed", tableNeigh.Ipaddr)
			continue
		}
	}

	return nil
}
