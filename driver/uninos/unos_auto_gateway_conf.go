package driver

import (
	"strconv"
	"strings"

	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"

	"github.com/ebay/libovsdb"
)

type autoGatewayConfAPI struct {
	moduleID int
}

var autoGatewayConfAPIs = autoGatewayConfAPI{
	moduleID: tai.ObjectIDAutoGatewayConf,
}

// AutoGatewayConfRelevantTable definition
type AutoGatewayConfRelevantTable struct {
	Vlan         int
	IP           string
	Bdname       string
	PhysicalPort string
	Vrf          string
}

func addAutoGatewayconfRelevantTable(table AutoGatewayConfRelevantTable) error {
	log.Info("Add auto gateway relevant table %+v.\n", table)
	subportIndex := cdb.SubPortIndex{
		Name: table.PhysicalPort + "." + strconv.Itoa(table.Vlan),
	}
	_, errExist := cdb.SubPortGetByIndex(subportIndex)
	if errExist != nil {
		tableSubport := cdb.TableSubPort{
			Name: table.PhysicalPort + "." + strconv.Itoa(table.Vlan),
			Vlan: []string{strconv.Itoa(table.Vlan)},
		}

		portIndex := cdb.PortIndex{
			Name: table.PhysicalPort,
		}

		err := cdb.PortUpdateAddSubport(portIndex, tableSubport)
		if err != nil {
			log.Error("Create subport v% failed.\n", tableSubport.Name)
			return err
		}
	}

	vrfIndex := cdb.VrfIndex{
		Name: table.Vrf,
	}

	tableVrf, errExist := cdb.VrfGetByIndex(vrfIndex)
	if errExist != nil {
		log.Error("Get vrf table failed.\n")
		return errExist
	}

	ifIndex := cdb.InterfaceIndex{
		Name: table.PhysicalPort + "." + strconv.Itoa(table.Vlan),
		Type: cdb.InterfaceTypeSubPort,
	}

	tableIF, errExist := cdb.InterfaceGetByIndex(ifIndex)
	if errExist == nil {
		log.Info("Interface for %s already exist\n", ifIndex.Name)

		err := cdb.InterfaceUpdateIPAddvalue(ifIndex, []string{table.IP})
		if err != nil {
			log.Error("Update interface: %v ip: %v failed.\n", ifIndex.Name, table.IP)
			return err
		}

		if len(tableIF.Vrf) == 0 {
			err = cdb.InterfaceUpdateVrfAddvalue(ifIndex, []libovsdb.UUID{{GoUUID: tableVrf.UUID}})
			if err != nil {
				log.Error("Update interface: %v vrf: %+v failed.\n", ifIndex.Name, libovsdb.UUID{GoUUID: tableVrf.UUID})
				return err
			}
		}

	} else {
		interfaceCfg := cdb.TableInterface{
			Name:        table.PhysicalPort + "." + strconv.Itoa(table.Vlan),
			Type:        cdb.InterfaceTypeSubPort,
			AdminStatus: []string{cdb.InterfaceDefaultAdminStatus},
			ProxyArp:    []string{cdb.InterfaceProxyArpDisable},
			IP:          []string{table.IP},
			Mtu:         []int{cdb.InterfaceDefaultMtu},
			Vrf:         []libovsdb.UUID{{GoUUID: tableVrf.UUID}},
			SwitchPort:  []string{cdb.InterfaceSwitchPortDisable},
		}

		_, err := cdb.InterfaceAdd(interfaceCfg)
		if err != nil {
			log.Error("Create interface %v failed.\n", interfaceCfg.Name)
			return err
		}
	}
	return nil
}

func delAutoGatewayconfRelevantTableByVrf(vrf string) error {
	vrfIndex := cdb.VrfIndex{
		Name: vrf,
	}

	tableVrf, errExist := cdb.VrfGetByIndex(vrfIndex)
	if errExist != nil {
		log.Warning("Get vrf %v failed.\n", vrf)
		return nil
	}

	var conditions []interface{}
	conditions = append(conditions, libovsdb.NewCondition(cdb.InterfaceFieldType, "==", cdb.InterfaceTypeSubPort))
	conditions = append(conditions, libovsdb.NewCondition(cdb.InterfaceFieldVrf, "==", libovsdb.UUID{GoUUID: tableVrf.UUID}))

	row, num := cdb.InterfaceGet(conditions)
	if num == 0 {
		log.Info("Interface with Type : %v, Vrf : %v don't existed.\n", cdb.InterfaceTypeSubPort, vrf)
		return nil
	} else {
		ifIndex := cdb.InterfaceIndex{
			Name: cdb.ConvertRowToInterface(row[0]).Name,
			Type: cdb.InterfaceTypeSubPort,
		}

		_, errExist := cdb.InterfaceGetByIndex(ifIndex)
		if errExist == nil {
			err := cdb.InterfaceDelByIndex(ifIndex)
			if err != nil {
				log.Error("Delete interface %v failed.\n", ifIndex.Name)
				return err
			}
		}

		subportIndex := cdb.SubPortIndex{
			Name: ifIndex.Name,
		}
		tableSubPort, errExist := cdb.SubPortGetByIndex(subportIndex)
		if errExist == nil {
			portIndex := cdb.PortIndex{
				Name: ifIndex.Name[:strings.Index(ifIndex.Name, ".")],
			}
			err := cdb.PortUpdateSubportDelvalue(portIndex, []libovsdb.UUID{{GoUUID: tableSubPort.UUID}})
			if err != nil {
				log.Error("Delete subport %v failed.\n", portIndex.Name)
				return err
			}
		}
	}
	return nil
}

