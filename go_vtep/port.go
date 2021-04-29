package govtep

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnnb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnnorthbound"
	ovnsb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnsouthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// Logical port type
const (
	PortLtypeLSP = "lsp"
	PortLtypeLRP = "lrp"
)

// PortInfo layer
const (
	PortPtypeL2 = "l2port"
	PortPtypeL3 = "l3port"
)

// Location string
const (
	LocationUnknown = "unknown"
	LocationLocal   = "local"
	LocationRemote  = "remote"
)

// Port binding vtep options
const (
	PbOptionPhysicalSwitch      = "physical_switch"
	PbOptionPhysicalSwitchGroup = "physical_switch_group"
	PbOptionPhysicalParentPort  = "physical_parent_port"
)

// Logical port type with location
const (
	_ = iota
	LSPACLocal
	LSPACRemote
	LSPPatchLSP
	LSPPatchLRP
	LSPLocalNET
	LRPACLocal
	LRPACRemote
	LRPPatchLSP
	LRPPatchLRP
)

// PortInfo attributes
type PortInfo struct {
	LogicalPort       string // SB.Port_Binding.logical_port
	LnName            string // NB.Logical_Switch._uuid
	LnType            string // logical_switch or logical_router
	VnetTunnelKey     int    // SB.Datapath_Binding.tunnel_key
	LogicalParentPort string
	VlanTag           int
	Ipv4addr          []string // SB.Port_Binding.mac
	Ipv6addr          []string // SB.Port_Binding.mac
	Mac               []string // SB.Port_Binding.mac
	Nat               []string // SB.Port_Binding.nat_addresses
	Type              string   // logical port type, SB.Port_Binding.type, eg: "patch","localnet","vtep","localport"...
	Peer              string   // patch peer,logical port name
	PeerLtype         string
	PeerPort          string
	PeerPtype         string
	PeerBd            string
	PeerVrf           string
	DpUUID            string
	PortTunnelKey     int
	Chassis           string
	PhySwitch         string
	Name              string
	PhyParentPort     string
	Bd                string
	Irb               string
	Vrf               string
	Location          string
	Locator           string
	FailureReason     string
}

func isBelong(ip, cidr string) bool {
	ipAddr := strings.Split(ip, `.`)
	if len(ipAddr) < 4 {
		return false
	}
	cidrArr := strings.Split(cidr, `/`)
	if len(cidrArr) < 2 {
		return false
	}
	addrInt1 := strconv.FormatInt(ipAddrToInt(ip), 2)
	addrInt2 := strconv.FormatInt(ipAddrToInt(cidrArr[0]), 2)
	mask, _ := strconv.Atoi(cidrArr[1])
	for i := 0; i < len(addrInt1)+mask-32; i++ {
		if addrInt1[i] != addrInt2[i] {
			return false
		}
	}
	return true
}

func ipAddrToInt(ipAddr string) int64 {
	bits := strings.Split(ipAddr, ".")
	b0, _ := strconv.Atoi(bits[0])
	b1, _ := strconv.Atoi(bits[1])
	b2, _ := strconv.Atoi(bits[2])
	b3, _ := strconv.Atoi(bits[3])
	var sum int64
	sum += int64(b0) << 24
	sum += int64(b1) << 16
	sum += int64(b2) << 8
	sum += int64(b3)

	return sum
}

var portFailureChains map[interface{}]portFailureChain

type portFailureChain struct {
	fc map[interface{}]PortInfo
}

// PortOpts additional port options
type PortOpts struct {
	Peer         string
	NatAddresses []string
	QosMaxRate   string //VMI (or VIF) Options:
	QosBurst     string
	QdiscQueueID string
}

var portType = make(map[string]int)
var portInfoMap = make(map[string]PortInfo)

func getPortFromFailureChain(chain int, k string) (PortInfo, error) {
	if p, ok := portFailureChains[chain].fc[k]; ok {
		return p, nil
	}
	// blank PortInfo and error not found
	return PortInfo{}, errors.New("Not found")
}

func portAddToFailureChain(chain int, port PortInfo) error {
	_, err := getPortFromFailureChain(chain, port.LogicalPort)
	if err == nil {
		return errors.New("Already exist")
	}
	// TODO nil map should make before entry insert
	//portFailureChains[chain].fc[port.LogicalPort] = port

	return nil
}

func portReplaceToFailureChain(chain int, port PortInfo) error {
	portRemoveFromFailureChain(chain, port.LogicalPort)
	portAddToFailureChain(chain, port)
	return nil
}

func portRemoveFromFailureChain(chain int, k string) {
	delete(portFailureChains[chain].fc, k)
}

func portProcFailureChain(chain int) {

}

func portbindingNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate, pbUUID string) {
	if false == GatewayInitDone {
		log.Info("Gateway not init yet")
		return
	}

	switch op {
	case odbc.OpInsert:
		portbindingCreate(rowUpdate.New, pbUUID)
	case odbc.OpDelete:
		portbindingRemove(rowUpdate.Old, pbUUID)
	case odbc.OpUpdate:
		portbindingUpdate(rowUpdate.New, rowUpdate.Old, pbUUID)
	}
}

