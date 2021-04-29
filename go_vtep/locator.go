package govtep

import (
	"fmt"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnsb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnsouthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// GatewayInitDone identify wheather vnet process can be done
var GatewayInitDone bool

// Locator in vtep db
type Locator struct {
	UUID        string
	Type        string // vxlan
	Ipaddr      string
	Rmac        string
	ChassisName string // OVN SB.Chassis name
}

func encapNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate) {
	var err error
	switch op {
	case odbc.OpUpdate:
		encapUpdate(rowUpdate.New, rowUpdate.Old)
	}
	if err != nil {
		log.Error("locatorNotifyUpdate op %s failed: %v\n", op, err)
		return
	}
}

func encapUpdate(newrow libovsdb.Row, oldrow libovsdb.Row) {
	var err error

	for field, oldValue := range oldrow.Fields {
		switch field {
		// when encap ip changed, the encap would delete old then add new encap
		case ovnsb.EncapFieldIP:
			// err = encapUpdateIP(newrow, oldValue.(string))
		default:
			continue
		}

		if err != nil {
			log.Warning("update failed %s old-value %v failed: %v\n", field, oldValue, err)
			return
		}
	}
}

func locatorNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate) {
	var err error
	switch op {
	case odbc.OpInsert:
		err = locatorCreate(rowUpdate.New)
	case odbc.OpDelete:
		err = locatorRemove(rowUpdate.Old)
	case odbc.OpUpdate:
		err = locatorUpdate(rowUpdate.New, rowUpdate.Old)
	}
	if err != nil {
		log.Error("locatorNotifyUpdate op %s failed: %v\n", op, err)
		return
	}
}

func locatorCreate(chassis libovsdb.Row) error {
	var err error
	tableChassis := ovnsb.ConvertRowToChassis(libovsdb.ResultRow(chassis.Fields))
	// check chassis and encap
	if len(tableChassis.Encaps) == 0 {
		// the encap-ip must set when chassis created, chassis.encap min=1
		return fmt.Errorf("Chassis don't have encap, ignored")
	}
	if len(tableChassis.Name) == 0 {
		return fmt.Errorf("Get chassis name failed")
	}

	// check existence
	locatorIndex := vtepdb.LocatorIndex{
		ChassisName: tableChassis.Name,
	}
	dbLocator, err := vtepdb.LocatorGetByIndex(locatorIndex)
	if err == nil {
		if dbLocator.LocalLocator == true {
			GatewayInitDone = true
		}

		log.Info("Locator for chassis %s already exist", tableChassis.Name)
		return nil
	}

	// LocatorAdd
	tableLocator := vtepdb.TableLocator{
		Type:        vtepdb.LocatorTypeVxlan,
		ChassisName: tableChassis.Name,
	}

	for _, encapUUID := range tableChassis.Encaps {
		tableEncap, err := ovnsb.EncapGetByUUID(encapUUID.GoUUID)
		if err != nil {
			log.Warning("Encap %s not found", encapUUID.GoUUID)
			continue
		}

		tableLocator.Ipaddr = append(tableLocator.Ipaddr, tableEncap.IP)
		tableLocator.RouteMac = tableEncap.RouterMac
	}

	// local locator decision
	psIndex := vtepdb.PhysicalSwitchIndex1{
		SystemID: tableChassis.Name,
	}
	tablePS, err := vtepdb.PhysicalSwitchGetByIndex(psIndex)
	if err == nil {
		if tablePS.GatewayGroup == true || tablePS.GroupMember == false {
			tableLocator.LocalLocator = true
		}
	}

	// if the locator is local vtep chassis, add other chassis rmac to ip mapping
	if tableLocator.LocalLocator == true {
		rmacMap := make(map[interface{}]interface{})

		var conditions []interface{}
		conditions = append(conditions, libovsdb.
			NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))
		rows, num := vtepdb.LocatorGet(conditions)
		if num > 0 {
			for _, row := range rows {
				dbLocator := vtepdb.ConvertRowToLocator(row)
				if dbLocator.LocalLocator == false {
					for _, rip := range dbLocator.Ipaddr {
						rmacMap[rip] = dbLocator.RouteMac
					}
				}
			}
		}

		tableLocator.RmacMap = rmacMap
	} else {
		// update existed local locator
		var conditions []interface{}
		conditions = append(conditions, libovsdb.
			NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))
		rows, num := vtepdb.LocatorGet(conditions)
		if num > 0 {
			for _, row := range rows {
				dbLocator := vtepdb.ConvertRowToLocator(row)
				if dbLocator.LocalLocator == true {
					dbLocatorIndex := vtepdb.LocatorIndex{
						ChassisName: dbLocator.ChassisName,
					}

					for _, rip := range tableLocator.Ipaddr {
						rmacUpdate := make(map[interface{}]interface{})
						if dbLocator.RmacMap[rip] != nil {
							rmacUpdate[rip] = dbLocator.RmacMap[rip]
							vtepdb.LocatorUpdateRmacMapDelkey(dbLocatorIndex, rmacUpdate)
						}

						rmacUpdate[rip] = tableLocator.RouteMac
						vtepdb.LocatorUpdateRmacMapSetkey(dbLocatorIndex, rmacUpdate)
					}
				}
			}
		}
	}

	_, err = vtepdb.LocatorAdd(tableLocator)

	if err == nil {
		if tableLocator.LocalLocator == true {
			GatewayInitDone = true
			vnetProcessAll()
		}
	}

	return err
}

