package driver

import (
	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"
)

type fdbAPI struct {
	moduleID int
}

var fdbAPIs = fdbAPI{
	moduleID: tai.ObjectIDFDB,
}

func (v fdbAPI) CreateObject(obj interface{}) error {
	objFdb := obj.(tai.FdbObj)

	fdbIndex := cdb.FdbIndex{
		Address:       objFdb.Mac,
		ForwardDomain: objFdb.Bridge,
	}
	_, err := cdb.FdbGetByIndex(fdbIndex)
	if err == nil {
		log.Info("[Driver] fdb %+v already exist\n", fdbIndex)
		return nil
	}

	fdbCfg := cdb.TableFdb{
		Address:       objFdb.Mac,
		ForwardDomain: objFdb.Bridge,
	}

	_, err = cdb.FdbAdd(fdbCfg)
	if err != nil {
		return err
	}
	return nil
}

func (v fdbAPI) RemoveObject(obj interface{}) error {
	objFdb := obj.(tai.FdbObj)

	fdbIndex := cdb.FdbIndex{
		Address:       objFdb.Mac,
		ForwardDomain: objFdb.Bridge,
	}

	err := cdb.FdbDelByIndex(fdbIndex)
	if err != nil {
		return err
	}
	return nil
}

func (v fdbAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objFdb := obj.(tai.FdbObj)

	fdbIndex := cdb.FdbIndex{
		Address:       objFdb.Mac,
		ForwardDomain: objFdb.Bridge,
	}
	_, err := cdb.FdbGetByIndex(fdbIndex)
	if err != nil {
		log.Warning("[Driver] fdb %+v not exist\n", fdbIndex)
		return nil
	}

	for attr, attrValue := range attrs {
		switch attr {
		case tai.FdbAttrRemoteIP:
			if remoteIP, ok := attrValue.(string); ok {
				cdb.FdbSetField(fdbIndex, cdb.FdbFieldRemoteIP, []string{remoteIP})
			}
		case tai.FdbAttrTunnelName:
			if TunnelName, ok := attrValue.(string); ok {
				cdb.FdbSetField(fdbIndex, cdb.FdbFieldTunnelName, []string{TunnelName})
			}
		}
	}

	return nil
}

func (v fdbAPI) DelObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v fdbAPI) SetObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v fdbAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v fdbAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