func definePortProcBranch(port PortInfo) int {
	if port.LnType == DatapathTypeLS {
		if port.Type == "" {
			if port.Location == LocationLocal {
				return LSPACLocal
			}

			if port.Location == LocationRemote {
				return LSPACRemote
			}
		}

		if port.Type == "patch" {
			if port.PeerLtype == PortLtypeLSP {
				return LSPPatchLSP
			}

			if port.PeerLtype == PortLtypeLRP {
				return LSPPatchLRP
			}
		}

		if port.Type == "localnet" {
			return LSPLocalNET
		}
	}

	if port.LnType == DatapathTypeLR {
		//if port.Type == "" {
		if port.Location == LocationLocal {
			return LRPACLocal
		}

		if port.Location == LocationRemote {
			return LRPACRemote
		}
		//}

		if port.Type == "patch" {
			if port.PeerLtype == PortLtypeLSP {
				return LRPPatchLSP
			}

			if port.PeerLtype == PortLtypeLRP {
				return LRPPatchLRP
			}
		}
	}

	return 0
}

// PortbindingParser OVNSB logical port binding table row TO PortInfo struct
func PortbindingParser(row libovsdb.Row) PortInfo {
	var (
		logicalPort, datapath            string
		lnType, lnName, pType            string
		vnetTunnelKey, portTunnelKey     int
		vlantag                          int
		peer, peerLtype, peerPort        string
		peerPtype, peerBd, peerVrf       string
		location                         = LocationUnknown
		chassis, phyParentPort, name     string
		bd, vrf                          string
		macaddr, ipv4addr, ipv6addr, nat []string
		irb                              string
		locator, logicalParentPort       string
	)

	tablePortBinding := ovnsb.ConvertRowToPortBinding(libovsdb.ResultRow(row.Fields))
	log.Info("tablePortBinding %+v\n", tablePortBinding)

	logicalPort = tablePortBinding.LogicalPort
	datapath = tablePortBinding.Datapath.GoUUID
	lnType, lnName, vnetTunnelKey = getLnByDpuuid(datapath)
	if lnType == DatapathTypeLS {
		bd = getBdNameByVni(vnetTunnelKey)
	} else if lnType == DatapathTypeLR {
		vrf = getVrfNameByVni(vnetTunnelKey)
	}

	macaddr, ipv4addr, ipv6addr = addressParser(tablePortBinding.Mac)
	nat = tablePortBinding.NatAddresses
	if len(tablePortBinding.ParentPort) == 1 {
		logicalParentPort = tablePortBinding.ParentPort[0]
	}

	pType = tablePortBinding.Type
	portTunnelKey = tablePortBinding.TunnelKey

	// TODO there are many more port-type specific option to be parsered
	options := tablePortBinding.Options

	// parser physical_switch and phyParentPort from sb.PortBinding.Options
	phySwitch, _ := options[PbOptionPhysicalSwitch].(string)
	phySwitchGroup, _ := options[PbOptionPhysicalSwitchGroup].(string)
	phyParentPort, _ = options[PbOptionPhysicalParentPort].(string)

	if len(tablePortBinding.Chassis) == 1 {
		chassis = tablePortBinding.Chassis[0].GoUUID

		tableChassis, err := ovnsb.ChassisGetByUUID(chassis)
		if err == nil {
			location = getLocation(tableChassis.Name, phySwitch, phySwitchGroup)
			locator, _ = getLocatorUUID(tableChassis.Name)

			if len(tableChassis.HardwareGatewayChassis) > 0 {
				if tableChassis.HardwareGatewayChassis[0] == ovnsb.ChassisHardwareGatewayChassisPhsicalSwitch ||
					tableChassis.HardwareGatewayChassis[0] == ovnsb.ChassisHardwareGatewayChassisPhsicalSwitchGroup {
					location = LocationLocal
				}
			}
		}
	}

	// do phsical switch chassis binding port
	if phySwitch != "" && len(tablePortBinding.Chassis) == 0 {
		phsicalSwitchIndex := vtepdb.PhysicalSwitchIndex{
			Name: phySwitch,
		}
		tablePS, err := vtepdb.PhysicalSwitchGetByIndex(phsicalSwitchIndex)
		if err != nil {
			log.Warning("Port-binding option physical-switch %s not exist", phsicalSwitchIndex.Name)
		} else {
			chassisIndex := ovnsb.ChassisIndex{
				Name: tablePS.SystemID,
			}
			tableChassis, err := ovnsb.ChassisGetByIndex(chassisIndex)
			if err != nil {
				log.Warning("Get chassis %v failed", chassisIndex.Name)
			} else {
				// Set chassis to port binding
				pbIndex := ovnsb.PortBindingIndex1{
					LogicalPort: logicalPort,
				}
				err = ovnsb.PortBindingSetField(pbIndex, ovnsb.
					PortBindingFieldChassis, libovsdb.UUID{GoUUID: tableChassis.UUID})
				if err != nil {
					log.Warning("PortBinding update Chassis failed")
				}
			}
		}
	}

	// Only support one tag in ovn for now
	if len(tablePortBinding.Tag) == 1 {
		vlantag = tablePortBinding.Tag[0]
	}

	if (pType == "") && (location == LocationLocal) {
		name = getSubportName(phyParentPort, []int{vlantag})
	}

	if peer, ok := options["peer"].(string); ok {
		peerLtype = getPortLtype(peer)
		// TODO:when peer port not created in VTEPDB, can't get
		peerPort = getPortByLp(peer)
		peerPtype = getPortPtype(peer)
		_v, _vt := lpBelongWhichVnet(peer)
		if _vt == VnetTypeBD {
			peerBd = _v
		} else if _vt == VnetTypeVRF {
			peerVrf = _v
		}

		if pType == "patch" {
			if (lnType == DatapathTypeLR) && (peerLtype == PortLtypeLSP) {
				name = getBdifportName(peerBd)
			} else {
				name = getVethportName(vnetTunnelKey, portTunnelKey)
			}
		}
	}

	if (lnType == DatapathTypeLS) && (pType == "") {
		irb = getBdifportName(bd)
	}

	port := PortInfo{
		LogicalPort:       logicalPort,
		LnName:            lnName,
		LnType:            lnType,
		VnetTunnelKey:     vnetTunnelKey,
		LogicalParentPort: logicalParentPort,
		VlanTag:           vlantag,
		Ipv4addr:          ipv4addr,
		Ipv6addr:          ipv6addr,
		Mac:               macaddr,
		Nat:               nat,
		Type:              pType,
		Peer:              peer,
		PeerLtype:         peerLtype,
		PeerPort:          peerPort,
		PeerPtype:         peerPtype,
		PeerBd:            peerBd,
		PeerVrf:           peerVrf,
		DpUUID:            datapath,
		PortTunnelKey:     portTunnelKey,
		Chassis:           chassis,
		PhySwitch:         phySwitch,
		Name:              name,
		PhyParentPort:     phyParentPort,
		Bd:                bd,
		Irb:               irb,
		Vrf:               vrf,
		Location:          location,
		Locator:           locator,
	}
	return port
}

