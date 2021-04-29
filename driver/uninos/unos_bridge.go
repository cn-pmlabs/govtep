package driver

import (
	"fmt"
	"strconv"

	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"

	"github.com/ebay/libovsdb"
)

func getVniByBdName(bdName string) int {
	vniStr := bdName[2:]
	vni, err := strconv.Atoi(vniStr)
	if err != nil {
		return 0
	}
	return vni
}

type bridgeAPI struct {
	moduleID int
}

var bdAPIs = bridgeAPI{
	moduleID: tai.ObjectIDBridge,
}

func (d bridgeAPI) CreateObject(obj interface{}) error {
	objBridge := obj.(tai.BridgeObj)

	vni := getVniByBdName(objBridge.Name)
	if vni < cdb.BridgeVniMin || vni > cdb.BridgeVniMax {
		return fmt.Errorf("[Driver] Invalid bridge name %s", objBridge.Name)
	}

	bridgeCfg := cdb.TableBridge{
		Name:           objBridge.Name,
		Vni:            []int{vni},
		MacLimit:       []int{cdb.BridgeDefaultMacLimit},
		MacAlarm:       []string{cdb.BridgeDefaultMacAlarm},
		MacLearn:       []string{cdb.BridgeDefaultMacLearn},
		McFlood:        []string{cdb.BridgeDefaultMcFlood},
		BcFlood:        []string{cdb.BridgeDefaultBcFlood},
		UnknownUcFlood: []string{cdb.BridgeDefaultUnknownUcFlood},
	}

	bridgeIndex := cdb.BridgeIndex{
		Name: objBridge.Name,
	}
	_, err := cdb.BridgeGetByIndex(bridgeIndex)
	if err == nil {
		log.Info("[Driver] bridge %s already exist\n", objBridge.Name)
		goto interfaceADD
	}

	_, err = cdb.BridgeAdd(bridgeCfg)
	if err != nil {
		return err
	}

interfaceADD:
	interfaceCfg := cdb.TableInterface{
		Name:        bridgeCfg.Name,
		Type:        cdb.InterfaceTypeBridgeDomain,
		AdminStatus: []string{cdb.InterfaceDefaultAdminStatus},
		Mtu:         []int{cdb.InterfaceDefaultMtu},
		ProxyArp:    []string{cdb.InterfaceDefaultProxyArp},
		SwitchPort:  []string{cdb.InterfaceDefaultSwitchPort},
	}

	ifIndex := cdb.InterfaceIndex{
		Name: bridgeCfg.Name,
		Type: cdb.InterfaceTypeBridgeDomain,
	}
	_, err = cdb.InterfaceGetByIndex(ifIndex)
	if err == nil {
		log.Info("[Driver] interface %s type %s already exist\n", ifIndex.Name, ifIndex.Type)
		return nil
	}

	_, err = cdb.InterfaceAdd(interfaceCfg)
	if err != nil {
		return err
	}

	return nil
}

func (d bridgeAPI) RemoveObject(obj interface{}) error {
	objBridge := obj.(tai.BridgeObj)

	bridgeIndex := cdb.BridgeIndex{
		Name: objBridge.Name,
	}
	err := cdb.BridgeDelByIndex(bridgeIndex)
	if err != nil {
		return err
	}

	interfaceIndex := cdb.InterfaceIndex{
		Name: bridgeIndex.Name,
		Type: cdb.InterfaceTypeBridgeDomain,
	}
	err = cdb.InterfaceDelByIndex(interfaceIndex)
	if err != nil {
		return err
	}

	return nil
}

func (d bridgeAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objBridge := obj.(tai.BridgeObj)

	bridgeIndex := cdb.BridgeIndex{
		Name: objBridge.Name,
	}
	_, err := cdb.BridgeGetByIndex(bridgeIndex)
	if err != nil {
		log.Warning("[Driver] BD %s not exist\n", bridgeIndex.Name)
		return err
	}

	if attrs[tai.BridgeAttrVxlanTunnel] != nil {
		tunnelName := attrs[tai.BridgeAttrVxlanTunnel].(string)
		tunnelIndex := cdb.TunnelIndex{
			Name: tunnelName,
		}
		tableTunnel, err := cdb.TunnelGetByIndex(tunnelIndex)
		if err != nil {
			log.Warning("[Driver] BD %s tunnel %s not exist\n", bridgeIndex.Name, tunnelName)
			return nil
		}

		cdb.BridgeUpdateVxlanTunnelAddvalue(bridgeIndex, []libovsdb.UUID{{GoUUID: tableTunnel.UUID}})
	}

	return nil
}

func (d bridgeAPI) DelObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (d bridgeAPI) SetObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (d bridgeAPI) GetObjectAttr(obj interface{}, attrIDs []interface{}) (map[interface{}]interface{}, error) {
	attrs := make(map[interface{}]interface{})
	objBridge := obj.(tai.BridgeObj)

	bridgeIndex := cdb.BridgeIndex{
		Name: objBridge.Name,
	}
	tableBridge, err := cdb.BridgeGetByIndex(bridgeIndex)
	if err != nil {
		return nil, err
	}

	for _, attr := range attrIDs {
		switch attr.(string) {
		case tai.BridgeAttrL2vni:
			attrs[attr.(string)] = tableBridge.Vni
		}
	}

	return attrs, nil
}

func (d bridgeAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
