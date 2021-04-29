package driver

import (
	"fmt"
	"strconv"
	"strings"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"

	"github.com/ebay/libovsdb"
)

type pbrAPI struct {
	moduleID int
}

var pbrAPIs = pbrAPI{
	moduleID: tai.ObjectIDPBR,
}

const pbrACLRuleDefSeq = 1

var ecmpGroupIDPool = make(map[int]int)

func getVniFromVrf(vrf string) int {
	vni, _ := strconv.Atoi(vrf[3:])
	return vni
}

const (
	pbrSequenceLBOffset       = 0
	pbrSequenceLBNoPortOffset = 4000
	pbrSequenceDNATOffset     = 8000
	pbrSequenceSNATOffset     = 12000
)

const (
	_ = iota
	eipOpAdd
	eipOpDel
)

func pbrAclCreate(vrf string) error {
	var err error

	aclIndex := cdb.ACLIndex{
		ACLName: "PBR_" + vrf,
	}

	tableACL, err := cdb.ACLGetByIndex(aclIndex)
	if err != nil {
		// 1. create pbr ACL
		tableACL = cdb.TableACL{
			ACLName: aclIndex.ACLName,
			Stage:   cdb.ACLStageIngress,
			Type:    cdb.ACLTypeL3,
		}

		// get gateway port for corresponding vrf after multi vrouter supported
		agcIndex := vtepdb.AutoGatewayConfIndex{
			Vrf: vrf,
		}
		tableAGC, err := vtepdb.AutoGatewayConfGetByIndex(agcIndex)
		if err == nil {
			// get gateway port for vrf, if not exist yet, update acl binding port later
			portIndex := cdb.PortIndex{
				Name: tableAGC.PhysicalPort,
			}
			tablePort, err := cdb.PortGetByIndex(portIndex)
			if err != nil {
				log.Warning("ACL add for PBR %s binding ingress port %s not exist\n", vrf, portIndex.Name)
			}
			tableACL.Ports = []libovsdb.UUID{{GoUUID: tablePort.UUID}}
		}

		_, err = cdb.ACLAdd(tableACL)
		if err != nil {
			log.Error("ACL add for PBR %s failed %v\n", vrf, err)
			return nil
		}
	}

	return err
}

func pbrAclRemove(vrf string) error {
	aclIndex := cdb.ACLIndex{
		ACLName: "PBR_" + vrf,
	}

	return cdb.ACLDelByIndex(aclIndex)
}

func pbrUpdateGateway(vrf string, port string, vlan int) error {
	aclIndex := cdb.ACLIndex{
		ACLName: "PBR_" + vrf,
	}

	portIndex := cdb.PortIndex{
		Name: port,
	}
	tablePort, err := cdb.PortGetByIndex(portIndex)
	if err != nil {
		log.Warning("ACL update for PBR %s binding ingress port %s not exist\n", vrf, portIndex.Name)
	}

	tableACL, err := cdb.ACLGetByIndex(aclIndex)
	if err == nil {
		if len(tableACL.Ports) == 1 {
			if tableACL.Ports[0].GoUUID == tablePort.UUID {
				log.Info("ACL for PBR vrf %s ingress port %s not changed\n", vrf, port)
			} else {
				err = cdb.ACLSetField(aclIndex, cdb.ACLFieldPorts, []libovsdb.UUID{{GoUUID: tablePort.UUID}})
			}
		}
	}

	subIfIndex := cdb.InterfaceIndex{
		Name: port + "." + strconv.Itoa(vlan),
		Type: cdb.InterfaceTypeSubPort,
	}
	tableSubIF, err := cdb.InterfaceGetByIndex(subIfIndex)
	if err == nil {
		var conditions []interface{}
		conditions = append(conditions, libovsdb.
			NewCondition("vrf", "==", vrf))
		rows, num := vtepdb.ExternalIPGet(conditions)
		if num > 0 {
			for _, row := range rows {
				tableEIP := vtepdb.ConvertRowToExternalIP(row)
				var extIP string
				if strings.Contains(tableEIP.IP, ":") {
					eip := strings.Split(tableEIP.IP, ":")
					extIP = eip[0]
				} else {
					extIP = tableEIP.IP
				}

				eipConfigured := false
				for _, ipPrefix := range tableSubIF.IP {
					ip := strings.Split(ipPrefix, "/")[0]
					if ip == extIP {
						eipConfigured = true
						break
					}
				}

				if false == eipConfigured {
					err = cdb.InterfaceUpdateIPAddvalue(subIfIndex, []string{extIP + "/32"})
				}
			}
		}
	}

	return nil
}