func portbindLocalNET(port PortInfo) error {
	var conditions []interface{}
	conditions = append(conditions, libovsdb.NewCondition("local_locator", "==", true))
	row, _ := vtepdb.LocatorGet(conditions)
	if len(row) == 0 {
		log.Error("Get the value of local_locator failed.\n")
		return fmt.Errorf("Get the value of local_locator failed")
	}
	dbLocator := vtepdb.ConvertRowToLocator(row[0])

	chassisIndex := ovnsb.ChassisIndex{
		Name: dbLocator.ChassisName,
	}
	tableChassis, err := ovnsb.ChassisGetByIndex(chassisIndex)
	if err != nil {
		log.Error("Get chassis %v table failed.\n", dbLocator.ChassisName)
		return err
	}
	// Set chassis to port binding
	pbIndex := ovnsb.PortBindingIndex1{
		LogicalPort: port.LogicalPort,
	}

	err = ovnsb.PortBindingSetField(pbIndex, ovnsb.PortBindingFieldChassis, libovsdb.UUID{GoUUID: tableChassis.UUID})
	if err != nil {
		log.Error("PortBindingTableUpdate %s failed : %v.\n", pbIndex.LogicalPort, err)
		return err
	}

	err = addBdToAutoGatewayConf(port)
	if err != nil {
		log.Error("Add autogatewayconf table failed.\n")
	}
	return err
}

func clearAutoGatewayConfBdValue(port PortInfo) error {
	var conditions []interface{}
	conditions = append(conditions, libovsdb.NewCondition(vtepdb.AutoGatewayConfFieldBdname, "==", port.Bd))
	conditions = append(conditions, libovsdb.NewCondition(vtepdb.AutoGatewayConfFieldVlan, "==", port.VlanTag))
	row, num := vtepdb.AutoGatewayConfGet(conditions)
	if num == 0 {
		return nil
	}

	log.Warning("Clear autogatewayconf'Bd value.\n")

	tableAGC := vtepdb.ConvertRowToAutoGatewayConf(row[0])

	agcIndex := vtepdb.AutoGatewayConfIndex{
		Vrf: tableAGC.Vrf,
	}

	tableAutoGatewayConf := vtepdb.TableAutoGatewayConf{
		UUID:         tableAGC.UUID,
		PhysicalPort: tableAGC.PhysicalPort,
		Vrf:          tableAGC.Vrf,
	}

	err := vtepdb.AutoGatewayConfSet(agcIndex, tableAutoGatewayConf)
	if err != nil {
		log.Error("Unbind localnet port failed.\n")
	}
	return err
}

