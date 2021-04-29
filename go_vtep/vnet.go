package govtep

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnnb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnnorthbound"
	ovnsb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnsouthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// Datapath type string
const (
	DatapathTypeLS = "logical-switch"
	DatapathTypeLR = "logical-router"

	AutoGatewayConfDefaultPhysicalPort string = "ep60"
)

// Vnet type string
const (
	VnetTypeBD  = "bd"
	VnetTypeVRF = "vrf"
)

func datapathNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate, dpuuid string) {
	if false == GatewayInitDone {
		log.Info("Gateway not init yet")
		return
	}

	switch op {
	case odbc.OpInsert:
		vnetCreate(rowUpdate.New, dpuuid)
	case odbc.OpDelete:
		vnetRemove(rowUpdate.Old)
	case odbc.OpUpdate:
		vnetUpdate(rowUpdate.New, rowUpdate.Old)
	}
}

func vnetProcessAll() {
	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))

	rows, num := ovnsb.DatapathBindingGet(conditions)
	if num > 0 {
		for _, row := range rows {
			if UUID, ok := row["_uuid"].(libovsdb.UUID); ok {
				rowUpdate := libovsdb.Row{Fields: row}
				vnetCreate(rowUpdate, UUID.GoUUID)
			}
		}
	}

	rows, num = ovnsb.PortBindingGet(conditions)
	if num > 0 {
		for _, row := range rows {
			if UUID, ok := row["_uuid"].(libovsdb.UUID); ok {
				rowUpdate := libovsdb.Row{Fields: row}
				portbindingCreate(rowUpdate, UUID.GoUUID)
			}
		}
	}

	rows, num = ovnnb.LogicalRouterGet(conditions)
	if num > 0 {
		for _, row := range rows {
			if UUID, ok := row["_uuid"].(libovsdb.UUID); ok {
				rowUpdate := libovsdb.Row{Fields: row}
				logicalRouterCreate(rowUpdate, UUID.GoUUID)
			}
		}
	}
}

// when local switch not longer being DC gateway remove all vnet
func vnetRemoveAll() {
	// remove bd
	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))
	rows, num := vtepdb.BridgeDomainGet(conditions)
	if num > 0 {
		for _, row := range rows {
			if UUID, ok := row["_uuid"].(libovsdb.UUID); ok {
				vtepdb.BridgeDomainDelByUUID(UUID.GoUUID)
			}
		}
	}

	// remove vrf
	rows, num = vtepdb.VrfGet(conditions)
	if num > 0 {
		for _, row := range rows {
			if UUID, ok := row["_uuid"].(libovsdb.UUID); ok {
				vtepdb.VrfDelByUUID(UUID.GoUUID)
			}
		}
	}

	// remove locators
	rows, num = vtepdb.LocatorGet(conditions)
	if num > 0 {
		for _, row := range rows {
			if UUID, ok := row["_uuid"].(libovsdb.UUID); ok {
				vtepdb.LocatorDelByUUID(UUID.GoUUID)
			}
		}
	}
}

func vnetCreate(row libovsdb.Row, dpuuid string) {
	tableDatapathBinding := ovnsb.ConvertRowToDatapathBinding(libovsdb.ResultRow(row.Fields))
	tableDatapathBinding.UUID = dpuuid

	if tableDatapathBinding.ExternalIds[DatapathTypeLS] != nil {
		bridgeDomainAdd(tableDatapathBinding)
	} else if tableDatapathBinding.ExternalIds[DatapathTypeLR] != nil {
		vrfAdd(tableDatapathBinding)
	}
}

func vnetRemove(row libovsdb.Row) {
	tableDatapathBinding := ovnsb.ConvertRowToDatapathBinding(libovsdb.ResultRow(row.Fields))
	vni := tableDatapathBinding.TunnelKey

	if tableDatapathBinding.ExternalIds[DatapathTypeLS] != nil {
		bridgeDomainDel(vni)
	} else if tableDatapathBinding.ExternalIds[DatapathTypeLR] != nil {
		vrfDel(vni)
	}
}

func vnetUpdate(newrow libovsdb.Row, oldrow libovsdb.Row) {

}

func bridgeDomainAdd(tableDp ovnsb.TableDatapathBinding) error {
	bdIndex := vtepdb.BridgeDomainIndex{
		Name: getBdNameByVni(tableDp.TunnelKey),
	}
	_, err := vtepdb.BridgeDomainGetByIndex(bdIndex)
	if err == nil {
		log.Info("BridgeDomain %s already exist", bdIndex.Name)
		return nil
	}

	tableBridgeDomain := vtepdb.TableBridgeDomain{
		L2vni:    tableDp.TunnelKey,
		Name:     getBdNameByVni(tableDp.TunnelKey),
		Datapath: tableDp.UUID,
		Lsname:   tableDp.ExternalIds[DatapathTypeLS].(string),
	}
	_, err = vtepdb.BridgeDomainAdd(tableBridgeDomain)
	if err != nil {
		log.Error("BridgeDomainAdd %s failed : %v", tableBridgeDomain.Name, err)
	}
	return err
}

func bridgeDomainDel(vni int) error {
	bdIndex := vtepdb.BridgeDomainIndex{
		Name: getBdNameByVni(vni),
	}
	err := vtepdb.BridgeDomainDelByIndex(bdIndex)
	if err != nil {
		log.Error("BridgeDomainDel %s failed : %v", bdIndex.Name, err)
	}
	return err
}

func vrfAdd(tableDp ovnsb.TableDatapathBinding) error {
	vrfIndex := vtepdb.VrfIndex{
		Name: getVrfNameByVni(tableDp.TunnelKey),
	}
	_, err := vtepdb.VrfGetByIndex(vrfIndex)
	if err == nil {
		log.Info("Vrf %s already exist", vrfIndex.Name)
		return nil
	}

	tableVrf := vtepdb.TableVrf{
		L3vni:    tableDp.TunnelKey,
		Name:     getVrfNameByVni(tableDp.TunnelKey),
		Datapath: tableDp.UUID,
		Lrname:   tableDp.ExternalIds[DatapathTypeLR].(string),
	}
	_, err = vtepdb.VrfAdd(tableVrf)
	if err != nil {
		log.Error("VrfAdd %s failed : %v", tableVrf.Name, err)
	}

	tableAutoGatewayConf := vtepdb.TableAutoGatewayConf{
		PhysicalPort: AutoGatewayConfDefaultPhysicalPort,
		Vrf:          tableVrf.Name,
	}

	err = vtepdb.VrfUpdateAddGatewayConf(vrfIndex, tableAutoGatewayConf)
	if err != nil {
		log.Error("AutoGAtewayConfAdd %s failed : %v.\n", tableVrf.Name, err)
	}
	return err
}

func vrfDel(vni int) error {
	vrfIndex := vtepdb.VrfIndex{
		Name: getVrfNameByVni(vni),
	}
	err := vtepdb.VrfDelByIndex(vrfIndex)
	if err != nil {
		log.Error("VrfDel %s failed : %v", vrfIndex.Name, err)
	}
	return err
}