func externalIPProcess(eip string, vrf string, op int) error {
	var err error

	acgIndex := vtepdb.AutoGatewayConfIndex{
		Vrf: vrf,
	}
	tableACG, err := vtepdb.AutoGatewayConfGetByIndex(acgIndex)
	if err != nil {
		return fmt.Errorf("Gateway port for vrf %s not found", vrf)
	}

	if false == strings.Contains(tableACG.IP, eip) {
		subPortIndex := cdb.InterfaceIndex{
			Name: tableACG.PhysicalPort + "." + strconv.Itoa(tableACG.Vlan),
			Type: cdb.InterfaceTypeSubPort,
		}
		_, err := cdb.InterfaceGetByIndex(subPortIndex)
		if err != nil {
			return fmt.Errorf("Sub interface %s not found", subPortIndex.Name)
		}

		if eipOpAdd == op {
			err = cdb.InterfaceUpdateIPAddvalue(subPortIndex, []string{eip + "/32"})
		} else if eipOpDel == op {
			err = cdb.InterfaceUpdateIPDelvalue(subPortIndex, []string{eip + "/32"})
		}
	}

	return err
}

func getSequenceFromPBR(pbr tai.PBRObj, op int) (int, error) {
	sequence := 0

	var tableExtIP vtepdb.TableExternalIP
	extIPIndex := vtepdb.ExternalIPIndex{
		IP:  pbr.IP,
		Vrf: pbr.Vrf,
	}
	vrfIndex := vtepdb.VrfIndex{
		Name: pbr.Vrf,
	}
	if pbr.Port != 0 {
		extIPIndex.IP += ":"
		extIPIndex.IP += strconv.Itoa(pbr.Port)
	}

	tableExtIP, err := vtepdb.ExternalIPGetByIndex(extIPIndex)
	if err != nil {
		if eipOpAdd == op {
			tableExtIP.IP = extIPIndex.IP
			tableExtIP.Vrf = extIPIndex.Vrf
			err = vtepdb.VrfUpdateAddExternalIps(vrfIndex, tableExtIP)
			if err != nil {
				return sequence, fmt.Errorf("Create External IP failed")
			}

			vtepdb.ExternalIPSetField(extIPIndex, vtepdb.ExternalIPFieldRefCount, 1)

			err = externalIPProcess(pbr.IP, tableExtIP.Vrf, eipOpAdd)
			if err != nil {
				log.Warning("Add eip %s for vrf %s failed", tableExtIP.IP, tableExtIP.Vrf)
			}
		} else {
			return sequence, fmt.Errorf("Get seuqence ID failed")
		}
	} else {
		if eipOpAdd == op {
			vtepdb.ExternalIPSetField(extIPIndex, vtepdb.ExternalIPFieldRefCount, tableExtIP.RefCount+1)
		} else if eipOpDel == op {
			vtepdb.ExternalIPSetField(extIPIndex, vtepdb.ExternalIPFieldRefCount, tableExtIP.RefCount-1)

			if tableExtIP.RefCount <= 1 {
				vtepdb.VrfUpdateExternalIpsDelvalue(vrfIndex, []libovsdb.UUID{{GoUUID: tableExtIP.UUID}})

				eipExist := false
				var conditions []interface{}
				conditions = append(conditions, libovsdb.
					NewCondition("vrf", "==", tableExtIP.Vrf))
				rows, num := vtepdb.ExternalIPGet(conditions)
				if num > 0 {
					for _, row := range rows {
						tableEIP := vtepdb.ConvertRowToExternalIP(row)
						if strings.Contains(tableEIP.IP, ":") {
							eip := strings.Split(tableEIP.IP, ":")
							if eip[0] == pbr.IP {
								eipExist = true
								break
							}
						} else {
							if tableEIP.IP == pbr.IP {
								eipExist = true
								break
							}
						}
					}
				}

				if false == eipExist {
					err = externalIPProcess(pbr.IP, tableExtIP.Vrf, eipOpDel)
					if err != nil {
						log.Warning("Del eip %s for vrf %s failed", tableExtIP.IP, tableExtIP.Vrf)
					}
				}
			}
		}
	}

	if 0 == tableExtIP.Sequence {
		for i := 1; i <= 4000; i++ {
			idUsed := false

			var conditions []interface{}
			conditions = append(conditions, libovsdb.
				NewCondition("vrf", "==", tableExtIP.Vrf))
			rows, num := vtepdb.ExternalIPGet(conditions)
			if num > 0 {
				for _, row := range rows {
					tableEIP := vtepdb.ConvertRowToExternalIP(row)
					if tableEIP.Sequence == i {
						idUsed = true
						break
					}
				}
			}

			if false == idUsed {
				tableExtIP.Sequence = i
				vtepdb.ExternalIPSetField(extIPIndex, vtepdb.ExternalIPFieldSequence, i)
				break
			}
		}
	}
	sequence += tableExtIP.Sequence

	switch pbr.Type {
	case vtepdb.PolicyBasedRouteTypeLb:
		if pbr.Port == 0 {
			sequence += pbrSequenceLBNoPortOffset
		} else {
			sequence += pbrSequenceLBOffset
		}
	case vtepdb.PolicyBasedRouteTypeDnatAndSnat:
		sequence += pbrSequenceDNATOffset
	case vtepdb.PolicyBasedRouteTypeDnat:
		sequence += pbrSequenceDNATOffset
	case vtepdb.PolicyBasedRouteTypeSnat:
		sequence += pbrSequenceSNATOffset
	}

	return sequence, nil
}