func portunbindLocalNET(port PortInfo) error {
	log.Warning("Unbind port with type: LocalNET.\n")
	pbIndex := ovnsb.PortBindingIndex1{
		LogicalPort: port.LogicalPort,
	}
	err := ovnsb.PortBindingUpdateChassisDelvalue(pbIndex, []libovsdb.UUID{{GoUUID: port.Chassis}})
	if err != nil {
		log.Error("Localnet port unbind failed.\n")
		return err
	}

	err = clearAutoGatewayConfBdValue(port)
	if err != nil {
		log.Error("Clear autogatewayconf'Bd value failed.\n")
	}

	return err
}

func getexternalVrf(port PortInfo) (string, error) {
	var conditions []interface{}
	conditions = append(conditions, libovsdb.NewCondition(ovnsb.PortBindingFieldDatapath, "==", libovsdb.UUID{GoUUID: port.DpUUID}))
	conditions = append(conditions, libovsdb.NewCondition(ovnsb.PortBindingFieldType, "==", "patch"))
	row, num := ovnsb.PortBindingGet(conditions)
	if num == 0 {
		log.Warning("Get port-binding table failed.\n")
		return "", fmt.Errorf("Get port-binding table failed")
	}
	dbPortBinding := ovnsb.ConvertRowToPortBinding(row[0])
	lp, lpExisted := dbPortBinding.Options["peer"]
	if lpExisted {
		pbIndex1 := ovnsb.PortBindingIndex1{
			LogicalPort: lp.(string),
		}
		pbTable, err := ovnsb.PortBindingGetByIndex(pbIndex1)
		if err != nil {
			log.Warning("Get port-binding table failed.\n")
			return "", err
		}

		pdbTabel, err := ovnsb.DatapathBindingGetByUUID(pbTable.Datapath.GoUUID)
		if err != nil {
			log.Warning("Get datapath table failed.\n")
			return "", err
		}

		Vrfname := fmt.Sprintf("%s%d", "Vrf", pdbTabel.TunnelKey)
		return Vrfname, nil
	}
	log.Warning("Peer port did't existed.\n")
	return "", fmt.Errorf("Peer port did't existed")
}

func addBdToAutoGatewayConf(port PortInfo) error {
	lsIndex := ovnnb.LogicalSwitchUUIDIndex{
		UUID: port.LnName,
	}
	tableLs, err := ovnnb.LogicalSwitchGetByIndex(lsIndex)
	if err != nil {
		log.Error("Get logical switch table failed.\n")
		return err
	}

	for _, portUUID := range tableLs.Ports {
		tableLsp, err := ovnnb.LogicalSwitchPortGetByUUID(portUUID.GoUUID)
		if err != nil {
			log.Error("Get logical switch port table failed.\n")
			return err
		}
		_, cidrIsExisted := tableLsp.ExternalIds["neutron:cidrs"]
		if cidrIsExisted {
			ipaddress := (tableLsp.ExternalIds["neutron:cidrs"]).(string)
			if isBelong(port.Ipv4addr[0], ipaddress) {
				Vrfname, errVrf := getexternalVrf(port)
				if errVrf != nil {
					log.Warning("Vrf did't exist.\n")
					return nil
				}

				agcIndex := vtepdb.AutoGatewayConfIndex{
					Vrf: Vrfname,
				}

				tableAGC, err := vtepdb.AutoGatewayConfGetByIndex(agcIndex)
				if err != nil {
					log.Error("Get autogatewayconf table by vrf %v failed.\n", Vrfname)
					return err
				}

				tableAGC.Bdname = port.Bd
				tableAGC.Vlan = port.VlanTag
				tableAGC.IP = ipaddress

				errSet := vtepdb.AutoGatewayConfSet(agcIndex, tableAGC)
				if errSet != nil {
					log.Error("Set auto gateway conf table failed.\n")
				}
				return errSet
			}
		}
	}
	return nil
}

func autoGatewayConfTableUpdate(port PortInfo) {
	if strings.Contains(port.LogicalPort, "provnet") && port.VlanTag != 0 {
		if port.Mac[0] != "unknown" && len(port.Chassis) > 0 {
			Vrfname, errVrf := getexternalVrf(port)
			if errVrf != nil {
				log.Warning("Vrf did't exist.\n")
				return
			}

			agcIndex := vtepdb.AutoGatewayConfIndex{
				Vrf: Vrfname,
			}

			tableAGC, err := vtepdb.AutoGatewayConfGetByIndex(agcIndex)
			if err != nil {
				log.Error("Get auto gateway conf table failed.\n")
				return
			}

			if tableAGC.Bdname == port.Bd && tableAGC.Vlan == port.VlanTag {
				log.Info("Auto gateway conf table existed.\n")
				return
			}

			err = addBdToAutoGatewayConf(port)
			if err != nil {
				log.Error("Add auto gateway conf table failed.\n")
				return
			}
		}
	}
	return
}

