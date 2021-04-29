package driver

import (
	"fmt"

	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"

	"github.com/ebay/libovsdb"
)

type routeAPI struct {
	moduleID int
}

var routeAPIs = routeAPI{
	moduleID: tai.ObjectIDRoute,
}

func (v routeAPI) CreateObject(obj interface{}) error {
	objRoute := obj.(tai.RouteObj)

	routeCfg := cdb.TableStaticRoute{
		Vrf:  objRoute.Vrf,
		IP:   objRoute.IPPrefix,
		Flag: []string{cdb.StaticRouteFlagVxlan},
	}

	nhMap := make(map[interface{}]interface{})
	nhKey := "vrfname:" + objRoute.Vrf + ",ip:" + objRoute.Nexthop + ",port:" + "Bd" + objRoute.Vrf
	nhMap[nhKey] = "label:,onlink:,color:"
	routeCfg.Nexthop = nhMap

	vrfIndex := cdb.VrfIndex{
		Name: objRoute.Vrf,
	}
	_, err := cdb.VrfGetByIndex(vrfIndex)
	if err != nil {
		log.Warning("[Driver] Static route %+v create failed because of invalid vrf", routeCfg)
	}

	routeIndex := cdb.StaticRouteIndex{
		Vrf: objRoute.Vrf,
		IP:  objRoute.IPPrefix,
	}
	if _, err := cdb.StaticRouteGetByIndex(routeIndex); err == nil {
		log.Info("[Driver] Route %+v already exist", routeIndex)
		return nil
	}

	err = cdb.VrfUpdateAddRoute(vrfIndex, routeCfg)
	if err != nil {
		return err
	}

	return nil
}

func (v routeAPI) RemoveObject(obj interface{}) error {
	objRoute := obj.(tai.RouteObj)

	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("vrf", "==", objRoute.Vrf))
	conditions = append(conditions, libovsdb.
		NewCondition("ip", "==", objRoute.IPPrefix))
	rows, num := cdb.StaticRouteGet(conditions)

	if num != 1 {
		log.Warning("[Driver] Static route %+v not exist", objRoute)
		return fmt.Errorf("[Driver] Static route %+v not exist", objRoute)
	}
	tableRoute := cdb.ConvertRowToStaticRoute(rows[0])

	/*routeIndex := cdb.StaticRouteIndex{
		Vrf: objRoute.Vrf,
		IP:  objRoute.IPPrefix,
	}
	tableRoute, err := cdb.StaticRouteGetByIndex(routeIndex)
	if err != nil {
		log.Warning("[Driver] Static route %+v not exist", routeIndex)
	}*/

	vrfIndex := cdb.VrfIndex{
		Name: objRoute.Vrf,
	}
	_, err := cdb.VrfGetByIndex(vrfIndex)
	if err != nil {
		log.Warning("[Driver] Static route %+v remove failed because of invalid vrf", objRoute)
	}

	routeUpdate := []libovsdb.UUID{{GoUUID: tableRoute.UUID}}
	err = cdb.VrfUpdateRouteDelvalue(vrfIndex, routeUpdate)
	if err != nil {
		return err
	}

	return nil
}

func (v routeAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objRoute := obj.(tai.RouteObj)

	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("vrf", "==", objRoute.Vrf))
	conditions = append(conditions, libovsdb.
		NewCondition("ip", "==", objRoute.IPPrefix))
	rows, num := cdb.StaticRouteGet(conditions)

	if num != 1 {
		log.Warning("[Driver] Static route %+v not exist", objRoute)
		return fmt.Errorf("[Driver] Static route %+v not exist", objRoute)
	}
	tableRoute := cdb.ConvertRowToStaticRoute(rows[0])

	routeIndex := cdb.StaticRouteUUIDIndex{
		UUID: tableRoute.UUID,
	}

	for attr, attrValue := range attrs {
		switch attr {
		case tai.RouteAttrNexthop:
			if nh, ok := attrValue.(string); ok {
				//cdb.StaticRouteSetField(routeIndex, cdb.StaticRouteFieldNexthop, nh)
				nhMap := make(map[interface{}]interface{})
				nhKey := "vrfname:" + objRoute.Vrf + ",ip:" + nh + ",port:" + "Bd" + objRoute.Vrf
				nhMap[nhKey] = "label:,onlink:,color:"
				cdb.StaticRouteSetField(routeIndex, cdb.StaticRouteFieldNexthop, nhMap)
			}
		}
	}

	return nil
}

func (v routeAPI) DelObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v routeAPI) SetObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objRoute := obj.(tai.RouteObj)

	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("vrf", "==", objRoute.Vrf))
	conditions = append(conditions, libovsdb.
		NewCondition("ip", "==", objRoute.IPPrefix))
	rows, num := cdb.StaticRouteGet(conditions)

	if num != 1 {
		log.Warning("[Driver] Static route %+v not exist", objRoute)
		return fmt.Errorf("[Driver] Static route %+v not exist", objRoute)
	}
	tableRoute := cdb.ConvertRowToStaticRoute(rows[0])

	routeIndex := cdb.StaticRouteUUIDIndex{
		UUID: tableRoute.UUID,
	}

	for attr, attrValue := range attrs {
		switch attr {
		case tai.RouteAttrNexthop:
			if nh, ok := attrValue.(string); ok {
				nhMap := make(map[interface{}]interface{})
				nhKey := "vrfname:" + objRoute.Vrf + ",ip:" + nh + ",port:" + "Bd" + objRoute.Vrf
				nhMap[nhKey] = "label:,onlink:,color:"

				cdb.StaticRouteSetField(routeIndex, cdb.StaticRouteFieldNexthop, nhMap)
			}
		}
	}

	return nil
}

func (v routeAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v routeAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
