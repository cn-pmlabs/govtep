package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"

	"github.com/ebay/libovsdb"
)

// AutoGatewayConfObj attr list
const (
	AutoGatewayConfAttrBdname       = "autogatewayconf_bdname"
	AutoGatewayConfAttrVlan         = "autogatewayconf_vlan"
	AutoGatewayConfAttrIP           = "autogatewayconf_ip"
	AutoGatewayConfAttrPhysicalPort = "autogatewayconf_physicalport"
)

// AutoGatewayConfObj ...
type AutoGatewayConfObj struct {
	Bdname       string
	Vlan         int
	IP           string
	PhysicalPort string
	Vrf          string
}

func rowToAutoGatewayConfObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableAutoGatewayConf := vtepdb.ConvertRowToAutoGatewayConf(libovsdb.ResultRow(row.Fields))
	log.Info("Table autogatewayconf : %+v.\n", tableAutoGatewayConf)

	obj := AutoGatewayConfObj{
		Bdname:       tableAutoGatewayConf.Bdname,
		Vlan:         tableAutoGatewayConf.Vlan,
		IP:           tableAutoGatewayConf.IP,
		PhysicalPort: tableAutoGatewayConf.PhysicalPort,
		Vrf:          tableAutoGatewayConf.Vrf,
	}
	attrs := map[interface{}]interface{}{
		AutoGatewayConfAttrBdname:       tableAutoGatewayConf.Bdname,
		AutoGatewayConfAttrVlan:         tableAutoGatewayConf.Vlan,
		AutoGatewayConfAttrIP:           tableAutoGatewayConf.IP,
		AutoGatewayConfAttrPhysicalPort: tableAutoGatewayConf.PhysicalPort,
	}
	return obj, attrs
}

func rowToAutoGatewayConfAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.AutoGatewayConfFieldBdname:
			if bdname, ok := value.(string); ok {
				attrs[AutoGatewayConfAttrBdname] = bdname
			}
		case vtepdb.AutoGatewayConfFieldVlan:
			if vlan, ok := value.(string); ok {
				attrs[AutoGatewayConfAttrVlan] = vlan
			}
		case vtepdb.AutoGatewayConfFieldIP:
			if ip, ok := value.(string); ok {
				attrs[AutoGatewayConfAttrIP] = ip
			}
		case vtepdb.AutoGatewayConfFieldPhysicalPort:
			if physicalPort, ok := value.(string); ok {
				attrs[AutoGatewayConfAttrPhysicalPort] = physicalPort
			}
		}
	}
	return attrs
}