func portbindingUpdateLocalNET(port PortInfo) error {
	if strings.Contains(port.LogicalPort, "provnet") && port.VlanTag != 0 {
		if port.Mac[0] != "unknown" && len(port.Chassis) == 0 {
			err := portbindLocalNET(port)
			if err != nil {
				log.Error("Port localnet binding failed.\n")
				return err
			}
		}

		if port.Mac[0] == "unknown" && len(port.Chassis) > 0 {
			err := portunbindLocalNET(port)
			if err != nil {
				log.Error("Port localnet unbinding failed.\n")
				return err
			}
		}
	}
	return nil
}

func portGenRfdbSet(port PortInfo) []RemoteFdb {
	var fdbs []RemoteFdb
	for _, mac := range port.Mac {
		fdb := RemoteFdb{
			UUID:          "",
			Bridge:        port.Bd,
			Mac:           mac,
			RemoteLocator: port.Locator,
		}
		log.Info("Generate remote fdb %v\n", fdb)
		fdbs = append(fdbs, fdb)
	}
	return fdbs
}

func portGenRneighSet(port PortInfo) []RemoteNeigh {
	var nhs []RemoteNeigh
	for _, ipv4addr := range port.Ipv4addr {
		neigh := RemoteNeigh{
			UUID:          "",
			OutL3Port:     port.Irb,
			Ipaddr:        ipv4addr,
			Mac:           port.Mac[0],
			RemoteLocator: port.Locator,
		}
		log.Info("Generate remote neigh %+v\n", neigh)
		nhs = append(nhs, neigh)
	}
	for _, ipv6addr := range port.Ipv4addr {
		neigh := RemoteNeigh{
			UUID:          "",
			OutL3Port:     port.Irb,
			Ipaddr:        ipv6addr,
			Mac:           port.Mac[0],
			RemoteLocator: port.Locator,
		}
		nhs = append(nhs, neigh)
	}
	return nhs
}

func portGenRouteSet(port PortInfo) []Route {
	var rts []Route
	for _, ipv4addr := range port.Ipv4addr {
		ipv4CIDR, err := ipToCIDR(ipv4addr)
		if err != nil {
			continue
		}
		rt := Route{
			UUID:          "",
			Vrf:           port.Vrf,
			IPPrefix:      ipv4CIDR,
			Nexthop:       "",
			NhVrf:         "",
			OutputPort:    "",
			RemoteLocator: port.Locator,
			Policy:        RoutePolicyDefault,
		}
		rts = append(rts, rt)
	}
	for _, ipv6addr := range port.Ipv6addr {
		ipv6CIDR, err := ipToCIDR(ipv6addr)
		if err != nil {
			continue
		}
		rt := Route{
			UUID:          "",
			Vrf:           port.Vrf,
			IPPrefix:      ipv6CIDR,
			Nexthop:       "",
			NhVrf:         "",
			OutputPort:    "",
			RemoteLocator: port.Locator,
			Policy:        RoutePolicyDefault,
		}
		rts = append(rts, rt)
	}
	return rts
}

func portbindingCreate(row libovsdb.Row, pbUUID string) {
	var port = PortbindingParser(row)

	procBranch := definePortProcBranch(port)

	if _, ok := portInfoMap[pbUUID]; !ok {
		portInfoMap[pbUUID] = port
		portType[pbUUID] = procBranch
	} else {
		portbindingUpdatePortBranch(row, pbUUID)
	}

	log.Info("portbindingCreate PortInfo %+v ==> Branch %d\n", port, procBranch)

	switch procBranch {
	case LSPACLocal:
		err := l2PortCreate(port)
		if err != nil {
			log.Warning("l2PortCreate failed %v\n", err)
			portAddToFailureChain(LSPACLocal, port)
			return
		}
		autoGatewayConfTableUpdate(port)
	case LSPACRemote:
		fdbs := portGenRfdbSet(port)
		nhs := portGenRneighSet(port)
		err1 := remoteFdbCreate(fdbs)
		err2 := remoteNeighCreate(nhs)
		if err1 != nil || err2 != nil {
			log.Warning("LSPACRemote failed err1 %v err2 %b\n", err1, err2)
			portAddToFailureChain(LSPACRemote, port)
			return
		}
	case LSPPatchLSP:
		err := l2PortCreate(port)
		if err != nil {
			log.Warning("l2PortCreate failed %v\n", err)
			portAddToFailureChain(LSPPatchLSP, port)
			return
		}
	case LSPPatchLRP:
		err := l2PortCreate(port)
		if err != nil {
			log.Warning("l2PortCreate failed %v\n", err)
			portAddToFailureChain(LSPPatchLRP, port)
			return
		}

		portProcFailureChain(LSPACRemote)
	case LSPLocalNET:
		err := clearAutoGatewayConfBdValue(port)
		if err != nil {
			log.Error("Clear autogatewayconf'Bd value failed.\n")
			return
		}
	case LRPACLocal:
		err := l3portCreate(port)
		if err != nil {
			log.Warning("l3PortCreate failed %v\n", err)
			portAddToFailureChain(LRPACLocal, port)
			return
		}
	case LRPACRemote:
		rts := portGenRouteSet(port)
		err := routeSetCreate(rts)
		if err != nil {
			portAddToFailureChain(LRPACRemote, port)
			return
		}
	case LRPPatchLSP:
		// TODO: no need to add vtep l3port?
		err := l3portCreate(port)
		if err != nil {
			log.Warning("l3PortCreate failed %v\n", err)
			portAddToFailureChain(LRPPatchLSP, port)
			return
		}

		portProcFailureChain(LSPACRemote)
	case LRPPatchLRP:
		err := l3portCreate(port)
		if err != nil {
			log.Warning("l3PortCreate failed %v\n", err)
			portAddToFailureChain(LRPPatchLRP, port)
			return
		}
	default:
		return
	}
	portRemoveFromFailureChain(procBranch, port.LogicalPort)
}

