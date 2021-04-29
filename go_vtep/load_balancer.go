package govtep

import (
	"fmt"
	"strconv"
	"strings"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnnb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnnorthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

func loadBalancerPreparePBR(vip interface{}, backends interface{}) (string, int, []string, error) {
	var (
		externalIP  string
		port        int
		logicalIPs  []string
		err         error
		backendsStr string
		backendsSet []string
	)

	vipStr, ok := vip.(string)
	if false == ok {
		err = fmt.Errorf("invalid vip %v", vip)
		goto errOut
	}
	backendsStr, ok = backends.(string)
	if false == ok {
		err = fmt.Errorf("invalid backends %v", backends)
		goto errOut
	}

	externalIP, port, err = loadBalancerParserIP(vipStr)
	if err != nil {
		err = fmt.Errorf("invalid vip %v", vip)
		goto errOut
	}

	backendsSet = strings.Split(backendsStr, ",")
	for _, backend := range backendsSet {
		logicalIP, _, err := loadBalancerParserIP(backend)
		if err != nil {
			err = fmt.Errorf("invalid backends %v", backends)
			goto errOut
		}
		logicalIPs = append(logicalIPs, logicalIP)
	}

errOut:
	return externalIP, port, logicalIPs, err
}

func loadBalancerParserIP(vip string) (string, int, error) {
	var ip string
	var port int

	ipport := strings.Split(vip, ":")
	if len(ipport) == 2 {
		p, err := strconv.Atoi(ipport[1])
		if err != nil {
			return ip, port, fmt.Errorf("invalid port %s", ipport[1])
		}
		// can't assign port to atoi
		port = p
	}
	ip = ipport[0]

	return ip, port, nil
}

func loadBalancerNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate, UUID string) {
	var err error

	switch op {
	case odbc.OpInsert:
		err = loadBalancerCreate(rowUpdate.New, UUID)
	case odbc.OpDelete:
		err = loadBalancerRemove(rowUpdate.Old, UUID)
	case odbc.OpUpdate:
		err = loadBalancerUpdate(rowUpdate.New, rowUpdate.Old, UUID)
	}

	if err != nil {
		log.Error("loadBalancerNotifyUpdate op %s failed: %v\n", op, err)
		return
	}
}

func loadBalancerCreate(lbRow libovsdb.Row, UUID string) error {
	tableLB := ovnnb.ConvertRowToLoadBalancer(lbRow.Fields)
	tableLB.UUID = UUID

	// don't need process load balancer creation before applied on LR for now

	return nil
}

func loadBalancerRemove(lbRow libovsdb.Row, UUID string) error {
	// lb remove, iterate applied lr to remove pbr
	tableLB := ovnnb.ConvertRowToLoadBalancer(lbRow.Fields)
	tableLB.UUID = UUID

	for vip, backends := range tableLB.Vips {
		externalIP, port, logicalIPs, err := loadBalancerPreparePBR(vip, backends)
		if err != nil {
			log.Warning("LB %s vip %+v add prepare failed %v\n",
				UUID, tableLB.Vips, err)
		}

		for _, lrUUID := range tableLB.AppliedLr {
			vrf, err := getVrfFromLR(lrUUID)
			if err != nil {
				log.Error("LR %s vtepdb.vrf not found\n", lrUUID)
				continue
			}

			pbr := PolicyBasedRoute{
				Vrf:        vrf,
				IP:         externalIP,
				Port:       port,
				LogicalIPs: logicalIPs,
				Type:       vtepdb.PolicyBasedRouteTypeLb,
			}
			if len(tableLB.Protocol) == 1 {
				pbr.Protocol = tableLB.Protocol[0]
			} else {
				pbr.Protocol = ovnnb.LoadBalancerProtocolTCP
			}
			if pbr.Port == 0 {
				pbr.Protocol = ovnnb.LoadBalancerProtocolTCP
			}

			// remove pbr
			err = policyBasedRouteDel(pbr)
			if err != nil {
				log.Warning("policyBasedRoute %+v Del failed\n", pbr)
			}
		}
	}

	return nil
}

