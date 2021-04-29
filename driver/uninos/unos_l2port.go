package driver

import (
	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"

	"github.com/ebay/libovsdb"
)

type l2portAPI struct {
	moduleID int
}

var l2portAPIs = l2portAPI{
	moduleID: tai.ObjectIDL2Port,
}

func (v l2portAPI) CreateObject(obj interface{}) error {
	objL2port := obj.(tai.L2portObj)

	bridgePortCfg := cdb.TableBridgePort{
		Name:           objL2port.Name,
		Bdname:         objL2port.BridgeName,
		TagMode:        []string{cdb.BridgePortDefaultTagMode},
		MacLearn:       []string{cdb.BridgePortDefaultMacLearn},
		MacLimit:       []int{cdb.BridgePortDefaultMacLimit},
		MacAlarm:       []string{cdb.BridgePortDefaultMacAlarm},
		MacLimitAction: []string{cdb.BridgePortDefaultMacLimitAction},
	}

	bridgePortIndex := cdb.BridgePortIndex{
		Name:   objL2port.Name,
		Bdname: objL2port.BridgeName,
	}
	if _, err := cdb.BridgePortGetByIndex(bridgePortIndex); err == nil {
		log.Info("[Driver] Bridge Port %s already exist", objL2port.Name)
		return nil
	}

	_, err := cdb.BridgePortAdd(bridgePortCfg)
	if err != nil {
		return err
	}

	interfaceCfg := cdb.TableInterface{
		Name:        objL2port.Name,
		Type:        cdb.InterfaceTypeBridgeDomain,
		AdminStatus: []string{cdb.InterfaceDefaultAdminStatus},
		Mtu:         []int{cdb.InterfaceDefaultMtu},
	}

	ifUUID, err := cdb.InterfaceAdd(interfaceCfg)
	if err != nil {
		return err
	}

	bridgeIndex := cdb.BridgeIndex{
		Name: objL2port.BridgeName,
	}
	err = cdb.BridgeSetField(bridgeIndex, cdb.BridgeFieldBridgePorts, libovsdb.UUID{GoUUID: ifUUID})
	if err != nil {
		return err
	}

	return nil
}

func (v l2portAPI) RemoveObject(obj interface{}) error {
	objL2port := obj.(tai.L2portObj)

	bridgePortIndex := cdb.BridgePortIndex{
		Name:   objL2port.Name,
		Bdname: objL2port.BridgeName,
	}
	err := cdb.BridgePortDelByIndex(bridgePortIndex)
	if err != nil {
		return err
	}

	interfaceIndex := cdb.InterfaceIndex{
		Name: objL2port.Name,
		Type: cdb.InterfaceTypeBridgeDomain,
	}
	err = cdb.InterfaceDelByIndex(interfaceIndex)
	if err != nil {
		return err
	}

	return nil
}

func (v l2portAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objL2port := obj.(tai.L2portObj)
	bridgePortIndex := cdb.BridgePortIndex{
		Name:   objL2port.Name,
		Bdname: objL2port.BridgeName,
	}

	if attrs[tai.L2portAttrVlanTag] != nil {
		cdb.BridgePortSetField(bridgePortIndex, cdb.BridgePortFieldTagMode, "tag")
	}

	return nil
}

func (v l2portAPI) DelObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v l2portAPI) SetObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v l2portAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v l2portAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