func portbindingRemove(row libovsdb.Row, pbUUID string) {
	//var port = PortbindingParser(row)

	// get port info form portInfoMap
	port, ok := portInfoMap[pbUUID]
	if !ok {
		log.Warning("Get port info failed\n")
		return
	}
	procBranch := definePortProcBranch(port)
	var err error

	log.Info("PortbindingRemove PortInfo %+v ==> Branch %d\n", port, procBranch)

	switch procBranch {
	case LSPACLocal:
		err = l2PortRemove(port)
	case LSPACRemote:
		fdbs := portGenRfdbSet(port)
		nhs := portGenRneighSet(port)
		err = remoteFdbRemove(fdbs)
		err = remoteNeighRemove(nhs)
	case LSPPatchLSP:
		err = l2PortRemove(port)
	case LSPPatchLRP:
		err = l2PortRemove(port)
	case LRPACLocal:
		err = l3portRemove(port)
	case LRPACRemote:
		rts := portGenRouteSet(port)
		err = routeSetRemove(rts)
	case LRPPatchLSP:
		err = l3portRemove(port)
	case LRPPatchLRP:
		err = l3portRemove(port)
	}

	if err != nil {
		log.Warning("PortbindingRemove failed %v\n", err)
	}

	// remove port info map
	delete(portType, pbUUID)
	delete(portInfoMap, pbUUID)
}

func portbindingUpdate(newrow libovsdb.Row, oldrow libovsdb.Row, pbUUID string) {
	var err error

	for field, oldValue := range oldrow.Fields {
		log.Info("update field %s old-value %v\n", field, oldValue)
		switch field {
		case ovnsb.PortBindingFieldType:
			err = portbindingUpdateType(newrow, oldValue, pbUUID)
		case ovnsb.PortBindingFieldMac:
			err = portbindingUpdateMac(newrow, oldValue, pbUUID)
		case ovnsb.PortBindingFieldChassis:
			err = portbindingUpdateChassis(newrow, oldValue, pbUUID)
		case ovnsb.PortBindingFieldOptions:
			oldOptions := oldValue.(libovsdb.OvsMap).GoMap
			err = portbindingUpdateOptions(newrow, oldOptions)
		default:
			continue
		}
	}

	portbindingUpdatePortBranch(newrow, pbUUID)

	if err != nil {
		log.Error("portbindingUpdate failed %v\n", err)
	}
}

func portbindingUpdatePortBranch(newrow libovsdb.Row, pbUUID string) error {
	// get new port branch
	port := PortbindingParser(newrow)

	procBranch := definePortProcBranch(port)
	if portType[pbUUID] == procBranch {
		return nil
	}

	switch portType[pbUUID] {
	case LSPACLocal:
		l2PortRemove(portInfoMap[pbUUID])
	case LSPACRemote:
		fdbs := portGenRfdbSet(portInfoMap[pbUUID])
		nhs := portGenRneighSet(portInfoMap[pbUUID])
		remoteFdbRemove(fdbs)
		remoteNeighRemove(nhs)
	case LSPPatchLSP:
		l2PortRemove(portInfoMap[pbUUID])
	case LSPPatchLRP:
		l2PortRemove(portInfoMap[pbUUID])
	case LRPACLocal:
		l3portRemove(portInfoMap[pbUUID])
	case LRPACRemote:
		rts := portGenRouteSet(portInfoMap[pbUUID])
		routeSetRemove(rts)
	case LRPPatchLSP:
		l3portRemove(portInfoMap[pbUUID])
	case LRPPatchLRP:
		l3portRemove(portInfoMap[pbUUID])
	}

	portType[pbUUID] = procBranch
	portInfoMap[pbUUID] = port

	switch procBranch {
	case LSPACLocal:
		err := l2PortCreate(port)
		if err != nil {
			log.Warning("l2PortCreate failed %v\n", err)
			portAddToFailureChain(LSPACLocal, port)
		}
	case LSPACRemote:
		fdbs := portGenRfdbSet(port)
		nhs := portGenRneighSet(port)
		err1 := remoteFdbCreate(fdbs)
		err2 := remoteNeighCreate(nhs)
		if err1 != nil || err2 != nil {
			log.Warning("LSPACRemote failed err1 %v err2 %b\n", err1, err2)
			portAddToFailureChain(LSPACRemote, port)
		}
	case LSPPatchLSP:
		err := l2PortCreate(port)
		if err != nil {
			portAddToFailureChain(LSPPatchLSP, port)
		}
	case LSPPatchLRP:
		err := l2PortCreate(port)
		if err != nil {
			portAddToFailureChain(LSPPatchLRP, port)
		}
	case LRPACLocal:
		err := l3portCreate(port)
		if err != nil {
			portAddToFailureChain(LRPACLocal, port)
		}
	case LRPACRemote:
		rts := portGenRouteSet(port)
		err := routeSetCreate(rts)
		if err != nil {
			portAddToFailureChain(LRPACRemote, port)
		}
	case LRPPatchLSP:
		err := l3portCreate(port)
		if err != nil {
			portAddToFailureChain(LRPPatchLSP, port)
		}
	case LRPPatchLRP:
		err := l3portCreate(port)
		if err != nil {
			portAddToFailureChain(LRPPatchLRP, port)
		}
	}

	return nil
}

