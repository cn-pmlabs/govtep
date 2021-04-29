package driver

import (
	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"
)

type neighbourAPI struct {
	moduleID int
}

var neighbourAPIs = neighbourAPI{
	moduleID: tai.ObjectIDNeighbour,
}

func (v neighbourAPI) CreateObject(obj interface{}) error {
	objNeighbour := obj.(tai.NeighbourObj)

	neighbourCfg := cdb.TableNeighbor{
		IP: objNeighbour.Ipaddr,
	}
	neighbourIndex := cdb.NeighborIndex{
		IP: objNeighbour.Ipaddr,
	}

	_, err := cdb.NeighborGetByIndex(neighbourIndex)
	if err == nil {
		log.Info("[Driver] Neighbour for %s already exist\n", neighbourIndex.IP)
		return nil
	}

	_, err = cdb.NeighborAdd(neighbourCfg)
	if err != nil {
		return err
	}
	return nil
}

func (v neighbourAPI) RemoveObject(obj interface{}) error {
	objNeighbour := obj.(tai.NeighbourObj)

	neighbourIndex := cdb.NeighborIndex{
		IP: objNeighbour.Ipaddr,
	}

	err := cdb.NeighborDelByIndex(neighbourIndex)
	if err != nil {
		return err
	}
	return nil
}

func (v neighbourAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objNeighbour := obj.(tai.NeighbourObj)

	neighbourIndex := cdb.NeighborIndex{
		IP: objNeighbour.Ipaddr,
	}
	_, err := cdb.NeighborGetByIndex(neighbourIndex)
	if err != nil {
		log.Warning("[Driver] Neighbour for %s not exist\n", neighbourIndex.IP)
		return nil
	}

	for attr, value := range attrs {
		switch attr {
		case tai.NeighbourAttrMacaddr:
			if cdb.NeighborSetField(neighbourIndex, cdb.NeighborFieldMac, value.(string)) != nil {
				log.Warning("[Driver] Neighbour update mac %s for %s failed\n", value.(string), neighbourIndex.IP)
			}
		case tai.NeighbourAttrOutPort:
			if cdb.NeighborSetField(neighbourIndex, cdb.NeighborFieldOutport, value.(string)) != nil {
				log.Warning("[Driver] Neighbour update outport %s for %s failed\n", value.(string), neighbourIndex.IP)
			}

			bdIndex := cdb.BridgeIndex{
				Name: value.(string),
			}
			_, err := cdb.BridgeGetByIndex(bdIndex)
			if err != nil {
				log.Warning("[Driver] BD %s not exist for NeighbourAttrOutPort add\n", bdIndex.Name)
				continue
			}

			cdb.NeighborSetField(neighbourIndex, cdb.NeighborFieldBridge, bdIndex.Name)
			cdb.NeighborSetField(neighbourIndex, cdb.NeighborFieldVxlanid, bdIndex.Name[2:])

		case tai.NeighbourAttrRemoteIP:
			if cdb.NeighborSetField(neighbourIndex, cdb.NeighborFieldRemoteIP, value.(string)) != nil {
				log.Warning("[Driver] Neighbour update remote ip %s for %s failed\n", value.(string), neighbourIndex.IP)
			}
		}
	}

	return nil
}

func (v neighbourAPI) DelObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v neighbourAPI) SetObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v neighbourAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v neighbourAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
