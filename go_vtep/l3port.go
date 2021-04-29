package govtep

import (
	"errors"
	"fmt"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"

	"github.com/ebay/libovsdb"
)

// L3Port declaration
type L3Port struct {
	UUID        string
	LogicalPort string //对应的lrp
	PhySwitch   string //
	/*1、switchname or chassisname？待定*/
	/*2、option，只有逻辑port映射单个成员交换机端口时才会出现*/

	Name string
	//1、通常为patch口，如果patch peer为L2Port，则该L3Port为桥接接口，
	//如BDif20，如果patch peer为L3Port，则为内环接口，可以考虑配置为veth peers，涉及vrf互访的应用
	//2、如果为AC，即lrp与交换机端口直接映射

	PhyParentPort string //物理父port
	VlanTag       []int
	Ipv4addr      []string
	Ipv6addr      []string
	Mac           string
	Type          string //逻辑属性：“AC”、“patch”
	Vrf           string //绑定的Vrf实例name，通过datapath查到
	PeerType      string //“L2Port”or L3Port
	PeerPort      string //patch peer，L2Port or L3Port的name
	Neighour      []string
}

func l3portCreate(port PortInfo) error {
	if !vrfIsExist(port.Vrf) {
		return errors.New("l3port Create fail, because vrf not found")
	}

	l3portIndex := vtepdb.L3portIndex1{
		Name: port.Name,
	}
	_, err := vtepdb.L3portGetByIndex(l3portIndex)
	if err == nil {
		log.Info("l3port %s already exist", l3portIndex.Name)
		return nil
	}

	vrfIndex := vtepdb.VrfIndex{
		Name: port.Vrf,
	}
	tableL3port := vtepdb.TableL3port{
		Name:          port.Name,
		Vrf:           port.Vrf,
		Ipv4addr:      port.Ipv4addr,
		Ipv6addr:      port.Ipv6addr,
		Peerport:      port.PeerPort,
		Peertype:      port.PeerLtype,
		LogicalPort:   port.LogicalPort,
		PhyparentPort: port.PhyParentPort,
		Type:          port.Type,
	}
	if port.VlanTag != 0 {
		tableL3port.Vlantag = []int{port.VlanTag}
	}

	err = vtepdb.VrfUpdateAddL3ports(vrfIndex, tableL3port)

	return err
}

func l3portRemove(port PortInfo) error {
	vrfIndex := vtepdb.VrfIndex{
		Name: port.Vrf,
	}

	l3portIndex := vtepdb.L3portIndex1{
		Name: port.Name,
	}
	tableL3port, err := vtepdb.L3portGetByIndex(l3portIndex)
	if err != nil {
		return fmt.Errorf("l3port %s not exist", l3portIndex.Name)
	}

	// process neighbour remove first
	var rns []RemoteNeigh
	for _, neighUUID := range tableL3port.Neighbour {
		tableNeigh, err := vtepdb.RemoteNeighGetByUUID(neighUUID.GoUUID)
		if err != nil {
			log.Warning("neighbour %+v not exist", neighUUID)
			continue
		}

		neigh := RemoteNeigh{
			OutL3Port:     tableNeigh.OutL3port,
			Ipaddr:        tableNeigh.Ipaddr,
			Mac:           tableNeigh.Mac,
			RemoteLocator: tableNeigh.RemoteLocator,
		}
		rns = append(rns, neigh)
	}
	if len(rns) > 0 {
		remoteNeighRemove(rns)
	}

	// delete l3port in vrf
	err = vtepdb.VrfUpdateL3portsDelvalue(vrfIndex, []libovsdb.UUID{{GoUUID: tableL3port.UUID}})

	return err
}
