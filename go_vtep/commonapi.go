package govtep

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnsb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnsouthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
)

func getVnetName(lnType string, vni int) (string, error) {
	var name string
	var err error
	switch lnType {
	case DatapathTypeLS:
		name = getBdNameByVni(vni)
	case DatapathTypeLR:
		name = getVrfNameByVni(vni)
	default:
		err = errors.New("Invalid Logical network type")
	}
	return name, err
}

func getBdNameByVni(vni int) string {
	return "Bd" + strconv.Itoa(int(vni))
}

func getVrfNameByVni(vni int) string {
	return "Vrf" + strconv.Itoa(int(vni))
}

func getLnByDpuuid(dpuuid string) (string, string, int) {
	var lnType string
	var lnName string
	var tunnelKey int

	tableDatapathBinding, err := ovnsb.DatapathBindingGetByUUID(dpuuid)
	if err != nil {
		log.Warning("Get Datapath_Binding %s failed\n", dpuuid)
		return "", "", 0
	}

	tunnelKey = tableDatapathBinding.TunnelKey
	if tableDatapathBinding.ExternalIds[DatapathTypeLS] != nil {
		lnType = DatapathTypeLS
		lnName = tableDatapathBinding.ExternalIds[DatapathTypeLS].(string)
	} else if tableDatapathBinding.ExternalIds[DatapathTypeLR] != nil {
		lnType = DatapathTypeLR
		lnName = tableDatapathBinding.ExternalIds[DatapathTypeLR].(string)
	}

	return lnType, lnName, tunnelKey
}

func getSubportName(parentPort string, vlantag []int) string {
	if len(vlantag) == 0 {
		return parentPort
	} else if len(vlantag) == 1 {
		return parentPort + "." + strconv.Itoa(vlantag[0])
	} else if len(vlantag) == 2 {
		return parentPort + "." + strconv.Itoa(vlantag[0]) + "." + strconv.Itoa(vlantag[1])
	} else {
		return ""
	}
}

func getDpuuidByLogicalPort(logicalPort string) (string, error) {
	pbIndex := ovnsb.PortBindingIndex1{
		LogicalPort: logicalPort,
	}
	tablePB, err := ovnsb.PortBindingGetByIndex(pbIndex)
	if err != nil {
		return "", fmt.Errorf("Get Port_Binding %s failed", logicalPort)
	}

	if tablePB.Datapath.GoUUID != "" {
		return tablePB.Datapath.GoUUID, nil
	}

	return "", errors.New("DP not found")
}

func lpBelongWhichVnet(logicalPort string) (string, string) {
	ltype := getPortLtype(logicalPort)
	if ltype == PortLtypeLSP {
		return lspBelongWhichBd(logicalPort)
	} else if ltype == PortLtypeLRP {
		return lspBelongWhichVrf(logicalPort)
	}
	return "", ""
}

func lspBelongWhichBd(lsp string) (string, string) {
	dpuuid, err := getDpuuidByLogicalPort(lsp)
	if err != nil {
		return "", ""
	}
	_, _, tunnelKey := getLnByDpuuid(dpuuid)
	return getBdNameByVni(tunnelKey), VnetTypeBD
}

func lspBelongWhichVrf(lrp string) (string, string) {
	dpuuid, err := getDpuuidByLogicalPort(lrp)
	if err != nil {
		return "", ""
	}
	_, _, tunnelKey := getLnByDpuuid(dpuuid)
	return getVrfNameByVni(tunnelKey), VnetTypeVRF
}

func getBdByLogicalPort(logicalPort string) (string, error) {
	dpuuid, err := getDpuuidByLogicalPort(logicalPort)
	if err != nil {
		return "", err
	}
	_, _, tunnelKey := getLnByDpuuid(dpuuid)
	return getBdNameByVni(tunnelKey), nil
}

func getBdifportName(bd string) string {
	return bd
}

func getVethportName(vnetTunnelKey int, portTunnelKey int) string {
	return "veth" + strconv.Itoa(int(vnetTunnelKey)) + "-" + strconv.Itoa(int(portTunnelKey))
}

func getPortLtype(logicalPort string) string {
	var ltype string
	dpuuid, err := getDpuuidByLogicalPort(logicalPort)
	if err != nil {
		return ""
	}
	lnType, _, _ := getLnByDpuuid(dpuuid)
	switch lnType {
	case DatapathTypeLS:
		ltype = PortLtypeLSP
	case DatapathTypeLR:
		ltype = PortLtypeLRP
	}
	return ltype
}

func getPortPtype(logicalPort string) string {
	var ptype string
	dpuuid, err := getDpuuidByLogicalPort(logicalPort)
	if err != nil {
		return ""
	}
	lnType, _, _ := getLnByDpuuid(dpuuid)
	switch lnType {
	case DatapathTypeLS:
		ptype = PortPtypeL2
	case DatapathTypeLR:
		ptype = PortPtypeL3
	}
	return ptype
}

func getPortByLp(logicalPort string) string {
	var pport string
	ptype := getPortPtype(logicalPort)
	if ptype == PortPtypeL2 {
		pport, _ = getL2port(logicalPort)
	} else if ptype == PortPtypeL3 {
		pport, _ = getL3port(logicalPort)
	}
	return pport
}

