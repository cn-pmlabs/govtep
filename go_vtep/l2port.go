package govtep

import (
	"errors"
	"fmt"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"

	"github.com/ebay/libovsdb"
)

// L2Port declaration
type L2Port struct {
	UUID        string
	LogicalPort string //对应的lsp name
	PhySwitch   string //
	/*1、switchname or chassisname？待定*/
	/*2、option，只有逻辑port映射单个成员交换机端口时才会出现*/

	Name          string //接口名称，如eth1、eth1.1
	PhyParentPort string //物理父port
	VlanTag       []int  //
	Type          string //逻辑属性：“AC”、“patch”
	Bd            string //绑定的Bd实例name，通过datapath查到
	PeerType      string //“L2Port”or “L3Port”
	PeerPort      string //patch peer，L2Port or L3Port的name
}

func l2PortCreate(port PortInfo) error {
	if !bdIsExist(port.Bd) {
		return errors.New("l2port Create failed, because bd not found")
	}

	l2portIndex := vtepdb.L2portIndex1{
		Name: port.Name,
	}
	_, err := vtepdb.L2portGetByIndex(l2portIndex)
	if err == nil {
		log.Info("l2port %s already exist", l2portIndex.Name)
		return nil
	}

	bdIndex := vtepdb.BridgeDomainIndex{
		Name: port.Bd,
	}
	tableL2port := vtepdb.TableL2port{
		Name:          port.Name,
		Bd:            port.Bd,
		Peerport:      port.PeerPort,
		Peertype:      port.PeerLtype,
		LogicalPort:   port.LogicalPort,
		PhyparentPort: port.PhyParentPort,
		Type:          port.Type,
	}
	if port.VlanTag != 0 {
		tableL2port.Vlantag = []int{port.VlanTag}
	}
	err = vtepdb.BridgeDomainUpdateAddL2ports(bdIndex, tableL2port)

	return err
}

func l2PortRemove(port PortInfo) error {
	if !bdIsExist(port.Bd) {
		return errors.New("l2port delete failed, because bd not found")
	}

	bdIndex := vtepdb.BridgeDomainIndex{
		Name: port.Bd,
	}

	l2portIndex := vtepdb.L2portIndex1{
		Name: port.Name,
	}
	tableL2port, err := vtepdb.L2portGetByIndex(l2portIndex)
	if err != nil {
		return fmt.Errorf("l2port %s not exist", l2portIndex.Name)
	}

	err = vtepdb.BridgeDomainUpdateL2portsDelvalue(bdIndex, []libovsdb.UUID{{GoUUID: tableL2port.UUID}})

	return err
}