func (v autoGatewayConfAPI) CreateObject(obj interface{}) error {
	log.Info("Create object %+v.\n", obj)
	objAutoGatewayConf := obj.(tai.AutoGatewayConfObj)

	if strings.Contains(objAutoGatewayConf.Bdname, "Bd") && strings.Contains(objAutoGatewayConf.PhysicalPort, "ep") {
		err := delAutoGatewayconfRelevantTableByVrf(objAutoGatewayConf.Vrf)
		if err != nil {
			log.Error("Del autoGatewayconf relevant table by Vrf failed.\n")
			return err
		}
		autoGatewayConfRelevantTable := AutoGatewayConfRelevantTable{
			Vlan:         objAutoGatewayConf.Vlan,
			IP:           objAutoGatewayConf.IP,
			Bdname:       objAutoGatewayConf.Bdname,
			PhysicalPort: objAutoGatewayConf.PhysicalPort,
			Vrf:          objAutoGatewayConf.Vrf,
		}

		err = addAutoGatewayconfRelevantTable(autoGatewayConfRelevantTable)
		if err != nil {
			log.Error("Add auto gateway conf relevant table failed.\n")
			return err
		}
	}

	pbrUpdateGateway(objAutoGatewayConf.Vrf, objAutoGatewayConf.PhysicalPort, objAutoGatewayConf.Vlan)

	return nil
}

func (v autoGatewayConfAPI) RemoveObject(obj interface{}) error {
	log.Info("Remove object %+v.\n", obj)
	objAutoGatewayConf := obj.(tai.AutoGatewayConfObj)
	err := delAutoGatewayconfRelevantTableByVrf(objAutoGatewayConf.Vrf)
	if err != nil {
		log.Error("Del autoGatewayconf relevant table by Vrf failed.\n")
	}
	return err
}
func (v autoGatewayConfAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	log.Info("Set object : %v attr %+v.\n", obj, attrs)
	objAutoGatewayConf := obj.(tai.AutoGatewayConfObj)
	if strings.Contains(objAutoGatewayConf.Bdname, "Bd") && strings.Contains(objAutoGatewayConf.PhysicalPort, "ep") {
		err := delAutoGatewayconfRelevantTableByVrf(objAutoGatewayConf.Vrf)
		if err != nil {
			log.Error("Del autoGatewayconf relevant table by Vrf failed.\n")
			return err
		}

		autoGatewayConfRelevantTable := AutoGatewayConfRelevantTable{
			Vlan:         objAutoGatewayConf.Vlan,
			IP:           objAutoGatewayConf.IP,
			Bdname:       objAutoGatewayConf.Bdname,
			PhysicalPort: objAutoGatewayConf.PhysicalPort,
			Vrf:          objAutoGatewayConf.Vrf,
		}

		err = addAutoGatewayconfRelevantTable(autoGatewayConfRelevantTable)
		if err != nil {
			log.Error("Add auto gateway conf relevant table failed.\n")
			return err
		}
	}

	pbrUpdateGateway(objAutoGatewayConf.Vrf, objAutoGatewayConf.PhysicalPort, objAutoGatewayConf.Vlan)

	return nil
}

func (v autoGatewayConfAPI) DelObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	log.Info("Set object : %v attr %+v.\n", obj, attrs)
	objAutoGatewayConf := obj.(tai.AutoGatewayConfObj)

	if !(strings.Contains(objAutoGatewayConf.Bdname, "Bd") && strings.Contains(objAutoGatewayConf.PhysicalPort, "ep")) {
		err := delAutoGatewayconfRelevantTableByVrf(objAutoGatewayConf.Vrf)
		if err != nil {
			log.Error("Del autoGatewayconf relevant table by Vrf failed.\n")
			return err
		}
	}
	return nil
}

func (v autoGatewayConfAPI) SetObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	log.Info("Set object : %v attr %+v.\n", obj, attrs)
	objAutoGatewayConf := obj.(tai.AutoGatewayConfObj)
	err := delAutoGatewayconfRelevantTableByVrf(objAutoGatewayConf.Vrf)
	if err != nil {
		log.Error("Del autoGatewayconf relevant table by Vrf failed.\n")
		return err
	}

	if strings.Contains(objAutoGatewayConf.Bdname, "Bd") && strings.Contains(objAutoGatewayConf.PhysicalPort, "ep") {
		autoGatewayConfRelevantTable := AutoGatewayConfRelevantTable{
			Vlan:         objAutoGatewayConf.Vlan,
			IP:           objAutoGatewayConf.IP,
			Bdname:       objAutoGatewayConf.Bdname,
			PhysicalPort: objAutoGatewayConf.PhysicalPort,
			Vrf:          objAutoGatewayConf.Vrf,
		}

		err = addAutoGatewayconfRelevantTable(autoGatewayConfRelevantTable)
		if err != nil {
			log.Error("Add auto gateway conf relevant table failed.\n")
			return err
		}
	}

	pbrUpdateGateway(objAutoGatewayConf.Vrf, objAutoGatewayConf.PhysicalPort, objAutoGatewayConf.Vlan)

	return nil
}

func (v autoGatewayConfAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v autoGatewayConfAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
