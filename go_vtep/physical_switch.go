package govtep

import (
	"fmt"
	"net"
	"os"
	"time"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnsb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnsouthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// OVN SB encap types, vtep only support vxlan for now
const (
	EncapTypeVxlan  = "vxlan"
	EncapTypeStt    = "stt"
	EncapTypeGeneve = "geneve"
)

// Phsical_Switch fields update should processed
const (
	PSFieldSystemID = "system_id"
	PSFieldEncapIP  = "encap_ip"
)

// PhysicalSwitch ...
type PhysicalSwitch struct {
	UUID          string
	Name          string
	TaiDriverName string
	Description   string
	SystemID      string // Reported by PS
	ChassisName   string // Autogen from SystemID, write to SB.Chassis.hostname if not join PSGroup
	ManagementIP  []string
	EncapType     string // vxlan_over_ipv4
	EncapIP       string
	Ports         []string
}

// Chassis to SB.Chassis
type Chassis struct {
	UUID                string
	Name                string
	Hostname            string
	Encaps              []string
	VtepLogicalSwitches []string
}

// Encap to SB.Encap
type Encap struct {
	UUID        string
	Type        string
	IP          string
	Options     map[string]string
	ChassisName string
}

const (
	localChassisNo = iota
	localChassisGw
	localChassisGwGroup
)

var localChassisStatus int = localChassisNo

// SwitchConfFile default switch configure file path and name
var SwitchConfFile = "/etc/sonic/govtep/switch.conf"

// SwitchGroupConfFile default phsical switch vtep group configure file path and name
var SwitchGroupConfFile = "/etc/sonic/govtep/switch_group.conf"

func writeStringAtFileTail(filename string, data string) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(data))
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

func checkPhysicalSwitchValid(psIndex vtepdb.PhysicalSwitchIndex, tablePS vtepdb.TablePhysicalSwitch) error {
	var err error

	if tablePS.EncapType != "vxlan" {
		vtepdb.PhysicalSwitchSetField(psIndex, vtepdb.PhysicalSwitchFieldSwitchFaultStatus, []string{"unsuppoted encap type"})
		return fmt.Errorf("encap type must be vxlan")
	}
	if nil == net.ParseIP(tablePS.EncapIP) {
		vtepdb.PhysicalSwitchSetField(psIndex, vtepdb.PhysicalSwitchFieldSwitchFaultStatus, []string{"invalid encap ip"})
		return fmt.Errorf("invalid encap ip %s", tablePS.EncapIP)
	}
	_, err = net.ParseMAC(tablePS.RouterMac)
	if err != nil {
		vtepdb.PhysicalSwitchSetField(psIndex, vtepdb.PhysicalSwitchFieldSwitchFaultStatus, []string{"invalid route mac"})
		return fmt.Errorf("invalid route mac %s", tablePS.RouterMac)
	}

	return nil
}

func deleteOldChassisRecord(ps vtepdb.TablePhysicalSwitch) {
	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: ovnsb.InvalidUUID}))
	rows, num := ovnsb.ChassisGet(conditions)
	if num > 0 {
		for _, row := range rows {
			tableChassis := ovnsb.ConvertRowToChassis(row)
			if tableChassis.Name == ps.SystemID {
				ovnsb.ChassisDelByUUID(tableChassis.UUID)
				continue
			}

			for _, encapUUID := range tableChassis.Encaps {
				tableEncap, err := ovnsb.EncapGetByUUID(encapUUID.GoUUID)
				if err != nil {
					if tableEncap.Type == ps.EncapType && tableEncap.IP == ps.EncapIP && tableEncap.RouterMac == ps.RouterMac {
						ovnsb.ChassisDelByUUID(tableChassis.UUID)
						break
					}
				}
			}
		}
	}
}

// PhysicalSwitchInit process phsical switch (group)s in vtep DB
func PhysicalSwitchInit() {
	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))
	rows, num := vtepdb.PhysicalSwitchGet(conditions)
	if num > 0 {
		for _, row := range rows {
			rowPS := libovsdb.Row{
				Fields: row,
			}
			physicalSwitchCreate(rowPS)
		}
	}

	// wait for a while for vtep process phsicalswitch add chassis
	time.Sleep(1 * time.Second)

	return
}

func physicalSwitchNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate) {
	if OvnCentralConnected == false {
		log.Warning("Ovn central connection not established, process phsical switch update later\n")
		return
	}

	switch op {
	case odbc.OpInsert:
		physicalSwitchCreate(rowUpdate.New)
	case odbc.OpDelete:
		physicalSwitchRemove(rowUpdate.Old)
	case odbc.OpUpdate:
		physicalSwitchUpdate(rowUpdate.New, rowUpdate.Old)
	}
}

func physicalSwitchCreate(row libovsdb.Row) {
	tablePS := vtepdb.ConvertRowToPhysicalSwitch(row.Fields)

	psIndex := vtepdb.PhysicalSwitchIndex{
		Name: tablePS.Name,
	}

	// validity physical switch configuration
	err := checkPhysicalSwitchValid(psIndex, tablePS)
	if err != nil {
		log.Warning("Physical Switch %v\n", err)
		return
	} else {
		vtepdb.PhysicalSwitchSetField(psIndex, vtepdb.PhysicalSwitchFieldSwitchFaultStatus, []string{""})
	}

	if tablePS.SystemID == "" {
		systemID, err := odbc.NewUUIDString()
		if err != nil {
			log.Error("PhysicalSwitchInit get new random UUID failed\n")
			return
		}

		tablePS.SystemID = systemID
		vtepdb.PhysicalSwitchSet(psIndex, tablePS)
		log.Warning("Physical Switch %s get random system ID %s\n", tablePS.Name, tablePS.SystemID)

		// process chassis add in physicalSwitch update system id notification
		return
	}

	// using system ID as chassis name
	chassisIndex := ovnsb.ChassisIndex{
		Name: tablePS.SystemID,
	}
	_, err = ovnsb.ChassisGetByIndex(chassisIndex)
	if err == nil {
		log.Warning("Chassis for Physical Switch System ID %s existed\n", tablePS.SystemID)

		if tablePS.GatewayGroup == true {
			err = ovnsb.ChassisUpdateGatewayChassisMembersAddvalue(chassisIndex, []string{tablePS.UUID})
			if err != nil {
				log.Warning("ChassisUpdateGatewayChassisMembersAddvalue failed\n")
			}

			// process local locator
			locatorIndex := vtepdb.LocatorIndex{
				ChassisName: tablePS.SystemID,
			}
			tableLocator, err := vtepdb.LocatorGetByIndex(locatorIndex)

			// only when the locator is not LocalLocator, remove it then add local locator from chassis update
			// the locator is already LocalLocator, don't remove to avoid tunnel removal
			if err == nil && tableLocator.LocalLocator == false {
				vtepdb.LocatorDelByIndex(locatorIndex)
			}
		}

		return

		// -- the ps parameter might changed in vtepdb, delete chassis then add new one
		// remove this operation to avoid gateway port unconfigured after chassis removal
		// ovnsb.ChassisDelByIndex(chassisIndex)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Warning("get hostname failed, default empty\n")
		hostname = ""
	}

	tableChassis := ovnsb.TableChassis{
		Name:     tablePS.SystemID,
		Hostname: hostname,
	}

	if tablePS.GatewayGroup == true {
		tableChassis.HardwareGatewayChassis = []string{ovnsb.ChassisHardwareGatewayChassisPhsicalSwitchGroup}
		tableChassis.GatewayChassisMembers = []string{tablePS.UUID}
	} else {
		tableChassis.HardwareGatewayChassis = []string{ovnsb.ChassisHardwareGatewayChassisPhsicalSwitch}
	}

	ifaceTypes := make(map[interface{}]interface{})
	ifaceTypes["iface-types"] = "vxlan"
	tableChassis.ExternalIds = ifaceTypes
	tableChassis.OtherConfig = ifaceTypes

	tableEncap := ovnsb.TableEncap{
		Type:        ovnsb.EncapTypeVxlan,
		IP:          tablePS.EncapIP,
		RouterMac:   tablePS.RouterMac,
		ChassisName: tableChassis.Name,
	}
	options := make(map[interface{}]interface{})
	options["csum"] = "false"
	tableEncap.Options = options

	// TODO: tmp reuse original chassis add op, optimize later
	// because chassis encap min=1 api not supported for now
	err = chassisCreate(tableChassis, tableEncap)
	if err != nil {
		log.Warning("%v\n", err)
		return
	}
}

