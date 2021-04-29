package govtep

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// RemoteFdb ...
type RemoteFdb struct {
	UUID          string
	Bridge        string //Bridge uuid
	Mac           string
	RemoteLocator string //VTEP DB Locator uuid
}

// LocalFdb ...
type LocalFdb struct {
	UUID      string
	Bridge    string //Bridge uuid
	Mac       string
	OutL2Port string
}

func remoteFdbCreate(fdbs []RemoteFdb) error {
	for _, fdb := range fdbs {
		/*if !bdIsExist(fdb.Bridge) {
			return errors.New("fdb Create fail, because bd not found")
		}
		namedUUID, err := odbc.NewRowUUID()
		if err != nil {
			return err
		}

		row := map[string]interface{}{
			"bridge":         fdb.Bridge,
			"mac":            fdb.Mac,
			"remote_locator": fdb.RemoteLocator,
		}
		insertOp := libovsdb.Operation{
			Op:       odbc.OpInsert,
			Table:    odbc.VTEP_RemoteFDB,
			Row:      row,
			UUIDName: namedUUID,
		}

		mutateUUID := []libovsdb.UUID{odbc.StringToGoUUID(namedUUID)}
		mutateSet, err := libovsdb.NewOvsSet(mutateUUID)
		if err != nil {
			return err
		}
		mutation := libovsdb.NewMutation("unicastFdb", odbc.OpInsert, mutateSet)
		condition := libovsdb.NewCondition("name", "==", fdb.Bridge)
		mutateOp := libovsdb.Operation{
			Op:        odbc.OpMutate,
			Table:     odbc.VTEP_Bridge_Domain,
			Mutations: []interface{}{mutation},
			Where:     []interface{}{condition},
		}*/

		fdbIndex := vtepdb.RemoteFdbIndex{
			Bridge: fdb.Bridge,
			Mac:    fdb.Mac,
		}
		_, err := vtepdb.RemoteFdbGetByIndex(fdbIndex)
		if err == nil {
			log.Info("fdb %+v already exist", fdbIndex)
			continue
		}

		tableFdb := vtepdb.TableRemoteFdb{
			Bridge:        fdb.Bridge,
			Mac:           fdb.Mac,
			RemoteLocator: fdb.RemoteLocator,
		}
		bdIndex := vtepdb.BridgeDomainIndex{
			Name: fdb.Bridge,
		}

		err = vtepdb.BridgeDomainUpdateAddUnicastfdb(bdIndex, tableFdb)
		if err != nil {
			return err
		}
	}
	return nil
}

func remoteFdbRemove(fdbs []RemoteFdb) error {
	// remove fdb from bd
	for _, fdb := range fdbs {
		fdbIndex := vtepdb.RemoteFdbIndex{
			Bridge: fdb.Bridge,
			Mac:    fdb.Mac,
		}
		tableFdb, err := vtepdb.RemoteFdbGetByIndex(fdbIndex)
		if err != nil {
			log.Error("Get RemoteFDB by bridge %s mac %s failed",
				fdb.Bridge, fdb.Mac)
			continue
		}

		bdIndex := vtepdb.BridgeDomainIndex{
			Name: fdb.Bridge,
		}
		var fdbMod []libovsdb.UUID
		fdbMod = append(fdbMod, libovsdb.UUID{GoUUID: tableFdb.UUID})
		err = vtepdb.BridgeDomainUpdateUnicastfdbDelvalue(bdIndex, fdbMod)
		if err != nil {
			log.Error("Bridge Domain %s remove mac %s failed",
				fdb.Bridge, fdb.Mac)
			continue
		}
		// No need to delete fdb table individually, auto removed after delete from BD

	}
	return nil
}

func remoteFdbUpdate(port PortInfo, fdbOp map[string]string) error {
	for mac, op := range fdbOp {
		switch op {
		case odbc.OpInsert:
			tableFdb := vtepdb.TableRemoteFdb{
				Bridge:        port.Bd,
				Mac:           mac,
				RemoteLocator: port.Locator,
			}
			bdIndex := vtepdb.BridgeDomainIndex{
				Name: port.Bd,
			}
			vtepdb.BridgeDomainUpdateAddUnicastfdb(bdIndex, tableFdb)
		case odbc.OpDelete:
			tableFdbIndex := vtepdb.RemoteFdbIndex{
				Bridge: port.Bd,
				Mac:    mac,
			}
			tableFdb, err := vtepdb.RemoteFdbGetByIndex(tableFdbIndex)
			if err != nil {
				log.Warning("Remote FDB %v delete failed: not exist", tableFdbIndex)
			}
			bdIndex := vtepdb.BridgeDomainIndex{
				Name: port.Bd,
			}
			vtepdb.BridgeDomainUpdateUnicastfdbDelvalue(bdIndex,
				[]libovsdb.UUID{{GoUUID: tableFdb.UUID}})
		}
	}

	return nil
}
