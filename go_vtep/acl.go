package govtep

import (
	"strconv"
	"strings"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnnb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnnorthbound"
	ovnsb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnsouthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// ACL in vtep db
type ACL struct {
	UUID     string
	ACLRules []string
	ACLName  string //   must be unique within table
}

// ACLRule in vtep db
type ACLRule struct {
	UUID         string
	Sequence     int
	SrcMac       string
	DstMac       string
	Ethertype    string
	SrcIP        string
	SrcMask      string
	DstIP        string
	DstMask      string
	Protocol     int
	SrcPortMin   int
	SrcPortMax   int
	DstPortMin   int
	DstPortMax   int
	TCPFlags     int
	TCPFlagsMask int
	IcmpType     int
	IcmpCode     int
	Direction    string //egress or ingress or all
	Action       string //either deny or permit
}

// acl match
const (
	OVSMatchInPort    string = "inport"
	OVSMatchOutPort   string = "outport"
	OVSMatchIP        string = "ip4"
	OVSMatchIPv6      string = "ip6"
	OVSMatchARP       string = "arp"
	OVSMatchICMP4     string = "icmp4"
	OVSMatchICMP6     string = "icmp6"
	OVSMatchTCP       string = "tcp"
	OVSMatchUDP       string = "udp"
	OVSMatchProtocol  string = "protocol"
	OVSMatchEthertype string = "ethertype"
	OVSMatchMacSrc    string = "eth.src"
	OVSMatchMacDst    string = "eth.dst"
	OVSMatchIPSrc     string = "ip4.src"
	OVSMatchIPDst     string = "ip4.dst"
	OVSMatchIPv6Src   string = "ip6.src"
	OVSMatchIPv6Dst   string = "ip6.dst"
	OVSMatchTCPSrc    string = "tcp.src"
	OVSMatchTCPDst    string = "tcp.dst"
	OVSMatchUDPSrc    string = "udp.src"
	OVSMatchUDPDst    string = "udp.dst"
	OVSMatchICMP4Type string = "icmp4.type"
	OVSMatchICMP4Code string = "icmp4.code"
	OVSMatchICMP6Type string = "icmp6.type"
	OVSMatchICMP6Code string = "icmp6.code"
)

func aclNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate) {
	switch op {
	case odbc.OpInsert:
		aclCreate(rowUpdate.New)
	case odbc.OpDelete:
		aclRemove(rowUpdate.Old)
	case odbc.OpUpdate:
		log.Warning("ACL not support update\n")
	}
}