func getACLNameFromPBR(pbr tai.PBRObj) string {
	aclName := "PBR_" + pbr.Vrf

	return aclName
}

func getIPProtocol(proto string) int {
	var ipProtocol int

	switch proto {
	case vtepdb.PolicyBasedRouteProtocolTCP:
		ipProtocol = 6
	case vtepdb.PolicyBasedRouteProtocolUDP:
		ipProtocol = 17
	case vtepdb.PolicyBasedRouteProtocolSctp:
		ipProtocol = 132
	}

	return ipProtocol
}

func getEcmpGroupID() (int, error) {
	ecmpGroupID := 0
	var ecmpGroupIndex cdb.EcmpGroupIndex

	for i := 1; i <= cdb.EcmpGroupIDMax; i++ {
		if idInUse, ok := ecmpGroupIDPool[i]; ok {
			if 0 == idInUse {
				ecmpGroupID = i
				ecmpGroupIndex.ID = ecmpGroupID
				_, err := cdb.EcmpGroupGetByIndex(ecmpGroupIndex)
				if err != nil {
					// get a valid unused ecmpGroupID
					ecmpGroupIDPool[i] = 1
					break
				}
				// the ecmpGroup already in use, get next usable ID
				continue
			}
			continue
		} else {
			ecmpGroupID = i
			ecmpGroupIndex.ID = ecmpGroupID
			_, err := cdb.EcmpGroupGetByIndex(ecmpGroupIndex)
			if err != nil {
				ecmpGroupIDPool[i] = 1
				break
			}
			continue
		}
	}

	if ecmpGroupID < cdb.EcmpGroupIDMin || ecmpGroupID > cdb.EcmpGroupIDMax {
		return ecmpGroupID, fmt.Errorf("No invalid ECMP Group ID")
	}

	return ecmpGroupID, nil
}

func releaseEcmpGroupID(ID int) {
	if _, ok := ecmpGroupIDPool[ID]; ok {
		ecmpGroupIDPool[ID] = 0
	}
}

