package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/ebay/libovsdb"
)

// L3port attr list
const (
	L3portAttrVrfBinding = "l3port_vrfbinding"
	L3portAttrIpaddr     = "l3port_ipaddr"
	L3portAttrMacaddr    = "l3port_macaddr"
	L3portAttrVlanTag    = "l3port_vlantag"
)

// L3portObj ...
type L3portObj struct {
	Name               string //port name，如eth1、eth1.1
	PhysicalParentPort string //物理父port
}

func rowToL3portObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableL3port := vtepdb.ConvertRowToL3port(libovsdb.ResultRow(row.Fields))

	obj := L3portObj{
		Name:               tableL3port.Name,
		PhysicalParentPort: tableL3port.PhyparentPort,
	}
	attrs := map[interface{}]interface{}{
		L3portAttrVrfBinding: tableL3port.Vrf,
		L3portAttrIpaddr:     tableL3port.Ipv4addr,
		L3portAttrVlanTag:    tableL3port.Vlantag,
		//L3portAttrMacaddr:tableL3port.
	}
	return obj, attrs
}

func rowToL3portAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.L3portFieldVrf:
			if vrf, ok := value.(string); ok {
				attrs[L3portAttrVrfBinding] = vrf
			}
		case vtepdb.L3portFieldIpv4addr:
			if ipAddr, ok := value.(string); ok {
				attrs[L3portAttrIpaddr] = ipAddr
			}
		case vtepdb.L3portFieldVlantag:
			if vlanTag, ok := value.(int); ok {
				attrs[L3portAttrVlanTag] = vlanTag
			}
		}
	}

	return attrs
}
