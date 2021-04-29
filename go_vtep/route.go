package govtep

import (
	"fmt"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"

	"github.com/ebay/libovsdb"
)

// Route ...
type Route struct {
	UUID          string
	Vrf           string //VTEP DB Vrf uuid
	IPPrefix      string
	Nexthop       string
	NhVrf         string //Nexthop vrf name, get from outputport patch peer vrf
	OutputPort    string //VTEP DB L3Port name
	RemoteLocator string //VTEP DB Locator uuid
	Policy        string //either dst−ip or src−ip
}

// route policy type
const (
	RoutePolicyDefault string = "dst-ip"
	RoutePolicySource  string = "src-ip"
)

func routeNhUpdateForLocator(locator string, nh string) error {
	var err error

	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition(vtepdb.RouteFieldRemoteLocator, "==", locator))
	rows, num := vtepdb.RouteGet(conditions)
	if num > 0 {
		for _, row := range rows {
			tableRoute := vtepdb.ConvertRowToRoute(row)
			if tableRoute.Nexthop != nh {
				rtIndex := vtepdb.RouteUUIDIndex{
					UUID: tableRoute.UUID,
				}
				err = vtepdb.RouteSetField(rtIndex, vtepdb.RouteFieldNexthop, nh)
				if err != nil {
					log.Error("Route %s update nexthop %s failed\n", tableRoute.IPPrefix, nh)
				}
			}
		}
	}

	return err
}

func routeCreate(route Route) error {
	var err error

	vrfIndex := vtepdb.VrfIndex{
		Name: route.Vrf,
	}

	_, err = vtepdb.VrfGetByIndex(vrfIndex)
	if err != nil {
		return fmt.Errorf("Vrf %s not found when create route", vrfIndex.Name)
	}

	rtIndex := vtepdb.RouteIndex{
		Vrf:      route.Vrf,
		IPPrefix: route.IPPrefix,
	}
	_, err = vtepdb.RouteGetByIndex(rtIndex)
	if err == nil {
		log.Info("Route %+v already exist\n", rtIndex)
		return nil
	}

	tableRoute := vtepdb.TableRoute{
		Vrf:           route.Vrf,
		IPPrefix:      route.IPPrefix,
		Nexthop:       route.Nexthop,
		NhVrf:         route.NhVrf,
		OutputPort:    route.OutputPort,
		RemoteLocator: route.RemoteLocator,
		Policy:        route.Policy,
	}

	err = vtepdb.VrfUpdateAddRoute(vrfIndex, tableRoute)
	if err == nil {
		// update lb nexthop group member
		pbrNhUpdateCb()
	}

	return err
}

func routeRemove(route Route) error {
	var err error

	vrfIndex := vtepdb.VrfIndex{
		Name: route.Vrf,
	}

	_, err = vtepdb.VrfGetByIndex(vrfIndex)
	if err != nil {
		return fmt.Errorf("Vrf %s not found when remove route", vrfIndex.Name)
	}

	rtIndex := vtepdb.RouteIndex{
		Vrf:      route.Vrf,
		IPPrefix: route.IPPrefix,
	}
	tableRoute, err := vtepdb.RouteGetByIndex(rtIndex)
	if err != nil {
		return fmt.Errorf("Route %+v not found", rtIndex)
	}

	err = vtepdb.VrfUpdateRouteDelvalue(vrfIndex, []libovsdb.UUID{{GoUUID: tableRoute.UUID}})
	if err == nil {
		// update lb nexthop group member
		pbrNhUpdateCb()
	}

	return err
}

func routeSetCreate(routes []Route) error {
	for _, route := range routes {
		err := routeCreate(route)
		if err != nil {
			log.Warning("%v\n", err)
		}
	}

	return nil
}

func routeSetRemove(routes []Route) error {
	for _, route := range routes {
		err := routeRemove(route)
		if err != nil {
			log.Warning("%v\n", err)
		}
	}

	return nil
}
