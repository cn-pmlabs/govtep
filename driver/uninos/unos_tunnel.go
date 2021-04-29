package driver

import (
	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"
)

type tunnelAPI struct {
	moduleID int
}

var tunnelAPIs = tunnelAPI{
	moduleID: tai.ObjectIDTunnel,
}

func (v tunnelAPI) CreateObject(obj interface{}) error {
	objTunnel := obj.(tai.TunnelObj)

	tunnelCfg := cdb.TableTunnel{
		Name:     objTunnel.Name,
		Type:     cdb.TunnelTypeVxlanTunnel,
		MacLearn: []string{cdb.TunnelDefaultMacLearn},
		DestPort: []int{cdb.TunnelDefaultDestPort},
		SrcIP:    []string{objTunnel.Ipaddr},
	}

	/* if objTunnel.Anycast == true {
		tunnelCfg.AnycastIP = []string{objTunnel.Ipaddr}
	} else {
		tunnelCfg.SrcIP = []string{objTunnel.Ipaddr}
	} */

	tunnelIndex := cdb.TunnelIndex{
		Name: objTunnel.Name,
	}
	if _, err := cdb.TunnelGetByIndex(tunnelIndex); err == nil {
		log.Info("[Driver] Tunnel %s already exist\n", objTunnel.Name)
		return nil
	}

	_, err := cdb.TunnelAdd(tunnelCfg)
	if err != nil {
		return err
	}

	// need add tunnel to all VTEP BD, but bd only support one tunnel in configDB ??????

	return nil
}

func (v tunnelAPI) RemoveObject(obj interface{}) error {
	objTunnel := obj.(tai.TunnelObj)

	tunnelIndex := cdb.TunnelIndex{
		Name: objTunnel.Name,
	}

	err := cdb.TunnelDelByIndex(tunnelIndex)
	if err != nil {
		return err
	}
	return nil
}

func (v tunnelAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objTunnel := obj.(tai.TunnelObj)

	tunnelIndex := cdb.TunnelIndex{
		Name: objTunnel.Name,
	}
	if _, err := cdb.TunnelGetByIndex(tunnelIndex); err != nil {
		log.Warning("[Driver] Tunnel %s not found when add attr\n", objTunnel.Name)
		return nil
	}

	if attrs[tai.TunnelAttrIpaddr] != nil {
		cdb.TunnelSetField(tunnelIndex, cdb.TunnelFieldSrcIP, attrs[tai.TunnelAttrIpaddr])
	}

	if attrs[tai.TunnelAttrRmacMap] != nil {
		cdb.TunnelSetField(tunnelIndex, cdb.TunnelFieldRmacMap, attrs[tai.TunnelAttrRmacMap])
	}

	return nil
}

func (v tunnelAPI) DelObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {

	return nil
}

func (v tunnelAPI) SetObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objTunnel := obj.(tai.TunnelObj)

	tunnelIndex := cdb.TunnelIndex{
		Name: objTunnel.Name,
	}

	if attrs[tai.TunnelAttrIpaddr] != nil {
		cdb.TunnelSetField(tunnelIndex, cdb.TunnelFieldSrcIP, attrs[tai.TunnelAttrIpaddr])
	}

	return nil
}

func (v tunnelAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v tunnelAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