func physicalSwitchRemove(row libovsdb.Row) {
	var err error

	tablePS := vtepdb.ConvertRowToPhysicalSwitch(row.Fields)
	if tablePS.SystemID == "" {
		log.Error("Physical Switch Group %s System ID not configured, ignored chassis update\n", tablePS.Name)
		return
	}

	chassisIndex := ovnsb.ChassisIndex{
		Name: tablePS.SystemID,
	}
	tableChassis, err := ovnsb.ChassisGetByIndex(chassisIndex)
	if err != nil {
		log.Warning("Chassis %s not found", chassisIndex.Name)
		return
	}

	// vtep gateway remove chassis
	if tablePS.GatewayGroup == false {
		err = ovnsb.ChassisDelByIndex(chassisIndex)
		if err != nil {
			log.Warning("Chassis delete for phsical switch group removed failed %v\n", err)
		}
		return
	}

	// vtep gateway remove GatewayChassisMembers, then get otherMemberExist
	otherMemberExist := false
	for _, chassis := range tableChassis.GatewayChassisMembers {
		if chassis == tablePS.UUID {
			err = ovnsb.ChassisUpdateGatewayChassisMembersDelvalue(chassisIndex, []string{tablePS.UUID})
			if err != nil {
				log.Warning("ChassisUpdateGatewayChassisMembersDelvalue failed\n")
			}
		} else {
			otherMemberExist = true
		}
	}

	if false == otherMemberExist {
		err = ovnsb.ChassisDelByIndex(chassisIndex)
		if err != nil {
			log.Warning("Chassis delete for phsical switch group removed failed %v\n", err)
		}
	} else {
		// delete local locator
		locatorIndex := vtepdb.LocatorIndex{
			ChassisName: tablePS.SystemID,
		}
		_, err := vtepdb.LocatorGetByIndex(locatorIndex)

		if err == nil {
			vtepdb.LocatorDelByIndex(locatorIndex)
			GatewayInitDone = false
			vnetRemoveAll()
		}
	}

	return
}

func physicalSwitchUpdate(newrow libovsdb.Row, oldrow libovsdb.Row) {
	var err error

	for field, oldValue := range oldrow.Fields {
		switch field {
		case vtepdb.PhysicalSwitchFieldSystemID:
			// invalid systemID should not changed
			err = physicalSwitchUpdateSystemID(newrow, oldValue.(string))
		case vtepdb.PhysicalSwitchFieldEncapIP:
			err = physicalSwitchUpdateEncapIP(newrow, oldValue.(string))
		case vtepdb.PhysicalSwitchFieldRouterMac:
			err = physicalSwitchUpdateRouteMac(newrow, oldValue.(string))
		default:
			// Don't care about other field update
			continue
		}

		if err != nil {
			log.Warning("update failed %s old-value %v failed: %v\n", field, oldValue, err)
			return
		}
	}
}