func locatorRemove(chassis libovsdb.Row) error {
	tableChassis := ovnsb.ConvertRowToChassis(libovsdb.ResultRow(chassis.Fields))

	// check chassis and encap
	if len(tableChassis.Name) == 0 {
		return fmt.Errorf("Get chassis name failed")
	}

	// check existence
	locatorIndex := vtepdb.LocatorIndex{
		ChassisName: tableChassis.Name,
	}
	tableLocator, err := vtepdb.LocatorGetByIndex(locatorIndex)
	if err != nil {
		return fmt.Errorf("Locator %v not exist", locatorIndex)
	}

	if tableLocator.LocalLocator == false {
		// update existed local locator
		var conditions []interface{}
		conditions = append(conditions, libovsdb.
			NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))
		rows, num := vtepdb.LocatorGet(conditions)
		if num > 0 {
			for _, row := range rows {
				dbLocator := vtepdb.ConvertRowToLocator(row)
				if dbLocator.LocalLocator == true {
					dbLocatorIndex := vtepdb.LocatorIndex{
						ChassisName: dbLocator.ChassisName,
					}

					for _, rip := range tableLocator.Ipaddr {
						rmacUpdate := make(map[interface{}]interface{})
						if rmac, ok := dbLocator.RmacMap[rip]; ok {
							rmacUpdate[rip] = rmac
							vtepdb.LocatorUpdateRmacMapDelkey(dbLocatorIndex, rmacUpdate)
						}
					}
				}
			}
		}
	}

	err = vtepdb.LocatorDelByIndex(locatorIndex)

	if err == nil {
		if tableLocator.LocalLocator == true {
			log.Warning("LocalLocator %s remove\n", tableLocator.ChassisName)
			GatewayInitDone = false
			vnetRemoveAll()
		}
	}

	return err
}

func locatorUpdate(newrow libovsdb.Row, oldrow libovsdb.Row) error {
	var err error

	for field, oldValue := range oldrow.Fields {
		switch field {
		case ovnsb.ChassisFieldEncaps:
			err = locatorUpdateEncap(newrow, oldValue)
		default:
			continue
		}

		if err != nil {
			log.Warning("update failed %s old-value %v failed: %v\n", field, oldValue, err)
			return nil
		}
	}

	return nil
}

func locatorUpdateEncap(newrow libovsdb.Row, oldValue interface{}) error {
	tableChassis := ovnsb.ConvertRowToChassis(libovsdb.ResultRow(newrow.Fields))

	// check chassis and encap
	if len(tableChassis.Encaps) == 0 {
		return fmt.Errorf("Chassis don't have encap, ignored")
	}

	locatorIndex := vtepdb.LocatorIndex{
		ChassisName: tableChassis.Name,
	}
	tableLocator, err := vtepdb.LocatorGetByIndex(locatorIndex)
	if err != nil {
		log.Warning("Locator for chassis %s not exist", tableChassis.Name)
		return nil
	}

	var locatorIPs []string
	for _, encapUUID := range tableChassis.Encaps {
		tableEncap, err := ovnsb.EncapGetByUUID(encapUUID.GoUUID)
		if err != nil {
			log.Warning("Encap %s not found", encapUUID.GoUUID)
			continue
		}

		locatorIPs = append(locatorIPs, tableEncap.IP)

		if tableLocator.RouteMac != tableEncap.RouterMac {
			tableLocator.RouteMac = tableEncap.RouterMac
			err = vtepdb.LocatorSetField(locatorIndex, vtepdb.LocatorFieldRouteMac, tableEncap.RouterMac)
		}
	}

	if len(locatorIPs) < 1 {
		log.Warning("Locator for chassis %s update ip invalid", tableChassis.Name)
		return nil
	}

	if len(tableLocator.Ipaddr) != len(locatorIPs) ||
		tableLocator.Ipaddr[0] != locatorIPs[0] {
		err = vtepdb.LocatorSetField(locatorIndex, vtepdb.LocatorFieldIpaddr, locatorIPs)

		// update route nh
		routeNhUpdateForLocator(tableLocator.UUID, locatorIPs[0])
	}

	// don't check wheather encap ip and mac changed,
	// assume they changed and update locator locator rmac_map
	//if tableLocator.Ipaddr != tableEncap.IP || tableLocator.RouteMac != tableEncap.RouterMac {
	if tableLocator.LocalLocator == false {
		// update existed local locator
		var conditions []interface{}
		conditions = append(conditions, libovsdb.
			NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))
		rows, num := vtepdb.LocatorGet(conditions)
		if num > 0 {
			for _, row := range rows {
				dbLocator := vtepdb.ConvertRowToLocator(row)
				if dbLocator.LocalLocator == true {
					dbLocatorIndex := vtepdb.LocatorIndex{
						ChassisName: dbLocator.ChassisName,
					}

					for _, rip := range tableLocator.Ipaddr {
						rmacUpdate := make(map[interface{}]interface{})
						if rmac, ok := dbLocator.RmacMap[rip]; ok {
							rmacUpdate[rip] = rmac
							vtepdb.LocatorUpdateRmacMapDelkey(dbLocatorIndex, rmacUpdate)
						}
					}

					for _, rip := range locatorIPs {
						rmacUpdate := make(map[interface{}]interface{})
						rmacUpdate[rip] = tableLocator.RouteMac
						vtepdb.LocatorUpdateRmacMapSetkey(dbLocatorIndex, rmacUpdate)
					}
				}
			}
		}
	}

	return nil
}