func aclCreate(row libovsdb.Row) error {
	tableACL := ovnnb.ConvertRowToACL(row.Fields)
	log.Info("aclCreate %+v\n", tableACL)

	// check if this acl attached to phsical switch Local port
	if !strings.Contains(tableACL.Match, "inport") && !strings.Contains(tableACL.Match, "outport") {
		log.Info("acl %s not supported without match port\n", tableACL.Name[0])
		return nil
	}

	aclRuleField := strParseToACLField(tableACL.Match)

	portName := ""
	if aclRuleField["inport"] != "" && tableACL.Direction == ovnnb.ACLDirectionFromlport {
		portName = aclRuleField["inport"]
	} else if aclRuleField["outport"] != "" && tableACL.Direction == ovnnb.ACLDirectionTolport {
		portName = aclRuleField["outport"]
	} else {
		log.Error("ACL should specific direction with same in/out port\n")
		return nil
	}
	if portName == "" {
		log.Warning("ACL %s get match port failed\n", tableACL.Name)
		return nil
	}
	pbIndex := ovnsb.PortBindingIndex1{LogicalPort: portName}
	tablePB, err := ovnsb.PortBindingGetByIndex(pbIndex)
	if err != nil {
		log.Warning("ACL match port %s not existed, ignored\n", portName)
		return nil
	}

	if len(tablePB.Chassis) == 0 {
		log.Warning("ACL match port %s not binding to local chassis, ignored\n", portName)
		return nil
	}

	tableChassis, err := ovnsb.ChassisGetByUUID(tablePB.Chassis[0].GoUUID)
	if err != nil {
		log.Warning("ACL match port %s chassis get failed\n", portName)
		return nil
	}

	psIndex := vtepdb.PhysicalSwitchIndex1{
		SystemID: tableChassis.Name,
	}
	_, err = vtepdb.PhysicalSwitchGetByIndex(psIndex)
	if err != nil {
		log.Warning("ACL match port %s chassis not binding to local phsical switch\n", portName)
		return nil
	}

	if len(tableACL.Name) == 0 {
		log.Warning("ACL match port %s chassis has no name, ignored\n", portName)
		return nil
	}

	vtepACL := vtepdb.TableACL{
		Name:  tableACL.Name[0],
		Ports: portName,
	}

	if tableACL.Direction == ovnnb.ACLDirectionFromlport {
		vtepACL.Stage = vtepdb.ACLStageIngress
	} else {
		vtepACL.Stage = vtepdb.ACLStageEgress
	}

	_, err = vtepdb.ACLAdd(vtepACL)
	if err != nil {
		log.Warning("ACL %s create in vtepdb failed\n", vtepACL.Name)
		return nil
	}

	// convert match to vtepdb ACL
	var vtepACLRule vtepdb.TableACLRule
	for key, val := range aclRuleField {
		switch key {
		case OVSMatchInPort:
			// vtepACLRule.InPorts
		case OVSMatchOutPort:
			// vtepACLRule.OutPorts
		case OVSMatchIP:
			vtepACLRule.Ethertype = []string{"0x0800"}
		case OVSMatchIPv6:
			vtepACLRule.Ethertype = []string{"0x86DD"}
		case OVSMatchARP:
			vtepACLRule.Ethertype = []string{"0x0806"}
		case OVSMatchICMP4:
			vtepACLRule.Protocol = []int{1}
		case OVSMatchICMP6:
			vtepACLRule.Protocol = []int{58}
		case OVSMatchTCP:
			vtepACLRule.Protocol = []int{6}
		case OVSMatchUDP:
			vtepACLRule.Protocol = []int{17}
		case OVSMatchProtocol:
			proto, _ := strconv.Atoi(val)
			vtepACLRule.Protocol = []int{proto}
		case OVSMatchEthertype:
			vtepACLRule.Ethertype = []string{val}
		case OVSMatchMacSrc:
			vtepACLRule.SourceMac = []string{val}
		case OVSMatchMacDst:
			vtepACLRule.DestMac = []string{val}
		case OVSMatchIPSrc:
			vtepACLRule.SourceIP = []string{val}
		case OVSMatchIPDst:
			vtepACLRule.DestIP = []string{val}
		case OVSMatchIPv6Src:
			vtepACLRule.SourceIP = []string{val}
		case OVSMatchIPv6Dst:
			vtepACLRule.DestIP = []string{val}
		case OVSMatchTCPSrc:
			port, _ := strconv.Atoi(val)
			vtepACLRule.DestPortMin = []int{port}
			vtepACLRule.DestPortMax = []int{port}
		case OVSMatchTCPDst:
			port, _ := strconv.Atoi(val)
			vtepACLRule.DestPortMin = []int{port}
			vtepACLRule.DestPortMax = []int{port}
		case OVSMatchUDPSrc:
			port, _ := strconv.Atoi(val)
			vtepACLRule.DestPortMin = []int{port}
			vtepACLRule.DestPortMax = []int{port}
		case OVSMatchUDPDst:
			port, _ := strconv.Atoi(val)
			vtepACLRule.DestPortMin = []int{port}
			vtepACLRule.DestPortMax = []int{port}
		case OVSMatchICMP4Type:
		case OVSMatchICMP4Code:
		case OVSMatchICMP6Type:
		case OVSMatchICMP6Code:
		default:
			log.Warning("Unknown field key %s for vtep\n", key)
			return nil
		}
	}

	vtepACLRule.ACLName = vtepACL.Name
	vtepACLRule.Sequence = tableACL.Priority
	vtepACLRule.Action = tableACL.Action

	vtepACLIndex := vtepdb.ACLIndex{
		Name: tableACL.Name[0],
	}

	err = vtepdb.ACLUpdateAddACLRules(vtepACLIndex, vtepACLRule)
	if err != nil {
		log.Warning("ACL %s add rule field %v\n", tableACL.Name, err)
	}

	return nil
}

func aclRemove(row libovsdb.Row) error {
	tableACL := ovnnb.ConvertRowToACL(row.Fields)

	// check if this acl attached to phsical switch Local port
	if !strings.Contains(tableACL.Match, "inport") && !strings.Contains(tableACL.Match, "outport") {
		log.Info("acl %s not supported without match port\n", tableACL.Name)
		return nil
	}

	aclRuleField := strParseToACLField(tableACL.Match)

	portName := ""
	if aclRuleField["inport"] != "" && tableACL.Direction == ovnnb.ACLDirectionFromlport {
		portName = aclRuleField["inport"]
	} else if aclRuleField["outport"] != "" && tableACL.Direction == ovnnb.ACLDirectionTolport {
		portName = aclRuleField["outport"]
	} else {
		log.Error("ACL should specific direction with same in/out port\n")
		return nil
	}
	if portName == "" {
		log.Warning("ACL %s get match port failed\n", tableACL.Name)
		return nil
	}
	l2PortIndex := vtepdb.L2portIndex{LogicalPort: portName}
	_, err := vtepdb.L2portGetByIndex(l2PortIndex)
	if err != nil {
		log.Warning("ACL match port %s is not local port, ignored\n", portName)
		return nil
	}

	vtepACLIndex := vtepdb.ACLIndex{
		Name: tableACL.Name[0],
	}
	vtepdb.ACLDelByIndex(vtepACLIndex)

	return nil
}

func strParseToACLField(str string) map[string]string {
	aclRuleField := make(map[string]string)

	str = strings.Replace(str, " ", "", -1)
	str = strings.Replace(str, "\"", "", -1)
	matches := strings.Split(str, "&&")
	for _, match := range matches {
		kv := strings.Split(match, "==")
		if len(kv) == 2 {
			aclRuleField[kv[0]] = kv[1]
		} else {
			aclRuleField[kv[0]] = ""
		}
	}

	return aclRuleField
}