func (v pbrAPI) CreateObject(obj interface{}) error {
	var err error
	objPBR := obj.(tai.PBRObj)

	pbrACLName := getACLNameFromPBR(objPBR)
	aclIndex := cdb.ACLIndex{
		ACLName: pbrACLName,
	}

	tableACL, err := cdb.ACLGetByIndex(aclIndex)
	if err != nil {
		// 1. create pbr ACL
		tableACL = cdb.TableACL{
			ACLName: pbrACLName,
			Stage:   cdb.ACLStageIngress,
			Type:    cdb.ACLTypeL3,
		}

		// get gateway port for corresponding vrf after multi vrouter supported
		agcIndex := vtepdb.AutoGatewayConfIndex{
			Vrf: objPBR.Vrf,
		}
		tableAGC, err := vtepdb.AutoGatewayConfGetByIndex(agcIndex)
		if err == nil {
			portIndex := cdb.PortIndex{
				Name: tableAGC.PhysicalPort,
			}
			tablePort, err := cdb.PortGetByIndex(portIndex)
			if err != nil {
				log.Warning("ACL add for PBR %+v get ingress port %s failed\n", objPBR, portIndex.Name)
			} else {
				tableACL.Ports = []libovsdb.UUID{{GoUUID: tablePort.UUID}}
			}
		}

		_, err = cdb.ACLAdd(tableACL)
		if err != nil {
			log.Error("ACL add for PBR %+v failed %v\n", objPBR, err)
			return nil
		}
	}

	sequenceID, err := getSequenceFromPBR(objPBR, eipOpAdd)
	if err != nil {
		log.Warning("ACL rule get sequence for PBR %+v failed\n", objPBR)
		return nil
	}

	aclRuleIndex := cdb.ACLRuleIndex{
		ACLName:  pbrACLName,
		Sequence: sequenceID,
	}
	_, err = cdb.ACLRuleGetByIndex(aclRuleIndex)
	if err == nil {
		log.Warning("ACL rule for PBR %+v sequenceID %d already exist\n", objPBR, sequenceID)
		return nil
	}

	// 2. create a ecmp group for acl redirect
	ecmpGroupID, err := getEcmpGroupID()
	if err != nil {
		log.Error("%v\n", err)

		return nil
	}
	tableEcmpGroup := cdb.TableEcmpGroup{
		ID: ecmpGroupID,
	}
	ecmpGroupUUID, err := cdb.EcmpGroupAdd(tableEcmpGroup)
	if err != nil {
		log.Error("ECMP Group %d add failed\n", ecmpGroupID)
		releaseEcmpGroupID(ecmpGroupID)
		return nil
	}

	// 3. create acl rule with redirect ecmp group
	tableACLRule := cdb.TableACLRule{
		ACLName:  pbrACLName,
		Sequence: sequenceID,
		DstIP:    []string{objPBR.IP + "/32"},
	}
	if objPBR.Protocol != vtepdb.PolicyBasedRouteProtocolIgnore {
		tableACLRule.IPProtocol = []int{getIPProtocol(objPBR.Protocol)}
	}
	if objPBR.Port != 0 {
		tableACLRule.L4DstPort = []int{objPBR.Port}
	}
	tableACLRule.RedirectEcmpgroup = []libovsdb.UUID{{GoUUID: ecmpGroupUUID}}

	err = cdb.ACLUpdateAddRuleName(aclIndex, tableACLRule)
	if err != nil {
		log.Error("ACL rule add for PBR %+v failed %v\n", objPBR, err)
		//cdb.ACLDelByIndex(aclIndex)

		err = cdb.EcmpGroupDelByUUID(ecmpGroupUUID)
		if err == nil {
			releaseEcmpGroupID(ecmpGroupID)
		}

		return nil
	}

	return nil
}