func loadBalancerUpdate(newrow libovsdb.Row, oldrow libovsdb.Row, UUID string) error {
	var err error

	for field, oldValue := range oldrow.Fields {
		switch field {
		case ovnnb.LoadBalancerFieldVips:
			err = loadBalancerUpdateVips(newrow, oldValue, UUID)
		case ovnnb.LoadBalancerFieldProtocol:
			err = loadBalancerUpdateProtocol(newrow, oldValue, UUID)
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

func loadBalancerUpdateVips(lbRow libovsdb.Row, oldValue interface{}, UUID string) error {
	tableLB := ovnnb.ConvertRowToLoadBalancer(lbRow.Fields)
	tableLB.UUID = UUID

	if len(tableLB.Protocol) == 0 {
		tableLB.Protocol = []string{ovnnb.LoadBalancerProtocolTCP}
	}

	oldVips := make(map[interface{}]interface{})
	if _, ok := oldValue.(libovsdb.OvsMap); ok {
		oldVips = oldValue.(libovsdb.OvsMap).GoMap
	}
	log.Info("LB %s vip update from %+vto %+v\n", tableLB.Name, oldVips, tableLB.Vips)

	// get Load balancer vip update operation
	vipOp := make(map[interface{}]string)
	for vip := range oldVips {
		vipOp[vip] = odbc.OpDelete
	}
	for vip, newBackends := range tableLB.Vips {
		if _, ok := vipOp[vip]; ok {
			if oldVips[vip].(string) == newBackends.(string) {
				vipOp[vip] = "keep"
			} else {
				vipOp[vip] = odbc.OpUpdate
			}
		} else {
			vipOp[vip] = odbc.OpInsert
		}
	}

	for vip, op := range vipOp {
		var backendsAdd interface{}
		var backendsDel interface{}

		if op == odbc.OpDelete {
			backendsDel = oldVips[vip]
		} else if op == odbc.OpInsert {
			backendsAdd = tableLB.Vips[vip]
		} else if op == odbc.OpUpdate {
			backendsDel = oldVips[vip]
			backendsAdd = tableLB.Vips[vip]
		} else {
			continue
		}

		for _, lrUUID := range tableLB.AppliedLr {
			vrf, err := getVrfFromLR(lrUUID)
			if err != nil {
				log.Error("LR %s vtepdb.vrf not found\n", lrUUID)
				continue
			}

			if backendsDel != nil {
				externalIP, port, logicalIPs, err := loadBalancerPreparePBR(vip, backendsDel)
				if err != nil {
					log.Warning("LB %s vip %+v del prepare failed %v\n",
						UUID, tableLB.Vips, err)
				}
				pbr := PolicyBasedRoute{
					Vrf:        vrf,
					IP:         externalIP,
					Port:       port,
					LogicalIPs: logicalIPs,
					Type:       vtepdb.PolicyBasedRouteTypeLb,
				}
				if len(tableLB.Protocol) == 1 {
					pbr.Protocol = tableLB.Protocol[0]
				} else {
					pbr.Protocol = ovnnb.LoadBalancerProtocolTCP
				}
				if pbr.Port == 0 {
					pbr.Protocol = ovnnb.LoadBalancerProtocolTCP
				}
				err = policyBasedRouteDel(pbr)
				if err != nil {
					log.Warning("policyBasedRoute %+v Del failed\n", pbr)
				}
			}

			if backendsAdd != nil {
				externalIP, port, logicalIPs, err := loadBalancerPreparePBR(vip, backendsAdd)
				if err != nil {
					log.Warning("LB %s vip %+v add prepare failed %v\n",
						UUID, tableLB.Vips, err)
				}
				pbr := PolicyBasedRoute{
					Vrf:        vrf,
					IP:         externalIP,
					Port:       port,
					LogicalIPs: logicalIPs,
					Type:       vtepdb.PolicyBasedRouteTypeLb,
				}
				if len(tableLB.Protocol) == 1 {
					pbr.Protocol = tableLB.Protocol[0]
				} else {
					pbr.Protocol = ovnnb.LoadBalancerProtocolTCP
				}
				if pbr.Port == 0 {
					pbr.Protocol = ovnnb.LoadBalancerProtocolTCP
				}
				err = policyBasedRouteAdd(pbr)
				if err != nil {
					log.Warning("policyBasedRoute %+v Add failed\n", pbr)
				}
			}
		}
	}

	return nil
}

func loadBalancerUpdateProtocol(lbRow libovsdb.Row, oldValue interface{}, UUID string) error {
	tableLB := ovnnb.ConvertRowToLoadBalancer(lbRow.Fields)
	tableLB.UUID = UUID

	if len(tableLB.AppliedLr) == 0 {
		log.Info("LB %s not applied to any LR, ignored\n", UUID)
		return nil
	}

	oldProtocol, ok := oldValue.(string)
	if !ok {
		oldProtocol = ovnnb.LoadBalancerProtocolTCP
	}

	if len(tableLB.Protocol) == 0 {
		tableLB.Protocol = []string{ovnnb.LoadBalancerProtocolTCP}
	}

	if oldProtocol == tableLB.Protocol[0] {
		log.Info("LB %s Protocol not updated\n", UUID)
	}
	log.Info("LB %s UpdateProtocol from %s to %s\n", UUID, oldProtocol, tableLB.Protocol[0])

	// need to remove old pbr with port, then add new pbr with updated protocol
	for vip, backends := range tableLB.Vips {
		externalIP, port, logicalIPs, err := loadBalancerPreparePBR(vip, backends)
		if err != nil {
			log.Warning("LB %s vip %+v add prepare failed %v\n",
				UUID, tableLB.Vips, err)
		}

		// LB protocol should be tcp when vip without port
		if 0 == port {
			log.Warning("protocol for vip without port always tcp, ignored\n")
			continue
		}

		for _, lrUUID := range tableLB.AppliedLr {
			vrf, err := getVrfFromLR(lrUUID)
			if err != nil {
				log.Error("LR %s vtepdb.vrf not found\n", lrUUID)
				continue
			}

			pbr := PolicyBasedRoute{
				Vrf:        vrf,
				IP:         externalIP,
				Port:       port,
				LogicalIPs: logicalIPs,
				Type:       vtepdb.PolicyBasedRouteTypeLb,
			}

			// remove pbr
			pbr.Protocol = oldProtocol
			err = policyBasedRouteDel(pbr)
			if err != nil {
				log.Warning("policyBasedRoute %+v Del failed\n", pbr)
			}
			// add new pbr
			pbr.Protocol = tableLB.Protocol[0]
			err = policyBasedRouteAdd(pbr)
			if err != nil {
				log.Warning("policyBasedRoute %+v Add failed\n", pbr)
			}
		}
	}

	return nil
}

func logicalRouterUpdateLoadBalancer(lrRow libovsdb.Row, oldValue interface{}, UUID string) error {
	tableLR := ovnnb.ConvertRowToLogicalRouter(lrRow.Fields)
	tableLR.UUID = UUID

	var oldLBs []libovsdb.UUID
	switch oldValue.(type) {
	case libovsdb.UUID:
		oldLBs = append(oldLBs, oldValue.(libovsdb.UUID))
	case libovsdb.OvsSet:
		if LBs, ok := oldValue.(libovsdb.OvsSet); ok {
			for _, LB := range LBs.GoSet {
				LBUUID, ok := LB.(libovsdb.UUID)
				if ok {
					oldLBs = append(oldLBs, LBUUID)
				}
			}
		}
	}

	lbOp := make(map[libovsdb.UUID]string)

	for _, LB := range oldLBs {
		lbOp[LB] = odbc.OpDelete
	}
	for _, LB := range tableLR.LoadBalancer {
		if _, ok := lbOp[LB]; ok {
			lbOp[LB] = "keep"
		} else {
			lbOp[LB] = odbc.OpInsert
		}
	}

	for LB, op := range lbOp {
		switch op {
		case odbc.OpInsert:
			err := logicalRouterAddLoadBalancer(tableLR, LB)
			if err != nil {
				log.Warning("LR %s add LB %s failed %v\n", tableLR.Name, LB.GoUUID, err)
			}
		case odbc.OpDelete:
			err := logicalRouterDelLoadBalancer(tableLR, LB)
			if err != nil {
				log.Warning("LR %s del LB %s failed %v\n", tableLR.Name, LB.GoUUID, err)
			}
		}
	}

	return nil
}

func logicalRouterAddLoadBalancer(tableLR ovnnb.TableLogicalRouter, LB libovsdb.UUID) error {
	tableLB, err := ovnnb.LoadBalancerGetByUUID(LB.GoUUID)
	if err != nil {
		return fmt.Errorf("LB %s not found", LB.GoUUID)
	}

	protocol := ovnnb.LoadBalancerProtocolTCP
	if len(tableLB.Protocol) == 1 {
		protocol = tableLB.Protocol[0]
	}

	vrf, err := getVrfFromLR(tableLR.UUID)
	if err != nil {
		return fmt.Errorf("LR %s vtepdb.vrf not found", tableLR.Name)
	}

	for vip, backends := range tableLB.Vips {
		externalIP, port, logicalIPs, err := loadBalancerPreparePBR(vip, backends)
		if err != nil {
			log.Warning("LR %s add LB %s vip %+v add prepare failed %v\n",
				tableLR.Name, LB.GoUUID, tableLB.Vips, err)
		}

		pbr := PolicyBasedRoute{
			Vrf:        vrf,
			IP:         externalIP,
			Port:       port,
			Protocol:   protocol,
			LogicalIPs: logicalIPs,
			Type:       vtepdb.PolicyBasedRouteTypeLb,
		}
		// LB protocol should be tcp when vip without port
		if 0 == pbr.Port {
			pbr.Protocol = ovnnb.LoadBalancerProtocolTCP
		}

		err = policyBasedRouteAdd(pbr)
		if err != nil {
			log.Warning("policyBasedRoute %+v Add failed\n", pbr)
		}
	}

	lbIndex := ovnnb.LoadBalancerUUIDIndex{
		UUID: LB.GoUUID,
	}
	err = ovnnb.LoadBalancerUpdateAppliedLrAddvalue(lbIndex, []string{tableLR.UUID})
	if err != nil {
		log.Warning("LB %s add appied lr %s failed\n", LB.GoUUID, tableLR.Name)
	}

	return nil
}

func logicalRouterDelLoadBalancer(tableLR ovnnb.TableLogicalRouter, LB libovsdb.UUID) error {
	tableLB, err := ovnnb.LoadBalancerGetByUUID(LB.GoUUID)
	if err != nil {
		return fmt.Errorf("LB %s has been removed", LB.GoUUID)
	}

	protocol := "tcp"
	if len(tableLB.Protocol) == 1 {
		protocol = tableLB.Protocol[0]
	}

	vrf, err := getVrfFromLR(tableLR.UUID)
	if err != nil {
		return fmt.Errorf("LR %s vtepdb.vrf not found", tableLR.Name)
	}

	for vip, backends := range tableLB.Vips {
		externalIP, port, logicalIPs, err := loadBalancerPreparePBR(vip, backends)
		if err != nil {
			log.Warning("LR %s del LB %s vip %+v add prepare failed %v\n",
				tableLR.Name, LB.GoUUID, tableLB.Vips, err)
		}

		log.Info("PBR del external %s:%d logicalIPs %v protocol %v vrf %s\n",
			externalIP, port, logicalIPs, protocol, vrf)

		pbr := PolicyBasedRoute{
			Vrf:        vrf,
			IP:         externalIP,
			Port:       port,
			Protocol:   protocol,
			LogicalIPs: logicalIPs,
			Type:       vtepdb.PolicyBasedRouteTypeLb,
		}
		if 0 == pbr.Port {
			pbr.Protocol = ovnnb.LoadBalancerProtocolTCP
		}

		err = policyBasedRouteDel(pbr)
		if err != nil {
			log.Warning("policyBasedRoute %+v Del failed\n", pbr)
		}
	}

	lbIndex := ovnnb.LoadBalancerUUIDIndex{
		UUID: LB.GoUUID,
	}
	err = ovnnb.LoadBalancerUpdateAppliedLrDelvalue(lbIndex, []string{tableLR.UUID})
	if err != nil {
		log.Warning("LB %s remove appied lr %s failed\n", LB.GoUUID, tableLR.Name)
	}

	return nil
}
