package driver

import (
	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"

	"github.com/ebay/libovsdb"
)

type l3portAPI struct {
	moduleID int
}

var l3portAPIs = l3portAPI{
	moduleID: tai.ObjectIDL3Port,
}

func (v l3portAPI) CreateObject(obj interface{}) error {

	return nil
}

func (v l3portAPI) RemoveObject(interface{}) error {
	return nil
}

func (v l3portAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objL3port := obj.(tai.L3portObj)

	ifIndex := cdb.InterfaceIndex{
		Name: objL3port.Name,
		Type: cdb.InterfaceTypeBridgeDomain,
	}
	_, err := cdb.InterfaceGetByIndex(ifIndex)
	if err != nil {
		log.Warning("[Driver] interface %s type %s not exist\n", ifIndex.Name, ifIndex.Type)
		return nil
	}

	if attrs[tai.L3portAttrVrfBinding] != nil {
		vrfName := attrs[tai.L3portAttrVrfBinding].(string)
		vrfIndex := cdb.VrfIndex{
			Name: vrfName,
		}
		tableVrf, err := cdb.VrfGetByIndex(vrfIndex)
		if err != nil {
			log.Warning("[Driver] Interface %s binding vrf %s not exist\n", ifIndex.Name, vrfName)
			return nil
		}

		cdb.InterfaceUpdateVrfAddvalue(ifIndex, []libovsdb.UUID{{GoUUID: tableVrf.UUID}})
	}

	return nil
}

func (v l3portAPI) DelObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objL3port := obj.(tai.L3portObj)

	ifIndex := cdb.InterfaceIndex{
		Name: objL3port.Name,
		Type: cdb.InterfaceTypeBridgeDomain,
	}
	_, err := cdb.InterfaceGetByIndex(ifIndex)
	if err != nil {
		log.Warning("[Driver] interface %s type %s not exist\n", ifIndex.Name, ifIndex.Type)
		return nil
	}

	if attrs[tai.L3portAttrVrfBinding] != nil {
		vrfName := attrs[tai.L3portAttrVrfBinding].(string)
		vrfIndex := cdb.VrfIndex{
			Name: vrfName,
		}
		tableVrf, err := cdb.VrfGetByIndex(vrfIndex)
		if err != nil {
			log.Warning("[Driver] Interface %s binding vrf %s not exist\n", ifIndex.Name, vrfName)
			return nil
		}

		cdb.InterfaceUpdateVrfDelvalue(ifIndex, []libovsdb.UUID{{GoUUID: tableVrf.UUID}})
	}

	return nil
}

func (v l3portAPI) SetObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v l3portAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v l3portAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
