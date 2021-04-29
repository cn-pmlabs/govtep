package govtep

import (
	"fmt"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnnb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnnorthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

func getVrfFromLR(lrUUID string) (string, error) {
	var conditions []interface{}

	conditions = append(conditions, libovsdb.
		NewCondition("lrname", "==", lrUUID))
	rows, num := vtepdb.VrfGet(conditions)
	if num != 1 {
		return "", fmt.Errorf("Vrf for lr %s not exist", lrUUID)
	}
	tableVrf := vtepdb.ConvertRowToVrf(rows[0])

	return tableVrf.Name, nil
}

func logicalRouterNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate, UUID string) {
	var err error

	switch op {
	case odbc.OpInsert:
		err = logicalRouterCreate(rowUpdate.New, UUID)
	case odbc.OpDelete:
		err = logicalRouterRemove(rowUpdate.Old, UUID)
	case odbc.OpUpdate:
		err = logicalRouterUpdate(rowUpdate.New, rowUpdate.Old, UUID)
	}

	if err != nil {
		log.Error("logicalRouterNotifyUpdate op %s failed: %v\n", op, err)
		return
	}
}

func logicalRouterCreate(lrRow libovsdb.Row, UUID string) error {
	tableLR := ovnnb.ConvertRowToLogicalRouter(lrRow.Fields)
	tableLR.UUID = UUID

	// LR create with load balancer
	for _, lb := range tableLR.LoadBalancer {
		log.Info("Logical switch %s create with LB %s\n", tableLR.Name, lb.GoUUID)
		err := logicalRouterAddLoadBalancer(tableLR, lb)
		if err != nil {
			log.Warning("Logical switch %s add LB %s failed %v\n", tableLR.Name, lb.GoUUID, err)
		}
	}

	for _, nat := range tableLR.Nat {
		log.Info("Logical switch %s create with nat %s\n", tableLR.Name, nat.GoUUID)
		err := logicalRouterAddNat(tableLR, nat)
		if err != nil {
			log.Warning("Logical switch %s add Nat %s failed %v\n", tableLR.Name, nat.GoUUID, err)
		}
	}

	return nil
}

func logicalRouterRemove(lrRow libovsdb.Row, UUID string) error {
	// remove applied_lr from lb, no need to remove pbr when vrf deleted
	var conditions []interface{}
	conditions = append(conditions, libovsdb.
		NewCondition("_uuid", "!=", libovsdb.UUID{GoUUID: ovnnb.InvalidUUID}))
	rows, num := ovnnb.LoadBalancerGet(conditions)
	if num > 0 {
		for _, row := range rows {
			tableLB := ovnnb.ConvertRowToLoadBalancer(row)

			lbIndex := ovnnb.LoadBalancerUUIDIndex{
				UUID: tableLB.UUID,
			}
			err := ovnnb.LoadBalancerUpdateAppliedLrDelvalue(lbIndex, []string{UUID})
			if err != nil {
				log.Warning("LB %s remove appied lr %s failed\n", tableLB.UUID, UUID)
			}
		}
	}

	return nil
}

func logicalRouterUpdate(newrow libovsdb.Row, oldrow libovsdb.Row, UUID string) error {
	var err error

	for field, oldValue := range oldrow.Fields {
		switch field {
		case ovnnb.LogicalRouterFieldLoadBalancer:
			err = logicalRouterUpdateLoadBalancer(newrow, oldValue, UUID)
		case ovnnb.LogicalRouterFieldNat:
			err = logicalRouterUpdateNat(newrow, oldValue, UUID)
		case ovnnb.LogicalRouterFieldStaticRoutes:
			//err = logicalRouterUpdateStaticRoutes(newrow, oldValue)
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