func portbindingUpdateChassis(newrow libovsdb.Row, oldValue interface{}, pbUUID string) error {
	oldChassis, ok := oldValue.(libovsdb.UUID)
	oldChassisExist := false
	if ok {
		oldChassisExist = true
	}
	log.Info("portbindingUpdateChassis old chassis %v|%v\n", oldChassisExist, oldChassis)

	// get new port branch
	port := PortbindingParser(newrow)
	portInfoMap[pbUUID] = port

	procBranch := definePortProcBranch(port)
	if portType[pbUUID] == procBranch {
		log.Info("portbindingUpdateChassis proc branch not changed, ignore\n")
		return nil
	}

	switch portType[pbUUID] {
	case LSPACLocal:
		l2PortRemove(port)
	case LSPACRemote:
		fdbs := portGenRfdbSet(port)
		nhs := portGenRneighSet(port)
		remoteFdbRemove(fdbs)
		remoteNeighRemove(nhs)
	case LSPPatchLSP:
		l2PortRemove(port)
	case LSPPatchLRP:
		l2PortRemove(port)
	case LRPACLocal:
		l3portRemove(port)
	case LRPACRemote:
		rts := portGenRouteSet(port)
		routeSetRemove(rts)
	case LRPPatchLSP:
		l3portRemove(port)
	case LRPPatchLRP:
		l3portRemove(port)
	}

	portType[pbUUID] = procBranch
	switch procBranch {
	case LSPACLocal:
		err := l2PortCreate(port)
		if err != nil {
			log.Warning("l2PortCreate failed %v\n", err)
			portAddToFailureChain(LSPACLocal, port)
		}
	case LSPACRemote:
		fdbs := portGenRfdbSet(port)
		nhs := portGenRneighSet(port)
		err1 := remoteFdbCreate(fdbs)
		err2 := remoteNeighCreate(nhs)
		if err1 != nil || err2 != nil {
			log.Warning("LSPACRemote failed err1 %v err2 %b\n", err1, err2)
			portAddToFailureChain(LSPACRemote, port)
		}
	case LSPPatchLSP:
		err := l2PortCreate(port)
		if err != nil {
			portAddToFailureChain(LSPPatchLSP, port)
		}
	case LSPPatchLRP:
		err := l2PortCreate(port)
		if err != nil {
			portAddToFailureChain(LSPPatchLRP, port)
		}
	case LRPACLocal:
		err := l3portCreate(port)
		if err != nil {
			portAddToFailureChain(LRPACLocal, port)
		}
	case LRPACRemote:
		rts := portGenRouteSet(port)
		err := routeSetCreate(rts)
		if err != nil {
			portAddToFailureChain(LRPACRemote, port)
		}
	case LRPPatchLSP:
		err := l3portCreate(port)
		if err != nil {
			portAddToFailureChain(LRPPatchLSP, port)
		}
	case LRPPatchLRP:
		err := l3portCreate(port)
		if err != nil {
			portAddToFailureChain(LRPPatchLRP, port)
		}
	}

	return nil
}

func portbindingUpdateType(newrow libovsdb.Row, oldValue interface{}, pbUUID string) error {
	newPort := PortbindingParser(newrow)

	procBranch := definePortProcBranch(newPort)
	if oldValue == "localnet" || procBranch == LSPLocalNET {
		err := portbindingUpdateLocalNET(newPort)
		if err != nil {
			log.Error("Update localnet portbinding failed.\n")
			return err
		}
	}

	return nil
}

