package tai

import (
	"errors"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// TAI tunnel attr list
const (
	TunnelAttrTunnelKey = "tunnel_tunnel_key"
	TunnelAttrIpaddr    = "tunnel_ipaddr"
	TunnelAttrRmacMap   = "tunnel_rmac_map"
)

// LocalPhsicalSwitchTunnelName save tunnel name for phsical switch
var LocalPhsicalSwitchTunnelName string

// TunnelObj ...
type TunnelObj struct {
	Name    string
	Type    string
	Ipaddr  string // support one vxlan tunnel source ip
	Anycast bool
}

func getTunnelName(chassisName string) string {
	tunnelName := chassisName
	if len(tunnelName) > 6 {
		tunnelName = tunnelName[:6]
	}
	tunnelName = "vxlan-" + "vtep-" + tunnelName
	return tunnelName
}

func rowToTunnelObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableLocator := vtepdb.ConvertRowToLocator(libovsdb.ResultRow(row.Fields))

	if tableLocator.LocalLocator == false {
		log.Warning("[TAI] not local phsical switch locator, ignore tunnel modification\n")
		return nil, nil
	}

	tunnelName := getTunnelName(tableLocator.ChassisName)

	if len(tableLocator.Ipaddr) != 1 {
		log.Warning("[TAI] system only support one vxlan tunnel with single source ip for now\n")
		return nil, nil
	}

	obj := TunnelObj{
		Name:   tunnelName,
		Type:   tableLocator.Type,
		Ipaddr: tableLocator.Ipaddr[0],
	}

	// gateway group should do vxlan tunnel anycast
	psIndex := vtepdb.PhysicalSwitchIndex1{
		SystemID: tableLocator.ChassisName,
	}
	tablePS, err := vtepdb.PhysicalSwitchGetByIndex(psIndex)
	if err == nil {
		if tablePS.GatewayGroup == true {
			obj.Anycast = true
		}
	}

	LocalPhsicalSwitchTunnelName = tunnelName
	attrs := map[interface{}]interface{}{
		TunnelAttrTunnelKey: tableLocator.TunnelKey,
		TunnelAttrRmacMap:   tableLocator.RmacMap,
	}
	return obj, attrs
}

func rowToTunnelAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.LocatorFieldTunnelKey:
			if tunnelKey, ok := value.(int); ok {
				attrs[TunnelAttrTunnelKey] = tunnelKey
			}
		case vtepdb.LocatorFieldIpaddr:
			if ipaddr, ok := value.(string); ok {
				attrs[TunnelAttrIpaddr] = ipaddr
			}
		case vtepdb.LocatorFieldRmacMap:
			if rmacMap, ok := value.(map[interface{}]interface{}); ok {
				attrs[TunnelAttrRmacMap] = rmacMap
			}
		}
	}

	return attrs
}

func taiGetLocatorObj(uuid string) (TunnelObj, error) {
	var lctObj TunnelObj
	condition := libovsdb.NewCondition("_uuid", "==", odbc.StringToGoUUID(uuid))
	operation := libovsdb.Operation{
		Op:    odbc.OpSelect,
		Table: odbc.VTEP_Locator,
		Where: []interface{}{condition},
	}
	results, err := taiDBClient.Transact(odbc.VTEPDB, operation)
	if err != nil || len(results) == 0 || results[0].Count == 0 {
		return lctObj, errors.New("NOT Found")
	}
	result := results[0]
	for _, row := range result.Rows {
		encapType, ok1 := row["type"].(string)
		ipaddr, ok2 := row["ipaddr"].(string)
		//rmac, ok3 := row["rmac"].(string)
		if ok1 && ok2 {
			lctObj.Type = encapType
			lctObj.Ipaddr = ipaddr
			//lctObj.Rmac = rmac
			return lctObj, nil
		}
	}
	return lctObj, errors.New("NOT FOUND")
}

func taiGetLocatorObjSet(uuids []string) []TunnelObj {
	var tls []TunnelObj
	for i, uuid := range uuids {
		tls[i], _ = taiGetLocatorObj(uuid)
	}
	return tls
}
