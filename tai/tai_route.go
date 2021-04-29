package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/ebay/libovsdb"
)

// route attr list
const (
	RouteAttrNexthop    = "route_nexthop"
	RouteAttrNhvrf      = "route_nhvrf"
	RouteAttrOutputPort = "route_outputport"
	RouteAttrPolicy     = "route_policy"
)

// RouteObj ...
type RouteObj struct {
	Vrf        string
	IPPrefix   string
	Nexthop    string
	Nhvrf      string
	OutputPort string
	Policy     string
}

func rowToRouteObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableRoute := vtepdb.ConvertRowToRoute(libovsdb.ResultRow(row.Fields))

	obj := RouteObj{
		Vrf:        tableRoute.Vrf,
		IPPrefix:   tableRoute.IPPrefix,
		Nexthop:    tableRoute.Nexthop,
		Nhvrf:      tableRoute.NhVrf,
		OutputPort: tableRoute.OutputPort,
		Policy:     tableRoute.Policy,
	}
	attrs := map[interface{}]interface{}{
		RouteAttrNexthop:    tableRoute.Nexthop,
		RouteAttrNhvrf:      tableRoute.NhVrf,
		RouteAttrOutputPort: tableRoute.OutputPort,
		RouteAttrPolicy:     tableRoute.Policy,
	}
	return obj, attrs
}

func rowToRouteAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		case vtepdb.RouteFieldNexthop:
			if nh, ok := value.(string); ok {
				attrs[RouteAttrNexthop] = nh
			}
		case vtepdb.RouteFieldNhVrf:
			if vrf, ok := value.(string); ok {
				attrs[RouteAttrNhvrf] = vrf
			}
		case vtepdb.RouteFieldOutputPort:
			if outPort, ok := value.(string); ok {
				attrs[RouteAttrOutputPort] = outPort
			}
		case vtepdb.RouteFieldPolicy:
			if policy, ok := value.(string); ok {
				attrs[RouteAttrPolicy] = policy
			}
		}
	}

	return attrs
}
