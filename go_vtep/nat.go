package govtep

import (
	"fmt"

	ovnnb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnnorthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// ovnnb.NAT is non-root table, create and remove msg in LR.nat update
func logicalRouterUpdateNat(lrRow libovsdb.Row, oldValue interface{}, UUID string) error {
	tableLR := ovnnb.ConvertRowToLogicalRouter(lrRow.Fields)
	tableLR.UUID = UUID

	var oldNats []libovsdb.UUID
	switch oldValue.(type) {
	case libovsdb.UUID:
		oldNats = append(oldNats, oldValue.(libovsdb.UUID))
	case libovsdb.OvsSet:
		if nats, ok := oldValue.(libovsdb.OvsSet); ok {
			for _, nat := range nats.GoSet {
				LBUUID, ok := nat.(libovsdb.UUID)
				if ok {
					oldNats = append(oldNats, LBUUID)
				}
			}
		}
	}

	natOp := make(map[libovsdb.UUID]string)

	for _, nat := range oldNats {
		natOp[nat] = odbc.OpDelete
	}
	for _, nat := range tableLR.Nat {
		if _, ok := natOp[nat]; ok {
			natOp[nat] = "keep"
		} else {
			natOp[nat] = odbc.OpInsert
		}
	}

	for nat, op := range natOp {
		switch op {
		case odbc.OpInsert:
			err := logicalRouterAddNat(tableLR, nat)
			if err != nil {
				log.Warning("LR %s add nat %s failed %v\n", tableLR.Name, nat.GoUUID, err)
			}
		case odbc.OpDelete:
			err := logicalRouterDelNat(tableLR, nat)
			if err != nil {
				log.Warning("LR %s del nat %s failed %v\n", tableLR.Name, nat.GoUUID, err)
			}
		}
	}

	return nil
}

func logicalRouterAddNat(tableLR ovnnb.TableLogicalRouter, nat libovsdb.UUID) error {
	tableNat, err := ovnnb.NatGetByUUID(nat.GoUUID)
	if err != nil {
		return fmt.Errorf("nat %s not found", nat.GoUUID)
	}

	vrf, err := getVrfFromLR(tableLR.UUID)
	if err != nil {
		return fmt.Errorf("LR %s vtepdb.vrf not found", tableLR.Name)
	}

	pbr := PolicyBasedRoute{
		Vrf:        vrf,
		IP:         tableNat.ExternalIP,
		LogicalIPs: []string{tableNat.LogicalIP},
		Type:       tableNat.Type,
	}

	err = policyBasedRouteAdd(pbr)
	if err != nil {
		log.Warning("policyBasedRoute %+v Add failed\n", pbr)
	}

	datapath := make(map[interface{}]interface{})
	datapath["datapath"] = vrf
	tableNat.ExternalIds = datapath
	natIndex := ovnnb.NatUUIDIndex{
		UUID: nat.GoUUID,
	}
	err = ovnnb.NatUpdateExternalIdsSetkey(natIndex, datapath)

	return nil
}

func logicalRouterDelNat(tableLR ovnnb.TableLogicalRouter, nat libovsdb.UUID) error {
	/*tableNat, err := ovnnb.NatGetByUUID(nat.GoUUID)
	if err != nil {
		return fmt.Errorf("nat %s not found", nat.GoUUID)
	}

	vrf, err := getVrfFromLR(tableLR.UUID)
	if err != nil {
		return fmt.Errorf("LR %s vtepdb.vrf not found", tableLR.Name)
	}

	pbr := PolicyBasedRoute{
		Vrf:        vrf,
		IP:         tableNat.ExternalIP,
		LogicalIPs: []string{tableNat.LogicalIP},
		Type:       tableNat.Type,
	}

	err = policyBasedRouteDel(pbr)
	if err != nil {
		log.Warning("policyBasedRoute %+v Add failed\n", pbr)
	}*/

	return nil
}

func natNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate, UUID string) {
	var err error

	switch op {
	case odbc.OpInsert:
		err = natCreate(rowUpdate.New, UUID)
	case odbc.OpDelete:
		err = natRemove(rowUpdate.Old, UUID)
	}

	if err != nil {
		log.Error("natNotifyUpdate op %s failed: %v\n", op, err)
		return
	}
}

func natCreate(lbRow libovsdb.Row, UUID string) error {
	return nil
}

func natRemove(lbRow libovsdb.Row, UUID string) error {
	tableNat := ovnnb.ConvertRowToNat(lbRow.Fields)
	tableNat.UUID = UUID

	vrf, ok := tableNat.ExternalIds["datapath"].(string)
	if !ok {
		log.Warning("Vrf for nat %s not found\n", tableNat.UUID)
		return nil
	}

	pbr := PolicyBasedRoute{
		Vrf:        vrf,
		IP:         tableNat.ExternalIP,
		LogicalIPs: []string{tableNat.LogicalIP},
		Type:       tableNat.Type,
	}

	err := policyBasedRouteDel(pbr)
	if err != nil {
		log.Warning("policyBasedRoute %+v del failed\n", pbr)
	}

	return nil
}