func (v pbrAPI) RemoveObject(obj interface{}) error {
	objPBR := obj.(tai.PBRObj)

	var err error
	var tableEcmpGroup cdb.TableEcmpGroup

	pbrACLName := getACLNameFromPBR(objPBR)
	aclIndex := cdb.ACLIndex{
		ACLName: pbrACLName,
	}

	tableACL, err := cdb.ACLGetByIndex(aclIndex)
	if err != nil {
		log.Warning("ACL for PBR %+v not found\n", objPBR)
		return nil
	}

	sequenceID, err := getSequenceFromPBR(objPBR, eipOpDel)
	if err != nil {
		log.Warning("ACL rule get sequence for PBR %+v failed\n", objPBR)
		return nil
	}

	aclRuleIndex := cdb.ACLRuleIndex{
		ACLName:  tableACL.ACLName,
		Sequence: sequenceID,
	}
	tableACLRule, err := cdb.ACLRuleGetByIndex(aclRuleIndex)
	if err != nil {
		log.Warning("ACL rule for PBR %+v not found\n", objPBR)
		goto out
	}

	if len(tableACLRule.RedirectEcmpgroup) != 1 {
		log.Warning("ACL Rule %s for PBR should have one redirect ecmp group action\n", pbrACLName)
		return nil
	}
	tableEcmpGroup, err = cdb.EcmpGroupGetByUUID(tableACLRule.RedirectEcmpgroup[0].GoUUID)
	if err != nil {
		log.Warning("ECMP Group %s not found\n", tableACLRule.RedirectEcmpgroup[0].GoUUID)
		return nil
	}

	// remove configDB tables
	err = cdb.EcmpGroupDelByUUID(tableACLRule.RedirectEcmpgroup[0].GoUUID)
	if err == nil {
		releaseEcmpGroupID(tableEcmpGroup.ID)
	}

	err = cdb.ACLUpdateRuleNameDelvalue(aclIndex, []libovsdb.UUID{{GoUUID: tableACLRule.UUID}})
	if err != nil {
		log.Warning("ACL Rule for PBR %+v remove failed %v\n", objPBR, err)
	}

out:
	// acl rule no exist any more, remove it
	// remove acl when vrf delete, 20210421
	/*if len(tableACL.RuleName) <= 1 {
		err = cdb.ACLDelByIndex(aclIndex)
		if err != nil {
			log.Warning("ACL %v remove failed %v\n", tableACL.RuleName, err)
		}
	}*/

	return nil
}

func (v pbrAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	var err error
	objPBR := obj.(tai.PBRObj)

	pbrACLName := getACLNameFromPBR(objPBR)
	aclIndex := cdb.ACLIndex{
		ACLName: pbrACLName,
	}

	tableACL, err := cdb.ACLGetByIndex(aclIndex)
	if err != nil {
		log.Warning("ACL for PBR %+v not found\n", objPBR)
		return nil
	}

	sequenceID, err := getSequenceFromPBR(objPBR, 0)
	if err != nil {
		log.Warning("ACL rule get sequence for PBR %+v failed\n", objPBR)
		return nil
	}

	aclRuleIndex := cdb.ACLRuleIndex{
		ACLName:  tableACL.ACLName,
		Sequence: sequenceID,
	}
	tableACLRule, err := cdb.ACLRuleGetByIndex(aclRuleIndex)
	if err != nil {
		log.Warning("ACL rule for PBR %+v not found\n", objPBR)
		return nil
	}

	if len(tableACLRule.RedirectEcmpgroup) != 1 {
		log.Warning("ACL Rule %s for PBR should have one redirect ecmp group action\n", pbrACLName)
		return nil
	}
	tableEcmpGroup, err := cdb.EcmpGroupGetByUUID(tableACLRule.RedirectEcmpgroup[0].GoUUID)
	if err != nil {
		log.Warning("ECMP Group %s not found\n", tableACLRule.RedirectEcmpgroup[0].GoUUID)
		return nil
	}
	ecmpGroupIndex := cdb.EcmpGroupIndex{
		ID: tableEcmpGroup.ID,
	}

	if attrs[tai.PBRAttrNexthopGroup] != nil {
		nhGroup, ok := attrs[tai.PBRAttrNexthopGroup].([]string)
		if !ok {
			log.Warning("Get nh group for PBR %+v failed\n", objPBR)
			return nil
		}

		for _, nh := range nhGroup {
			nhIndex := cdb.NexthopIndex{
				IP:         nh,
				Type:       cdb.NexthopTypeVxlan,
				NexthopVrf: objPBR.Vrf,
				Label:      getVniFromVrf(objPBR.Vrf),
			}
			tableNh := cdb.TableNexthop{
				IP:         nh,
				Type:       cdb.NexthopTypeVxlan,
				NexthopVrf: objPBR.Vrf,
				Label:      getVniFromVrf(objPBR.Vrf),
			}
			cdbNh, err := cdb.NexthopGetByIndex(nhIndex)
			if err != nil {
				// add new nexthop
				err = cdb.EcmpGroupUpdateAddNexthopGroup(ecmpGroupIndex, tableNh)
			} else {
				// add nexthop to ecmp group
				err = cdb.EcmpGroupUpdateNexthopGroupAddvalue(ecmpGroupIndex,
					[]libovsdb.UUID{{GoUUID: cdbNh.UUID}})
			}

			if err != nil {
				log.Warning("Add nexthop %s to ecmp group %d failed\n", nhIndex.IP, tableEcmpGroup.ID)
				continue
			}
		}
	}

	return nil
}