func getL2port(logicalPort string) (string, error) {
	l2PortIndex := vtepdb.L2portIndex{
		LogicalPort: logicalPort,
	}

	tableL2Port, err := vtepdb.L2portGetByIndex(l2PortIndex)
	if err != nil {
		return "", errors.New("get L2Port name failed")
	}

	return tableL2Port.Name, nil
}

func getL3port(logicalPort string) (string, error) {
	l3PortIndex := vtepdb.L3portIndex{
		LogicalPort: logicalPort,
	}

	tableL3Port, err := vtepdb.L3portGetByIndex(l3PortIndex)
	if err != nil {
		return "", errors.New("get L3Port name failed")
	}

	return tableL3Port.Name, nil
}

func getLocation(chassisName string, phySwitch string, phySwitchGroup string) string {
	var location = LocationRemote

	if phySwitch != "" {
		phsicalSwitchIndex := vtepdb.PhysicalSwitchIndex{
			Name: phySwitch,
		}
		tablePS, err := vtepdb.PhysicalSwitchGetByIndex(phsicalSwitchIndex)
		if err == nil {
			if tablePS.SystemID == chassisName {
				location = LocationLocal
			} else {
				location = LocationRemote
			}
		}
		return location
	}

	return location
}

func getVtepChassisNameBySystemID(systemID string) string {
	return "VTEP-" + systemID
}

func getOvnsbChassis(chassisUUID string) (ovnsb.TableChassis, error) {
	var chassis ovnsb.TableChassis
	chassis, err := ovnsb.ChassisGetByUUID(chassisUUID)
	if err != nil {
		return chassis, fmt.Errorf("Chassis %s not exist", chassisUUID)
	}
	return chassis, nil
}

func getLocatorUUID(chassisName string) (string, error) {
	locatorIndex := vtepdb.LocatorIndex{
		ChassisName: chassisName,
	}
	tableLocator, err := vtepdb.LocatorGetByIndex(locatorIndex)
	if err != nil {
		return "", fmt.Errorf("Locator for chassis %s not exist yet", chassisName)
	}
	return tableLocator.UUID, nil
}

func addressParser(addressSet []string) ([]string, []string, []string) {
	var mac []string
	var ipv4 []string
	var ipv6 []string
	for _, address := range addressSet {
		for _, addr := range strings.Split(address, " ") {
			if strings.Contains(addr, "/") {
				ip, _, err := net.ParseCIDR(addr)
				if err != nil {
					continue
				}
				if len(ip) == net.IPv6len {
					if ip.To4() == nil {
						ipv6 = append(ipv6, addr)
					} else {
						ipv4 = append(ipv4, addr)
					}
				} else {
					continue
				}
			} else {
				ip := net.ParseIP(addr)
				if len(ip) == net.IPv6len {
					if ip.To4() == nil {
						ipv6 = append(ipv6, addr)
					} else {
						ipv4 = append(ipv4, addr)
					}
				} else {
					mac = append(mac, addr)
				}
			}
		}
	}
	return mac, ipv4, ipv6
}

func ipToCIDR(ipStr string) (string, error) {
	ip := net.ParseIP(ipStr)
	if len(ip) == net.IPv4len {
		return ipStr + "/32", nil
	} else if len(ip) == net.IPv6len {
		return ipStr + "/128", nil
	} else {
		return "", nil
	}
}

func bdIsExist(bdName string) bool {
	bdIndex := vtepdb.BridgeDomainIndex{
		Name: bdName,
	}

	_, err := vtepdb.BridgeDomainGetByIndex(bdIndex)
	if err != nil {
		return false
	}

	return true
}

func vrfIsExist(vrfName string) bool {
	vrfIndex := vtepdb.VrfIndex{
		Name: vrfName,
	}

	_, err := vtepdb.VrfGetByIndex(vrfIndex)
	if err != nil {
		return false
	}

	return true
}

func l2portIsExist(l2PortName string) bool {
	l2portIndex := vtepdb.L2portIndex1{
		Name: l2PortName,
	}

	_, err := vtepdb.L2portGetByIndex(l2portIndex)
	if err != nil {
		return false
	}

	return true
}

func l3portIsExist(l3PortName string) bool {
	l3portIndex := vtepdb.L3portIndex1{
		Name: l3PortName,
	}

	_, err := vtepdb.L3portGetByIndex(l3portIndex)
	if err != nil {
		return false
	}

	return true
}

func l3portIsIRB(l3PortName string) bool {
	return strings.Contains(l3PortName, "bd")
}

func l3portIsVeth(l3PortName string) bool {
	return strings.Contains(l3PortName, "veth")
}

func deletePreAndSufSpace(str string) string {
	strList := []byte(str)
	spaceCount, count := 0, len(strList)
	for i := 0; i <= len(strList)-1; i++ {
		if strList[i] == 32 {
			spaceCount++
		} else {
			break
		}
	}

	strList = strList[spaceCount:]
	spaceCount, count = 0, len(strList)
	for i := count - 1; i >= 0; i-- {
		if strList[i] == 32 {
			spaceCount++
		} else {
			break
		}
	}

	return string(strList[:count-spaceCount])
}