func portbindingUpdateMac(newrow libovsdb.Row, oldValue interface{}, pbUUID string) error {
	newPort := PortbindingParser(newrow)

	procBranch := definePortProcBranch(newPort)
	if procBranch != LSPACRemote {
		log.Info("portbindingUpdateMac not AC remote port ignored\n")
		return nil
	}

	// update port info cache
	portInfoMap[pbUUID] = newPort
	portType[pbUUID] = procBranch

	var oldAddr []string
	switch oldValue.(type) {
	case string:
		oldAddr = append(oldAddr, oldValue.(string))
	case libovsdb.OvsSet:
		oldAddr = odbc.ConvertGoSetToStringArray(oldValue.(libovsdb.OvsSet))
	}

	macOp := make(map[string]string)
	macAddr, _, _ := addressParser(oldAddr)

	tablePortBinding := ovnsb.ConvertRowToPortBinding(libovsdb.ResultRow(newrow.Fields))
	newMacAddr, _, _ := addressParser(tablePortBinding.Mac)
	for _, mac := range macAddr {
		macOp[mac] = odbc.OpDelete
	}
	for _, mac := range newMacAddr {
		if _, ok := macOp[mac]; ok {
			macOp[mac] = "keep"
		} else {
			macOp[mac] = odbc.OpInsert
		}
	}

	remoteFdbUpdate(newPort, macOp)

	return nil
}

func portbindingUpdateOptionsPS(newPS string, oldPS string, pbIndex ovnsb.PortBindingIndex1) error {
	if newPS != "" && oldPS == "" {
		phsicalSwitchIndex := vtepdb.PhysicalSwitchIndex{
			Name: newPS,
		}
		tablePS, err := vtepdb.PhysicalSwitchGetByIndex(phsicalSwitchIndex)
		if err != nil {
			return fmt.Errorf("Port-binding option physical-switch %s not exist", phsicalSwitchIndex.Name)
		}

		chassisIndex := ovnsb.ChassisIndex{
			Name: tablePS.SystemID,
		}
		tableChassis, err := ovnsb.ChassisGetByIndex(chassisIndex)
		if err != nil {
			return fmt.Errorf("Get chassis %v failed", chassisIndex.Name)
		}

		// Set chassis to port binding
		err = ovnsb.PortBindingSetField(pbIndex, ovnsb.
			PortBindingFieldChassis, libovsdb.UUID{GoUUID: tableChassis.UUID})
		if err != nil {
			return fmt.Errorf("PortBinding update Chassis failed")
		}

		// remote port change to local would process in chassis update msg

	} else if newPS != "" && oldPS != "" {
		phsicalSwitchIndex := vtepdb.PhysicalSwitchIndex{
			Name: newPS,
		}
		tablePS, err := vtepdb.PhysicalSwitchGetByIndex(phsicalSwitchIndex)
		if err != nil {
			return fmt.Errorf("Port-binding option physical-switch %s not exist", phsicalSwitchIndex.Name)
		}

		chassisIndex := ovnsb.ChassisIndex{
			Name: tablePS.SystemID,
		}
		tableChassis, err := ovnsb.ChassisGetByIndex(chassisIndex)
		if err != nil {
			return fmt.Errorf("Get chassis %v failed", chassisIndex.Name)
		}

		// Set chassis to port binding
		err = ovnsb.PortBindingSetField(pbIndex, ovnsb.
			PortBindingFieldChassis, libovsdb.UUID{GoUUID: tableChassis.UUID})
		if err != nil {
			return fmt.Errorf("PortBinding update Chassis failed")
		}

		// update port PS
	} else if newPS == "" && oldPS != "" {
		phsicalSwitchIndex := vtepdb.PhysicalSwitchIndex{
			Name: oldPS,
		}
		tablePS, err := vtepdb.PhysicalSwitchGetByIndex(phsicalSwitchIndex)
		if err != nil {
			return fmt.Errorf("Port unbinding option physical-switch %s not exist", phsicalSwitchIndex.Name)
		}

		chassisIndex := ovnsb.ChassisIndex{
			Name: tablePS.SystemID,
		}
		tableChassis, err := ovnsb.ChassisGetByIndex(chassisIndex)
		if err != nil {
			return fmt.Errorf("Get chassis %v failed", chassisIndex.Name)
		}

		err = ovnsb.PortBindingUpdateChassisDelvalue(pbIndex, []libovsdb.UUID{{GoUUID: tableChassis.UUID}})
		if err != nil {
			return fmt.Errorf("Port %s unbinding to chassis failed", pbIndex.LogicalPort)
		}

		// local to remote
	}

	return nil
}

func portbindingUpdateOptions(newrow libovsdb.Row, oldOptions map[interface{}]interface{}) error {
	var err error

	tablePortBinding := ovnsb.ConvertRowToPortBinding(libovsdb.ResultRow(newrow.Fields))
	pbIndex := ovnsb.PortBindingIndex1{
		LogicalPort: tablePortBinding.LogicalPort,
	}

	// consider options:physical_switch
	newOptions := tablePortBinding.Options
	oldPS, _ := oldOptions[PbOptionPhysicalSwitch].(string)
	newPS, _ := newOptions[PbOptionPhysicalSwitch].(string)

	err = portbindingUpdateOptionsPS(newPS, oldPS, pbIndex)
	if err != nil {
		log.Error("%v\n", err)
	}

	return nil
}
