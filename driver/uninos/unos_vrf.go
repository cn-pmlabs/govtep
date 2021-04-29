package driver

import (
	"fmt"
	"strconv"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"

	"github.com/ebay/libovsdb"
)

func getVniByVrfName(vrfName string) int {
	vniStr := vrfName[3:]
	vni, err := strconv.Atoi(vniStr)
	if err != nil {
		return 0
	}
	return vni
}

type vrfAPI struct {
	moduleID int
}

var vrfAPIs = vrfAPI{
	moduleID: tai.ObjectIDVrf,
}

func (v vrfAPI) CreateObject(obj interface{}) error {
	var (
		err          error
		vni          int
		vrfIndex     cdb.VrfIndex
		vrfCfg       cdb.TableVrf
		vrfUUID      string
		ifIndex      cdb.InterfaceIndex
		interfaceCfg cdb.TableInterface

		conditions []interface{}
		rows       []libovsdb.ResultRow
		num        int
	)

	objVrf := obj.(tai.VrfObj)

	vni = getVniByVrfName(objVrf.Name)
	if vni < cdb.VrfL3vniMin || vni > cdb.VrfL3vniMax {
		return fmt.Errorf("[Driver] Invalid vrf name %s", objVrf.Name)
	}

	vrfIndex = cdb.VrfIndex{
		Name: objVrf.Name,
	}
	_, err = cdb.VrfGetByIndex(vrfIndex)
	if err == nil {
		log.Info("[Driver] Vrf %s already exist", objVrf.Name)
		goto bdvrf
	}

	vrfCfg = cdb.TableVrf{
		Name:  objVrf.Name,
		L3vni: []int{vni},
	}

	vrfUUID, err = cdb.VrfAdd(vrfCfg)
	if err != nil {
		return err
	}

bdvrf:
	interfaceCfg = cdb.TableInterface{
		Name:        "Bd" + vrfCfg.Name,
		Type:        cdb.InterfaceTypeBridgeDomain,
		AdminStatus: []string{cdb.InterfaceDefaultAdminStatus},
		Mtu:         []int{1500},
		ProxyArp:    []string{cdb.InterfaceDefaultProxyArp},
		SwitchPort:  []string{cdb.InterfaceDefaultSwitchPort},
		Bandwidth:   []int{cdb.InterfaceDefaultBandwidth},
		Vrf:         []libovsdb.UUID{{GoUUID: vrfUUID}},
	}

	// configure BdVrf.mac
	conditions = append(conditions, libovsdb.
		NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))
	rows, num = vtepdb.LocatorGet(conditions)
	if num > 0 {
		for _, row := range rows {
			dbLocator := vtepdb.ConvertRowToLocator(row)
			if dbLocator.LocalLocator == true {
				interfaceCfg.Mac = []string{dbLocator.RouteMac}
				break
			}
		}
	}

	ifIndex = cdb.InterfaceIndex{
		Name: "Bd" + vrfCfg.Name,
		Type: cdb.InterfaceTypeBridgeDomain,
	}
	_, err = cdb.InterfaceGetByIndex(ifIndex)
	if err == nil {
		log.Info("[Driver] interface %s type %s already exist\n", ifIndex.Name, ifIndex.Type)
		goto pbr
	}

	_, err = cdb.InterfaceAdd(interfaceCfg)
	if err != nil {
		return err
	}

pbr:
	pbrAclCreate(objVrf.Name)

	return nil
}

func (v vrfAPI) RemoveObject(obj interface{}) error {
	var (
		err            error
		vrfIndex       cdb.VrfIndex
		interfaceIndex cdb.InterfaceIndex
	)

	objVrf := obj.(tai.VrfObj)

	vrfIndex = cdb.VrfIndex{
		Name: objVrf.Name,
	}
	err = cdb.VrfDelByIndex(vrfIndex)
	if err != nil {
		log.Warning("[Driver] Vrf %s Del failed", objVrf.Name)
	}

	interfaceIndex = cdb.InterfaceIndex{
		Name: "Bd" + vrfIndex.Name,
		Type: cdb.InterfaceTypeBridgeDomain,
	}
	err = cdb.InterfaceDelByIndex(interfaceIndex)
	if err != nil {
		log.Warning("[Driver] Interface %s Del failed", interfaceIndex.Name)
	}

	pbrAclRemove(objVrf.Name)

	return nil
}

func (v vrfAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objVrf := obj.(tai.VrfObj)

	vrfIndex := cdb.VrfIndex{
		Name: objVrf.Name,
	}
	_, err := cdb.VrfGetByIndex(vrfIndex)
	if err != nil {
		log.Warning("[Driver] Vrf %+v not exist\n", vrfIndex)
		return nil
	}

	for attr, attrValue := range attrs {
		switch attr {
		case tai.VrfAttrTunnel:
			if tunnelName, ok := attrValue.(string); ok {
				tunnelIndex := cdb.TunnelIndex{
					Name: tunnelName,
				}
				tunnel, err := cdb.TunnelGetByIndex(tunnelIndex)
				if err != nil {
					log.Warning("[Driver] Vrf %+v update tunnel %s not exist\n", vrfIndex, tunnelName)
					return nil
				}

				cdb.VrfSetField(vrfIndex, cdb.VrfFieldTunnel, []libovsdb.UUID{{GoUUID: tunnel.UUID}})
			}
		}
	}

	return nil
}

func (v vrfAPI) DelObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v vrfAPI) SetObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objVrf := obj.(tai.VrfObj)

	vrfIndex := cdb.VrfIndex{
		Name: objVrf.Name,
	}
	_, err := cdb.VrfGetByIndex(vrfIndex)
	if err != nil {
		log.Warning("[Driver] Vrf %+v not exist\n", vrfIndex)
		return nil
	}

	for attr, attrValue := range attrs {
		switch attr {
		case tai.VrfAttrTunnel:
			if tunnelName, ok := attrValue.(string); ok {
				tunnelIndex := cdb.TunnelIndex{
					Name: tunnelName,
				}
				tunnel, err := cdb.TunnelGetByIndex(tunnelIndex)
				if err != nil {
					log.Warning("[Driver] Vrf %+v update tunnel %s not exist\n", vrfIndex, tunnelName)
					return nil
				}

				cdb.VrfSetField(vrfIndex, cdb.VrfFieldTunnel, []libovsdb.UUID{{GoUUID: tunnel.UUID}})
			}
		}
	}

	return nil
}

func (v vrfAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v vrfAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