func chassisCreate(chassis ovnsb.TableChassis, encap ovnsb.TableEncap) error {
	var ops []libovsdb.Operation

	// check Chassis exist
	chassisIndex := ovnsb.ChassisIndex{
		Name: chassis.Name,
	}
	_, err := ovnsb.ChassisGetByIndex(chassisIndex)
	if err == nil {
		return fmt.Errorf("Chassis %s exist", chassis.Name)
	}

	// check Encap exist
	var encapUUID libovsdb.UUID
	encapIndex := ovnsb.EncapIndex{
		Type: encap.Type,
		IP:   encap.IP,
	}
	tableEncap, err := ovnsb.EncapGetByIndex(encapIndex)
	if err == nil {
		log.Warning("Encap for chassis %s exist", chassis.Name)
	}

	var insertEncapOp libovsdb.Operation
	if err == nil {
		encapUUID = libovsdb.UUID{GoUUID: tableEncap.UUID}
	} else {
		// insert Encap Operation
		rowEncap := make(map[string]interface{})
		namedUUID, err := odbc.NewRowUUID()
		if err != nil {
			return err
		}

		rowEncap[ovnsb.EncapFieldType] = encap.Type
		rowEncap[ovnsb.EncapFieldChassisName] = encap.ChassisName
		rowEncap[ovnsb.EncapFieldIP] = encap.IP
		rowEncap[ovnsb.EncapFieldRouterMac] = encap.RouterMac

		if encap.Options != nil {
			oMap, err := libovsdb.NewOvsMap(encap.Options)
			if err != nil {
				return err
			}
			rowEncap[ovnsb.EncapFieldOptions] = oMap
		}
		insertEncapOp = libovsdb.Operation{
			Op:       odbc.OpInsert,
			Table:    ovnsb.Encap,
			Row:      rowEncap,
			UUIDName: namedUUID,
		}
		encapUUID.GoUUID = namedUUID
		ops = append(ops, insertEncapOp)
	}

	// insert Chassis Operation
	// encap must be inserted when chassis create, see schema Chassis.Encaps "min": 1
	encapU := []libovsdb.UUID{encapUUID}
	encapSet, err := libovsdb.NewOvsSet(encapU)
	externalIDs, err := libovsdb.NewOvsMap(chassis.ExternalIds)

	rowChassis := map[string]interface{}{
		ovnsb.ChassisFieldName:        chassis.Name,
		ovnsb.ChassisFieldHostname:    chassis.Hostname,
		ovnsb.ChassisFieldEncaps:      encapSet,
		ovnsb.ChassisFieldExternalIds: externalIDs,
		ovnsb.ChassisFieldOtherConfig: externalIDs,
	}

	if len(chassis.HardwareGatewayChassis) > 0 {
		hardwareGatewayChassisSet, _ := libovsdb.NewOvsSet(chassis.HardwareGatewayChassis)
		rowChassis[ovnsb.ChassisFieldHardwareGatewayChassis] = hardwareGatewayChassisSet
	}
	if len(chassis.GatewayChassisMembers) > 0 {
		gatewayChassisMembersSet, _ := libovsdb.NewOvsSet(chassis.GatewayChassisMembers)
		rowChassis[ovnsb.ChassisFieldGatewayChassisMembers] = gatewayChassisMembersSet
	}

	insertChassisOp := libovsdb.Operation{
		Op:    odbc.OpInsert,
		Table: ovnsb.Chassis,
		Row:   rowChassis,
	}
	ops = append(ops, insertChassisOp)
	_, err = ovnsb.Transact(ops...)

	return err
}

func chassisDelete(chassis Chassis) error {
	// check Chassis exist
	chassisIndex := ovnsb.ChassisIndex{
		Name: chassis.Name,
	}
	_, err := ovnsb.ChassisGetByIndex(chassisIndex)
	if err != nil {
		return fmt.Errorf("Chassis %s not exist", chassis.Name)
	}

	// when delete Chassis the related Encap can be deleted auto
	// by ovsdb when there is no remaining reference any more.
	// so we don't need to delete Encap manually
	err = ovnsb.ChassisDelByIndex(chassisIndex)
	if err != nil {
		return fmt.Errorf("Chassis %s delete failed", chassis.Name)
	}

	return nil
}

func physicalSwitchUpdateSystemID(newrow libovsdb.Row, oldValue string) error {
	// 1. Get chassis, if not exist, then ignore EncapIP update.
	// no need to store encapIP to SB, when update ps systemID create chassis
	// the EncapIP will also carried in newRow
	systemID, ok := newrow.Fields["system_id"].(string)
	if !ok {
		return fmt.Errorf("Physical Switch system_id get failed")
	}

	if len(oldValue) == 0 {
		// add system_id
		physicalSwitchCreate(newrow)
	} else if len(systemID) == 0 {
		// delete system_id
		chassisName := oldValue
		chassis := Chassis{
			Name: chassisName,
		}
		err := chassisDelete(chassis)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Don't support update Physical Switch's system_id from %s to %s",
			oldValue, systemID)
	}

	return nil
}