func (v pbrAPI) DelObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v pbrAPI) SetObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	var err error
	objPBR := obj.(tai.PBRObj)

	pbrACLName := getACLNameFromPBR(objPBR)
	aclIndex := cdb.ACLIndex{
		ACLName: pbrACLName,
	}

	tableACL, err := cdb.ACLGetByIndex(aclIndex)
	if err != nil {
		log.Warning("ACL for PBR %+v not found\n", objPBR)
		return nil
	}

	sequenceID, err := getSequenceFromPBR(objPBR, 0)
	if err != nil {
		log.Warning("ACL rule get sequence for PBR %+v failed\n", objPBR)
		return nil
	}

	aclRuleIndex := cdb.ACLRuleIndex{
		ACLName:  tableACL.ACLName,
		Sequence: sequenceID,
	}
	tableACLRule, err := cdb.ACLRuleGetByIndex(aclRuleIndex)
	if err != nil {
		log.Warning("ACL rule for PBR %+v not found\n", objPBR)
		return nil
	}

	if len(tableACLRule.RedirectEcmpgroup) != 1 {
		log.Warning("ACL Rule %s for PBR should have one redirect ecmp group action\n", pbrACLName)
		return nil
	}
	tableEcmpGroup, err := cdb.EcmpGroupGetByUUID(tableACLRule.RedirectEcmpgroup[0].GoUUID)
	if err != nil {
		log.Warning("ECMP Group %s not found\n", tableACLRule.RedirectEcmpgroup[0].GoUUID)
		return nil
	}
	ecmpGroupIndex := cdb.EcmpGroupIndex{
		ID: tableEcmpGroup.ID,
	}

	if attrs[tai.PBRAttrNexthopGroup] != nil {
		nhGroup, ok := attrs[tai.PBRAttrNexthopGroup].([]string)
		if !ok {
			log.Warning("Get nh group for PBR %+v failed\n", objPBR)
			return nil
		}

		// remove old nhs
		if len(tableEcmpGroup.NexthopGroup) > 0 {
			cdb.EcmpGroupUpdateNexthopGroupDelvalue(ecmpGroupIndex, tableEcmpGroup.NexthopGroup)
		}
		// set new nh group
		for _, nh := range nhGroup {
			nhIndex := cdb.NexthopIndex{
				IP:         nh,
				Type:       cdb.NexthopTypeVxlan,
				NexthopVrf: objPBR.Vrf,
				Label:      getVniFromVrf(objPBR.Vrf),
			}
			tableNh := cdb.TableNexthop{
				IP:         nh,
				Type:       cdb.NexthopTypeVxlan,
				NexthopVrf: objPBR.Vrf,
				Label:      getVniFromVrf(objPBR.Vrf),
			}
			cdbNh, err := cdb.NexthopGetByIndex(nhIndex)
			if err != nil {
				// add new nexthop
				err = cdb.EcmpGroupUpdateAddNexthopGroup(ecmpGroupIndex, tableNh)
			} else {
				// add nexthop to ecmp group
				err = cdb.EcmpGroupUpdateNexthopGroupAddvalue(ecmpGroupIndex,
					[]libovsdb.UUID{{GoUUID: cdbNh.UUID}})
			}

			if err != nil {
				log.Warning("Add nexthop %s to ecmp group %d failed\n", nhIndex.IP, tableEcmpGroup.ID)
				continue
			}
		}
	}

	return nil
}

func (v pbrAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v pbrAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
