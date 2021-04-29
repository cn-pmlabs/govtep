package tai

import (
	"errors"
	"math"
	"reflect"
	"sync"
	"time"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

type taiDriver struct {
	activeDriver  string
	handlers      map[string]DriverHandler
	handlersMutex *sync.Mutex
}

type ovsdbc struct {
	odbc.OvsdbC
}

var taiDBClient = ovsdbc{
	odbc.OvsdbC{
		Db:         odbc.VTEPDB,
		MonitorAll: true,
		TLSConfig:  nil,
		Client:     nil,
	},
}

// NewTaiDbClient connect and subscribe vtep DB
func NewTaiDbClient() {
	taiDBClient.Addr = odbc.VtepdbAddr
	err := taiDBClient.NewOvsDbClient()
	if err != nil {
		log.Warning("[TAI] Connect ovsdb %s[%s] failed, retry later\n", taiDBClient.Db, taiDBClient.Addr)
		go taiDBClient.reConnect()
	} else {
		log.Warning("[TAI] Connect ovsdb %s successed\n", taiDBClient.Db)
		initial, _ := taiDBClient.Client.MonitorAll(taiDBClient.Db, "")
		taiDBClient.taiProcessInitial(*initial)
		notifier := vtepdbNotifier{&taiDBClient}
		taiDBClient.Client.Register(notifier)
	}
}

func (c *ovsdbc) reConnect() {
	if c == nil {
		return
	}

	retryCnt := 0
	cycleTime := time.NewTimer(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
	for {
		select {
		case <-cycleTime.C:
			client, err := libovsdb.Connect(c.Addr, c.TLSConfig)
			if err == nil && client != nil {
				log.Warning("[TAI] Reconnect ovsdb %s successed\n", c.Db)
				c.Client = client
				initial, _ := c.MonitorDbTables(c.Db, c.MonitorAll, c.MonitorTables, "")
				c.taiProcessInitial(*initial)
				notifier := vtepdbNotifier{c}
				c.Client.Register(notifier)

				cycleTime.Stop()
				return
			}

			log.Info("[TAI] Try to connect ovsdb %s[%s] failed, retry after %v seconds\n",
				c.Db, c.Addr, math.Exp2(float64(retryCnt)))

			cycleTime.Reset(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
			if retryCnt <= 2 {
				retryCnt++
			}
		}
	}
}

var tai = taiDriver{
	handlers:      make(map[string]DriverHandler),
	handlersMutex: &sync.Mutex{},
}

// DriverHandler interface
type DriverHandler interface {
	TaiCreateObject(ObjID, interface{}) error
	TaiRemoveObject(ObjID, interface{}) error
	TaiAddObjectAttr(ObjID, interface{}, map[interface{}]interface{}) error
	TaiDelObjectAttr(ObjID, interface{}, map[interface{}]interface{}) error
	TaiSetObjectAttr(ObjID, interface{}, map[interface{}]interface{}) error
	TaiGetObjectAttr(ObjID, interface{}, []interface{}) (map[interface{}]interface{}, error)
	TaiListObject(ObjID) ([]interface{}, error)
}

// RegisterTaiDriverHandler register tai handler
func RegisterTaiDriverHandler(name string, handler DriverHandler) {
	tai.handlersMutex.Lock()
	defer tai.handlersMutex.Unlock()
	tai.handlers[name] = handler
	tai.activeDriver = name
}

// UnRegisterTaiDriverHandler unregister tai handler
func UnRegisterTaiDriverHandler(name string, handler DriverHandler) {
	tai.handlersMutex.Lock()
	defer tai.handlersMutex.Unlock()
	delete(tai.handlers, name)
}

func activeTaiDriver() (DriverHandler, error) {
	var err error
	tai.handlersMutex.Lock()
	defer tai.handlersMutex.Unlock()
	if len(tai.activeDriver) == 0 {
		return nil, errors.New("NULL ActiveDriver")
	}
	if handler, ok := tai.handlers[tai.activeDriver]; ok {
		return handler, err
	}
	return nil, errors.New("NULL ActiveDriver Register")
}

// getObjIDByTblName get switch Obj id from vtep DB table name
func getObjIDByTblName(tblName string) (ObjID, error) {
	var err error
	var taiObj ObjID

	switch tblName {
	case vtepdb.BridgeDomain:
		taiObj = ObjectIDBridge
	case vtepdb.Vrf:
		taiObj = ObjectIDVrf
	case vtepdb.L2port:
		taiObj = ObjectIDL2Port
	case vtepdb.L3port:
		taiObj = ObjectIDL3Port
	case vtepdb.Locator:
		taiObj = ObjectIDTunnel
	case vtepdb.RemoteFdb:
		taiObj = ObjectIDFDB
	case vtepdb.Route:
		taiObj = ObjectIDRoute
	case vtepdb.RemoteNeigh:
		taiObj = ObjectIDNeighbour
	case vtepdb.McastMacsLocal:
		taiObj = ObjectIDMcastFDB
	case vtepdb.ACL:
		taiObj = ObjectIDACL
	case vtepdb.ACLRule:
		taiObj = ObjectIDACLRule
	case vtepdb.PolicyBasedRoute:
		taiObj = ObjectIDPBR
	case vtepdb.AutoGatewayConf:
		taiObj = ObjectIDAutoGatewayConf
	default:
		return taiObj, errors.New("undefined tai object")
	}

	return taiObj, err
}

func rowToObj(objID ObjID, row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	var obj interface{}
	var attrs map[interface{}]interface{}
	switch objID {
	case ObjectIDBridge:
		obj, attrs = rowToBridgeObj(row)
	case ObjectIDVrf:
		obj, attrs = rowToVrfObj(row)
	case ObjectIDL2Port:
		obj, attrs = rowToL2portObj(row)
	case ObjectIDL3Port:
		obj, attrs = rowToL3portObj(row)
	case ObjectIDRoute:
		obj, attrs = rowToRouteObj(row)
	case ObjectIDTunnel:
		obj, attrs = rowToTunnelObj(row)
	case ObjectIDFDB:
		obj, attrs = rowToFdbObj(row)
	case ObjectIDNeighbour:
		obj, attrs = rowToNeighbourObj(row)
	case ObjectIDACL:
		obj, attrs = rowToACLObj(row)
	case ObjectIDACLRule:
		obj, attrs = rowToACLRuleObj(row)
	case ObjectIDMcastFDB:
		obj, attrs = rowToMcastfdbObj(row)
	case ObjectIDPBR:
		obj, attrs = rowToPBRObj(row)
	case ObjectIDAutoGatewayConf:
		obj, attrs = rowToAutoGatewayConfObj(row)
	}
	return obj, attrs
}

func rowToAttrs(objID ObjID, row libovsdb.Row) map[interface{}]interface{} {
	var attrs map[interface{}]interface{}
	switch objID {
	case ObjectIDBridge:
		attrs = rowToBridgeAttrs(row)
	case ObjectIDVrf:
		attrs = rowToVrfAttrs(row)
	case ObjectIDL2Port:
		attrs = rowToL2portAttrs(row)
	case ObjectIDL3Port:
		attrs = rowToL3portAttrs(row)
	case ObjectIDRoute:
		attrs = rowToRouteAttrs(row)
	case ObjectIDTunnel:
		attrs = rowToTunnelAttrs(row)
	case ObjectIDFDB:
		attrs = rowToFdbAttrs(row)
	case ObjectIDNeighbour:
		attrs = rowToNeighbourAttrs(row)
	case ObjectIDACL:
		attrs = rowToACLAttrs(row)
	case ObjectIDACLRule:
		attrs = rowToACLRuleAttrs(row)
	case ObjectIDMcastFDB:
		attrs = rowToMcastfdbAttrs(row)
	case ObjectIDPBR:
		attrs = rowToPBRAttrs(row)
	case ObjectIDAutoGatewayConf:
		attrs = rowToAutoGatewayConfAttrs(row)
	}
	return attrs
}

func (c *ovsdbc) taiProcessInitial(updates libovsdb.TableUpdates) {
	for table, tableupdate := range updates.Updates {
		objID, err := getObjIDByTblName(table)
		if err != nil {
			continue
		}
		if objID != ObjectIDTunnel {
			continue
		}

		for _, rowUpdate := range tableupdate.Rows {
			odbc.Float64ToInt(rowUpdate.New)
			odbc.Float64ToInt(rowUpdate.Old)
			op := odbc.GetRowUpdateOp(rowUpdate)

			switch op {
			case odbc.OpInsert:
				taiCreateObj(objID, rowUpdate.New)
			case odbc.OpDelete:
				taiRemoveObj(objID, rowUpdate.Old)
			case odbc.OpUpdate:
				taiUpdateObj(objID, rowUpdate.New, rowUpdate.Old)
			}
		}
	}

	for table, tableupdate := range updates.Updates {
		objID, err := getObjIDByTblName(table)
		if err != nil {
			continue
		}
		if (objID != ObjectIDBridge && objID != ObjectIDVrf) || objID == ObjectIDTunnel {
			continue
		}

		log.Info("[TAI] >>> table %v tableupdate %+v\n", table, tableupdate)

		for _, rowUpdate := range tableupdate.Rows {
			odbc.Float64ToInt(rowUpdate.New)
			odbc.Float64ToInt(rowUpdate.Old)
			op := odbc.GetRowUpdateOp(rowUpdate)

			switch op {
			case odbc.OpInsert:
				taiCreateObj(objID, rowUpdate.New)
			case odbc.OpDelete:
				taiRemoveObj(objID, rowUpdate.Old)
			case odbc.OpUpdate:
				taiUpdateObj(objID, rowUpdate.New, rowUpdate.Old)
			}
		}
	}

	for table, tableupdate := range updates.Updates {
		objID, err := getObjIDByTblName(table)
		if err != nil {
			continue
		}
		if objID == ObjectIDBridge || objID == ObjectIDVrf || objID == ObjectIDTunnel {
			continue
		}

		log.Info("[TAI] >>> table %v tableupdate %+v\n", table, tableupdate)

		for _, rowUpdate := range tableupdate.Rows {
			odbc.Float64ToInt(rowUpdate.New)
			odbc.Float64ToInt(rowUpdate.Old)
			op := odbc.GetRowUpdateOp(rowUpdate)

			switch op {
			case odbc.OpInsert:
				taiCreateObj(objID, rowUpdate.New)
			case odbc.OpDelete:
				taiRemoveObj(objID, rowUpdate.Old)
			case odbc.OpUpdate:
				taiUpdateObj(objID, rowUpdate.New, rowUpdate.Old)
			}
		}
	}
}

func (c *ovsdbc) taiNotifyUpdate(updates libovsdb.TableUpdates) {
	for table, tableupdate := range updates.Updates {
		objID, err := getObjIDByTblName(table)
		if err != nil {
			continue
		}

		log.Info("[TAI] >>> table %v tableupdate %+v\n", table, tableupdate)

		for uuid, rowUpdate := range tableupdate.Rows {
			rowUpdate = odbc.RowUpdateOptimize(rowUpdate, uuid)
			op := odbc.GetRowUpdateOp(rowUpdate)

			switch op {
			case odbc.OpInsert:
				taiCreateObj(objID, rowUpdate.New)
			case odbc.OpDelete:
				taiRemoveObj(objID, rowUpdate.Old)
			case odbc.OpUpdate:
				taiUpdateObj(objID, rowUpdate.New, rowUpdate.Old)
			}
		}
	}
}

func taiCreateObj(objID ObjID, row libovsdb.Row) {
	obj, attrs := rowToObj(objID, row)

	if obj == nil {
		log.Warning("[TAI] taiCreateObj convert obj %v failed\n", objID)
		return
	}
	log.Info("[TAI] obj %v attrs %v\n", obj, attrs)

	err := taiCreateObject(objID, obj)
	if err != nil {
		log.Warning("[TAI] taiCreateObj %s failed\n", ObjectOrder[objID])
		return
	}

	if len(attrs) != 0 {
		_ = taiAddObjectAttr(objID, obj, attrs)
	}
}

func taiRemoveObj(objID ObjID, row libovsdb.Row) {
	obj, attrs := rowToObj(objID, row)
	// Do we need delete object attr? just remove object can work either
	_ = taiDelObjectAttr(objID, obj, attrs)

	if obj == nil {
		log.Warning("[TAI] taiRemoveObj convert obj %v failed\n", objID)
		return
	}
	err := taiRemoveObject(objID, obj)
	if err != nil {
		log.Warning("[TAI] taiRemoveObj failed %v\n", err)
		return
	}
}

func taiUpdateObj(objID ObjID, newrow libovsdb.Row, oldrow libovsdb.Row) {
	newobj, newattrs := rowToObj(objID, newrow)
	oldattrs := rowToAttrs(objID, oldrow)

	if newobj == nil {
		log.Warning("[TAI] taiUpdateObj convert obj %v failed\n", objID)
		return
	}

	attrsAdd := make(map[interface{}]interface{})
	attrsDel := make(map[interface{}]interface{})
	attrsSet := make(map[interface{}]interface{})

	for k, v1 := range newattrs {
		if v2, ok := oldattrs[k]; ok {
			if !reflect.DeepEqual(v1, v2) {
				attrsSet[k] = v1
			}
		} else {
			attrsAdd[k] = v1
		}
	}

	for k, v := range oldattrs {
		if _, ok := oldattrs[k]; ok == false {
			attrsDel[k] = v
		}
	}

	_ = taiAddObjectAttr(objID, newobj, attrsAdd)
	_ = taiDelObjectAttr(objID, newobj, attrsDel)
	_ = taiSetObjectAttr(objID, newobj, attrsSet)
}

func taiCreateObject(objID ObjID, obj interface{}) error {
	handler, err := activeTaiDriver()
	if err != nil {
		return err
	}
	err = handler.TaiCreateObject(objID, obj)
	return err
}

func taiRemoveObject(objID ObjID, obj interface{}) error {
	handler, err := activeTaiDriver()
	if err != nil {
		return err
	}
	err = handler.TaiRemoveObject(objID, obj)
	return err
}

func taiAddObjectAttr(objID ObjID, obj interface{}, attrs map[interface{}]interface{}) error {
	handler, err := activeTaiDriver()
	if err != nil {
		return err
	}
	err = handler.TaiAddObjectAttr(objID, obj, attrs)
	return err
}

func taiDelObjectAttr(objID ObjID, obj interface{}, attrs map[interface{}]interface{}) error {
	handler, err := activeTaiDriver()
	if err != nil {
		return err
	}
	err = handler.TaiDelObjectAttr(objID, obj, attrs)
	return err
}

func taiSetObjectAttr(objID ObjID, obj interface{}, attrs map[interface{}]interface{}) error {
	handler, err := activeTaiDriver()
	if err != nil {
		return err
	}
	err = handler.TaiSetObjectAttr(objID, obj, attrs)
	return err
}

func taiGetObjectAttr(objID ObjID, obj interface{}, attrIDs []interface{}) (map[interface{}]interface{}, error) {
	handler, err := activeTaiDriver()
	if err != nil {
		return nil, err
	}
	attrlist, err := handler.TaiGetObjectAttr(objID, obj, attrIDs)
	return attrlist, err
}

func taiGetObject(objID ObjID) ([]interface{}, error) {
	handler, err := activeTaiDriver()
	if err != nil {
		return nil, err
	}
	objs, err := handler.TaiListObject(objID)
	return objs, err
}