func physicalSwitchUpdateEncapIP(newrow libovsdb.Row, oldValue string) error {
	tablePS := vtepdb.ConvertRowToPhysicalSwitch(newrow.Fields)

	// 1. Get chassis, if not exist, then ignore EncapIP update.
	// no need to store encapIP to SB, when update ps systemID create chassis
	// the EncapIP will also carried in newRow
	if len(tablePS.SystemID) == 0 {
		// if the old configured invalid ip, create new chassis
		if net.ParseIP(oldValue) == nil {
			physicalSwitchCreate(newrow)
			return nil
		}

		return fmt.Errorf("Physical Switch not configured system_id, no related chassis")
	}
	chassisName := tablePS.SystemID

	chassisIndex := ovnsb.ChassisIndex{
		Name: chassisName,
	}
	tableChassis, err := ovnsb.ChassisGetByIndex(chassisIndex)
	if err != nil {
		return fmt.Errorf("Chassis %s not exist", chassisName)
	}

	// 2.get SB.Encap and update it's IP
	if len(tableChassis.Encaps) != 1 {
		return fmt.Errorf("Chassis %s related Encap not exist", chassisName)
	}
	encapUUID := tableChassis.Encaps[0]

	ip := tablePS.EncapIP
	if len(ip) == 0 {
		ip = ""
	} else if net.ParseIP(ip) == nil {
		return fmt.Errorf("Invalid encap_ip %s, not update", ip)
	}

	// 2. insert new encap
	encapCfg := ovnsb.TableEncap{
		Type:        ovnsb.EncapTypeVxlan,
		ChassisName: chassisName,
		IP:          ip,
		RouterMac:   tablePS.RouterMac,
		Options:     map[interface{}]interface{}{"csum": "false"},
	}
	err = ovnsb.ChassisUpdateAddEncaps(chassisIndex, encapCfg)
	if err != nil {
		return fmt.Errorf("Add new encap %s failed", encapCfg.IP)
	}

	// 3. delete old encap
	err = ovnsb.ChassisUpdateEncapsDelvalue(chassisIndex, []libovsdb.UUID{encapUUID})
	if err != nil {
		return fmt.Errorf("Remove old encap %v failed", encapUUID.GoUUID)
	}

	return nil
}

func physicalSwitchUpdateRouteMac(newrow libovsdb.Row, oldValue string) error {
	tablePS := vtepdb.ConvertRowToPhysicalSwitch(newrow.Fields)

	// 1. Get chassis, if not exist, then ignore EncapIP update.
	// no need to store encapIP to SB, when update ps systemID create chassis
	// the EncapIP will also carried in newRow
	if len(tablePS.SystemID) == 0 {
		_, err := net.ParseMAC(oldValue)
		if err != nil {
			physicalSwitchCreate(newrow)
			return nil
		}

		return fmt.Errorf("Physical Switch not configured system_id, no related chassis")
	}
	chassisName := tablePS.SystemID

	chassisIndex := ovnsb.ChassisIndex{
		Name: chassisName,
	}
	tableChassis, err := ovnsb.ChassisGetByIndex(chassisIndex)
	if err != nil {
		return fmt.Errorf("Chassis %s not exist", chassisName)
	}

	_, err = net.ParseMAC(tablePS.RouterMac)
	if err != nil {
		return fmt.Errorf("Physical Switch update invalid route mac %s", tablePS.RouterMac)
	}

	// 2.get SB.Encap and update it's route mac
	if len(tableChassis.Encaps) != 1 {
		return fmt.Errorf("Chassis %s related Encap not exist", chassisName)
	}
	encapUUID := tableChassis.Encaps[0]

	// 2. insert new encap
	encapCfg := ovnsb.TableEncap{
		Type:        ovnsb.EncapTypeVxlan,
		ChassisName: chassisName,
		IP:          tablePS.EncapIP,
		RouterMac:   tablePS.RouterMac,
		Options:     map[interface{}]interface{}{"csum": "false"},
	}
	err = ovnsb.ChassisUpdateAddEncaps(chassisIndex, encapCfg)
	if err != nil {
		return fmt.Errorf("Add new encap %s failed", encapCfg.IP)
	}

	// 3. delete old encap
	err = ovnsb.ChassisUpdateEncapsDelvalue(chassisIndex, []libovsdb.UUID{encapUUID})
	if err != nil {
		return fmt.Errorf("Remove old encap %v failed", encapUUID.GoUUID)
	}

	return nil
}
