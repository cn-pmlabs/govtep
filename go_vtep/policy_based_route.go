package govtep

import (
	"fmt"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// PolicyBasedRoute struct
type PolicyBasedRoute struct {
	Type       string
	Vrf        string
	IP         string
	Port       int
	Protocol   string
	LogicalIPs []string
}

func getNexthopForLogicalIP(ip string, vrf string) (string, error) {
	var nexthopIP string

	routeIndex := vtepdb.RouteIndex{
		IPPrefix: ip + "/32",
		Vrf:      vrf,
	}
	tableRoute, err := vtepdb.RouteGetByIndex(routeIndex)
	if err != nil {
		log.Warning("getNexthopForLogicalIP %+s vrf %s route not found\n", ip, vrf)
		return nexthopIP, fmt.Errorf("not found")
	}

	nexthopIP = tableRoute.Nexthop
	return nexthopIP, nil
}

func getNexthopGroupRemote() []string {
	var nhs []string
	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))
	rows, num := vtepdb.LocatorGet(conditions)
	if num > 0 {
		for _, row := range rows {
			dbLocator := vtepdb.ConvertRowToLocator(row)
			if dbLocator.LocalLocator == true {
				for ip := range dbLocator.RmacMap {
					nhs = append(nhs, ip.(string))
				}
				break
			}
		}
	}

	return nhs
}

func pbrNhGroupContains(nhGroup []string, nh string) bool {
	for i := 0; i < len(nhGroup); i++ {
		if nhGroup[i] == nh {
			return true
		}
	}
	return false
}

func pbrNhUpdateCb() error {
	var err error
	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: vtepdb.InvalidUUID}))

	rows, num := vtepdb.PolicyBasedRouteGet(conditions)
	if num > 0 {
		for _, row := range rows {
			var nhGroup []string
			tablePBR := vtepdb.ConvertRowToPolicyBasedRoute(row)
			for _, ip := range tablePBR.LogicalIps {
				nexthopIP, err := getNexthopForLogicalIP(ip, tablePBR.Vrf)
				if err != nil {
					continue
				}

				if false == pbrNhGroupContains(nhGroup, nexthopIP) {
					nhGroup = append(nhGroup, nexthopIP)
				}
			}

			nhOp := make(map[string]string)
			for _, nh := range tablePBR.NhGroup {
				nhOp[nh] = odbc.OpDelete
			}
			for _, nh := range nhGroup {
				if _, ok := nhOp[nh]; ok {
					nhOp[nh] = "keep"
				} else {
					nhOp[nh] = odbc.OpInsert
				}
			}

			pbrIndex := vtepdb.PolicyBasedRouteUUIDIndex{
				UUID: tablePBR.UUID,
			}
			for nh, op := range nhOp {
				if op == odbc.OpInsert {
					err = vtepdb.PolicyBasedRouteUpdateNhGroupAddvalue(pbrIndex, []string{nh})
				} else if op == odbc.OpDelete {
					err = vtepdb.PolicyBasedRouteUpdateNhGroupDelvalue(pbrIndex, []string{nh})
				}
				if err != nil {
					log.Warning("Update nexthop group for PBR %s failed\n", pbrIndex.UUID)
					continue
				}
			}
		}
	}

	return nil
}

func policyBasedRouteAdd(pbr PolicyBasedRoute) error {
	log.Info("policyBasedRouteAdd %+v\n", pbr)

	tablePBR := vtepdb.TablePolicyBasedRoute{
		Type:       pbr.Type,
		IP:         pbr.IP,
		Port:       []int{pbr.Port},
		Vrf:        pbr.Vrf,
		NhVrf:      pbr.Vrf,
		LogicalIps: pbr.LogicalIPs,
	}

	if pbr.Protocol != "" {
		tablePBR.Protocol = []string{pbr.Protocol}
	} else {
		tablePBR.Protocol = []string{vtepdb.PolicyBasedRouteProtocolIgnore}
	}

	if tablePBR.Type == vtepdb.PolicyBasedRouteTypeSnat {
		// LogicalIps might be a network (e.g 192.168.1.0/24)
		tablePBR.NhGroup = getNexthopGroupRemote()
	} else {
		for _, ip := range pbr.LogicalIPs {
			nexthopIP, err := getNexthopForLogicalIP(ip, pbr.Vrf)
			if err != nil {
				continue
			}

			if false == pbrNhGroupContains(tablePBR.NhGroup, nexthopIP) {
				tablePBR.NhGroup = append(tablePBR.NhGroup, nexthopIP)
			}
		}
	}

	log.Info("tablePBR %+v\n", tablePBR)
	vrfIndex := vtepdb.VrfIndex{
		Name: pbr.Vrf,
	}
	err := vtepdb.VrfUpdateAddPbr(vrfIndex, tablePBR)
	if err != nil {
		log.Info("PBR %+v already existed", tablePBR)
		return nil
	}

	return err
}

func policyBasedRouteDel(pbr PolicyBasedRoute) error {
	log.Info("policyBasedRouteDel %+v\n", pbr)

	pbrIndex := vtepdb.PolicyBasedRouteIndex{
		Type: pbr.Type,
		IP:   pbr.IP,
		Port: pbr.Port,
		Vrf:  pbr.Vrf,
	}

	if pbr.Protocol != "" {
		pbrIndex.Protocol = pbr.Protocol
	} else {
		pbrIndex.Protocol = vtepdb.PolicyBasedRouteProtocolIgnore
	}

	tablePBR, err := vtepdb.PolicyBasedRouteGetByIndex(pbrIndex)
	if err != nil {
		log.Warning("PolicyBasedRoute %+v not found\n", pbrIndex)
		return fmt.Errorf("PBR not found")
	}

	vrfIndex := vtepdb.VrfIndex{
		Name: pbr.Vrf,
	}
	err = vtepdb.VrfUpdatePbrDelvalue(vrfIndex, []libovsdb.UUID{{GoUUID: tablePBR.UUID}})
	if err != nil {
		log.Warning("PBR %+v del failed", pbrIndex)
	}

	return err
}
