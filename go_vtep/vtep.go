package govtep

import (
	"math"
	"time"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnnb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnnorthbound"
	ovnsb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnsouthbound"

	"github.com/cn-pmlabs/govtep/lib/log"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"

	"github.com/ebay/libovsdb"
)

// OvnCentralSet false when system start up, set true if vtepdb global.ovntarget configured
var (
	OvnCentralSet       bool = false
	OvnCentralConnected bool = false
)

type ovsdbc struct {
	odbc.OvsdbC
}

var nbDBClient = ovsdbc{
	odbc.OvsdbC{
		Db:         ovnnb.OVNNORTHBOUND,
		TLSConfig:  nil,
		Client:     nil,
		Reconn:     true,
		MonitorAll: false,
		MonitorTables: []string{
			//ovnnb.LogicalSwitch,
			ovnnb.LogicalRouter,
			ovnnb.LoadBalancer,
			ovnnb.Nat,
			ovnnb.LogicalRouterStaticRoute,
			ovnnb.ACL,
		},
	},
}

var sbDBClient = ovsdbc{
	odbc.OvsdbC{
		Db:         ovnsb.OVNSOUTHBOUND,
		TLSConfig:  nil,
		Client:     nil,
		Reconn:     true,
		MonitorAll: false,
		MonitorTables: []string{
			ovnsb.Chassis,
			ovnsb.Encap,
			ovnsb.DatapathBinding,
			ovnsb.PortBinding,
			ovnsb.MacBinding,
		},
	},
}

var vtepDBClient = ovsdbc{
	odbc.OvsdbC{
		Db:         vtepdb.CONTROLLERVTEP,
		Addr:       odbc.VtepdbAddr,
		TLSConfig:  nil,
		Client:     nil,
		Reconn:     true,
		MonitorAll: false,
		MonitorTables: []string{
			vtepdb.Global,
			vtepdb.PhysicalSwitch,
		},
	},
}

// NewVtepDbClient connect to VTEP DB
func NewVtepDbClient() {
	// init vtepdb lib, disconnect handler todo
	vtepdb.InitControllervtep(odbc.VtepdbAddr)

	vtepDBClient.Addr = odbc.VtepdbAddr
	err := vtepDBClient.NewOvsDbClient()
	if err != nil {
		log.Warning("Connect ovsdb %s[%s] failed, retry later\n", vtepDBClient.Db, vtepDBClient.Addr)
		go vtepDBClient.vtepDBReConnect()
	} else {
		log.Warning("Connect ovsdb %s successed\n", vtepDBClient.Db)
		initial, _ := vtepDBClient.MonitorDbTables(vtepDBClient.Db, vtepDBClient.
			MonitorAll, vtepDBClient.MonitorTables, "")
		vtepDBClient.vtepDbNotifyUpdate(*initial)
		notifier := vtepDbNotifier{&vtepDBClient}
		vtepDBClient.Client.Register(notifier)
	}
}

func (c *ovsdbc) vtepDBReConnect() {
	if c == nil {
		return
	}

	retryCnt := 0
	cycleTime := time.NewTimer(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
	for {
		select {
		case <-cycleTime.C:
			c.Addr = odbc.VtepdbAddr
			client, err := libovsdb.Connect(c.Addr, c.TLSConfig)
			if err == nil && client != nil {
				log.Warning("Reconnect ovsdb %s successed\n", c.Db)
				c.Client = client
				vtepdb.ControllervtepClient.Client = client

				initial, _ := c.MonitorDbTables(c.Db, c.MonitorAll, c.MonitorTables, "")
				c.vtepDbNotifyUpdate(*initial)
				notifier := vtepDbNotifier{c}
				c.Client.Register(notifier)

				cycleTime.Stop()
				return
			}

			log.Info("Try to connect ovsdb %s[%s] failed, retry after %v seconds\n",
				c.Db, c.Addr, math.Exp2(float64(retryCnt)))

			cycleTime.Reset(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
			if retryCnt <= 2 {
				retryCnt++
			}
		}
	}
}

func (c *ovsdbc) vtepDbNotifyUpdate(updates libovsdb.TableUpdates) {
	var op string
	for table, tableupdate := range updates.Updates {
		for uuid, rowUpdate := range tableupdate.Rows {
			rowUpdate = odbc.RowUpdateOptimize(rowUpdate, uuid)
			op = odbc.GetRowUpdateOp(rowUpdate)

			log.Warning(">>> Vtep Table %s rowuuid %s op %s\n", table, uuid, op)

			// not process tables before Global ovn target set
			if OvnCentralSet == false && table != vtepdb.Global {
				log.Warning("Ovn central not connected, ignore vtepdb modification\n")
				continue
			}

			switch table {
			case vtepdb.Global:
				vtepGlobalNotifyUpdate(op, rowUpdate)
			case vtepdb.PhysicalSwitch:
				physicalSwitchNotifyUpdate(op, rowUpdate)
			default:
				continue
			}
		}
	}
}

func (c *ovsdbc) ovnNbNotifyUpdate(updates libovsdb.TableUpdates) {
	var op string
	for table, tableupdate := range updates.Updates {
		for uuid, rowUpdate := range tableupdate.Rows {
			rowUpdate = odbc.RowUpdateOptimize(rowUpdate, uuid)
			op = odbc.GetRowUpdateOp(rowUpdate)

			log.Info(">>> NB Table %s rowuuid %s op %s\n", table, uuid, op)

			switch table {
			case ovnnb.LogicalSwitch:
				//xxhNotifyUpdate(op, rowUpdate, uuid)
			case ovnnb.LogicalRouter:
				// process static_router/LB from Logical_Router static_routes and load_balancer update
				logicalRouterNotifyUpdate(op, rowUpdate, uuid)
			case ovnnb.LoadBalancer:
				// load balancer backends update and removal, creation should be processed in LR.load_balancer
				loadBalancerNotifyUpdate(op, rowUpdate, uuid)
			case ovnnb.Nat:
				natNotifyUpdate(op, rowUpdate, uuid)
			case ovnnb.ACL:
				// process ACL from Logical_Switch acls update
				// eg: ovn-nbctl --name=acl2 acl-add ls from-lport 1002 'outport == "ls-vm1" && ip && icmp' allow
				aclNotifyUpdate(op, rowUpdate)
			default:
				continue
			}
		}
	}
}

// optimize to sb<->vtepdb consistency
func (c *ovsdbc) ovnSbReComputeAll() {
	if c == nil {
		return
	}

	var tableUpdates libovsdb.TableUpdates
	tableUpdates.Updates = make(map[string]libovsdb.TableUpdate)

	for _, tableName := range c.MonitorTables {
		var conditions []interface{}
		tableRows, num := ovnsb.SelectRows(tableName, conditions)
		if num == 0 {
			continue
		}

		var tableUpdate libovsdb.TableUpdate
		tableUpdate.Rows = make(map[string]libovsdb.RowUpdate)
		for _, tableRow := range tableRows {
			if UUID, ok := tableRow["_uuid"].(libovsdb.UUID); ok {
				var rowUpdate libovsdb.RowUpdate
				rowUpdate.New.Fields = tableRow
				tableUpdate.Rows[UUID.GoUUID] = rowUpdate
			}
		}
		tableUpdates.Updates[tableName] = tableUpdate
	}

	c.ovnSbProcessInitial(tableUpdates)
	// consistency
}

func (c *ovsdbc) ovnSbReComputeSchedule() {
	cycleTime := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-cycleTime.C:
			if c.Client == nil || OvnCentralConnected == false {
				continue
			}

			var tableUpdates libovsdb.TableUpdates
			tableUpdates.Updates = make(map[string]libovsdb.TableUpdate)

			for _, tableName := range c.MonitorTables {
				var conditions []interface{}
				tableRows, num := ovnsb.SelectRows(tableName, conditions)
				if num == 0 {
					continue
				}

				var tableUpdate libovsdb.TableUpdate
				tableUpdate.Rows = make(map[string]libovsdb.RowUpdate)
				for _, tableRow := range tableRows {
					if UUID, ok := tableRow["_uuid"].(libovsdb.UUID); ok {
						var rowUpdate libovsdb.RowUpdate
						rowUpdate.New.Fields = tableRow
						tableUpdate.Rows[UUID.GoUUID] = rowUpdate
					}
				}
				tableUpdates.Updates[tableName] = tableUpdate
			}
			c.ovnSbProcessInitial(tableUpdates)
			// consistency

			cycleTime.Reset(time.Second * 5)
		}
	}
}

func (c *ovsdbc) ovnSbProcessInitial(updates libovsdb.TableUpdates) {
	var op string

	// process phsical network infomation first
	for table, tableupdate := range updates.Updates {
		if table != ovnsb.Chassis && table != ovnsb.Encap {
			continue
		}
		for _, rowUpdate := range tableupdate.Rows {
			// missing json number conversion in libovsdb, convert float64 to int
			odbc.Float64ToInt(rowUpdate.New)
			odbc.Float64ToInt(rowUpdate.Old)
			op = odbc.GetRowUpdateOp(rowUpdate)

			switch table {
			case ovnsb.Chassis:
				locatorNotifyUpdate(op, rowUpdate)
			case ovnsb.Encap:
				encapNotifyUpdate(op, rowUpdate)
			default:
				continue
			}
		}
	}

	// process datapaths
	if sbDatapaths, ok := updates.Updates[ovnsb.DatapathBinding]; ok {
		for uuid, rowUpdate := range sbDatapaths.Rows {
			odbc.Float64ToInt(rowUpdate.New)
			odbc.Float64ToInt(rowUpdate.Old)
			op = odbc.GetRowUpdateOp(rowUpdate)

			datapathNotifyUpdate(op, rowUpdate, uuid)
		}
	}

	// process phsical network and logical networking binding
	for table, tableupdate := range updates.Updates {
		if table != ovnsb.PortBinding && table != ovnsb.MacBinding {
			continue
		}
		for uuid, rowUpdate := range tableupdate.Rows {
			odbc.Float64ToInt(rowUpdate.New)
			odbc.Float64ToInt(rowUpdate.Old)
			op = odbc.GetRowUpdateOp(rowUpdate)

			switch table {
			case ovnsb.PortBinding:
				portbindingNotifyUpdate(op, rowUpdate, uuid)
			case ovnsb.MacBinding:
				macbindingNotifyUpdate(op, rowUpdate)
			default:
				continue
			}
		}
	}
}

func (c *ovsdbc) ovnSbNotifyUpdate(updates libovsdb.TableUpdates) {
	var op string
	for table, tableupdate := range updates.Updates {
		for uuid, rowUpdate := range tableupdate.Rows {
			// missing json number conversion in libovsdb, convert float64 to int
			rowUpdate = odbc.RowUpdateOptimize(rowUpdate, uuid)
			op = odbc.GetRowUpdateOp(rowUpdate)

			log.Info(">>> SB Table %s rowuuid %s op %s\n", table, uuid, op)

			switch table {
			case ovnsb.DatapathBinding:
				datapathNotifyUpdate(op, rowUpdate, uuid)
			case ovnsb.PortBinding:
				portbindingNotifyUpdate(op, rowUpdate, uuid)
			case ovnsb.MacBinding:
				macbindingNotifyUpdate(op, rowUpdate)
			case ovnsb.Chassis:
				locatorNotifyUpdate(op, rowUpdate)
			case ovnsb.Encap:
				encapNotifyUpdate(op, rowUpdate)
			default:
				continue
			}
		}
	}

	sbDBClient.ovnSbReComputeAll()
}

func (c *ovsdbc) ovnSbReConnect() {
	if c == nil {
		return
	}

	OvnCentralConnected = false
	retryCnt := 0
	cycleTime := time.NewTimer(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
	for {
		// for ovn db target change
		c.Addr = odbc.OvnsbAddr

		select {
		case <-cycleTime.C:
			client, err := libovsdb.Connect(c.Addr, c.TLSConfig)
			if err == nil && client != nil {
				log.Warning("Reconnect ovsdb %s successed\n", c.Db)

				c.Client = client
				OvnCentralConnected = true

				// init Phsical switch after ovn connected
				PhysicalSwitchInit()

				initial, _ := c.MonitorDbTables(c.Db, c.MonitorAll, c.MonitorTables, "")
				c.ovnSbProcessInitial(*initial)
				notifier := ovnSbNotifier{c}
				c.Client.Register(notifier)

				cycleTime.Stop()
				return
			}

			log.Info("Try to connect ovsdb %s[%s] failed, retry after %v seconds\n",
				c.Db, c.Addr, math.Exp2(float64(retryCnt)))

			cycleTime.Reset(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
			if retryCnt <= 2 {
				retryCnt++
			}
		}
	}
}

func (c *ovsdbc) ovnNbReConnect() {
	if c == nil {
		return
	}

	retryCnt := 0
	cycleTime := time.NewTimer(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
	for {
		// for ovn db target change
		c.Addr = odbc.OvnnbAddr

		select {
		case <-cycleTime.C:
			client, err := libovsdb.Connect(c.Addr, c.TLSConfig)
			if err == nil && client != nil {
				log.Warning("Reconnect ovsdb %s successed\n", c.Db)

				c.Client = client
				initial, _ := c.MonitorDbTables(c.Db, c.MonitorAll, c.MonitorTables, "")
				c.ovnNbNotifyUpdate(*initial)
				notifier := ovnNbNotifier{c}
				c.Client.Register(notifier)

				cycleTime.Stop()
				return
			}

			log.Info("Try to connect ovsdb %s[%s] failed, retry after %v seconds\n",
				c.Db, c.Addr, math.Exp2(float64(retryCnt)))

			cycleTime.Reset(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
			if retryCnt <= 2 {
				retryCnt++
			}
		}
	}
}

// OvnSbLibReConnect tmp solution for ovn southbound reconnect
func OvnSbLibReConnect() {
	retryCnt := 0
	cycleTime := time.NewTimer(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
	for {
		select {
		case <-cycleTime.C:
			client, err := libovsdb.Connect(odbc.OvnsbAddr, nil)
			if err == nil && client != nil {
				log.Warning("Reconnect ovnsb lib successed\n")
				ovnsb.OvnsouthboundClient.Client = client

				notifier := ovnSbLibNotifier{ovnsb.OvnsouthboundClient.Client}
				ovnsb.OvnsouthboundClient.Client.Register(notifier)

				cycleTime.Stop()
				return
			}

			log.Warning("Try to connect LIB ovsdb %s[%s] failed, retry after %v seconds\n",
				"ovnsb", odbc.OvnsbAddr, math.Exp2(float64(retryCnt)))

			cycleTime.Reset(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
			if retryCnt <= 2 {
				retryCnt++
			}
		}
	}
}

// OvnNbLibReConnect for ovn northbound reconnect
func OvnNbLibReConnect() {
	retryCnt := 0
	cycleTime := time.NewTimer(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
	for {
		select {
		case <-cycleTime.C:
			client, err := libovsdb.Connect(odbc.OvnnbAddr, nil)
			if err == nil && client != nil {
				log.Warning("Reconnect ovnnb lib successed\n")
				ovnnb.OvnnorthboundClient.Client = client

				notifier := ovnNbLibNotifier{ovnnb.OvnnorthboundClient.Client}
				ovnnb.OvnnorthboundClient.Client.Register(notifier)

				cycleTime.Stop()
				return
			}

			log.Warning("Try to connect LIB ovsdb %s[%s] failed, retry after %v seconds\n",
				"ovnnb", odbc.OvnnbAddr, math.Exp2(float64(retryCnt)))

			cycleTime.Reset(time.Second * time.Duration(math.Exp2(float64(retryCnt))))
			if retryCnt <= 2 {
				retryCnt++
			}
		}
	}
}

// NewNbDbClient connect to OVN northbound DB
func NewNbDbClient() {
	nbDBClient.Addr = odbc.OvnnbAddr
	err := nbDBClient.NewOvsDbClient()
	if err != nil {
		log.Warning("Connect ovsdb %s[%s] failed, retry later\n", nbDBClient.Db, nbDBClient.Addr)
		go nbDBClient.ovnNbReConnect()
	} else {
		log.Warning("Connect ovsdb %s successed\n", nbDBClient.Db)
		initial, _ := nbDBClient.MonitorDbTables(nbDBClient.Db, nbDBClient.
			MonitorAll, nbDBClient.MonitorTables, "")
		nbDBClient.ovnNbNotifyUpdate(*initial)
		notifier := ovnNbNotifier{&nbDBClient}
		nbDBClient.Client.Register(notifier)
	}
}

// NewSbDbClient connect to OVN sorthbound DB
func NewSbDbClient() {
	sbDBClient.Addr = odbc.OvnsbAddr
	err := sbDBClient.NewOvsDbClient()
	if err != nil {
		log.Warning("Connect ovsdb %s[%s] failed, retry later\n", sbDBClient.Db, sbDBClient.Addr)
		go sbDBClient.ovnSbReConnect()
	} else {
		log.Warning("Connect ovsdb %s successed\n", sbDBClient.Db)
		OvnCentralConnected = true
		// init Phsical switch after ovn connected
		PhysicalSwitchInit()

		initial, _ := sbDBClient.MonitorDbTables(sbDBClient.Db, sbDBClient.
			MonitorAll, sbDBClient.MonitorTables, "")
		sbDBClient.ovnSbProcessInitial(*initial)
		notifier := ovnSbNotifier{&sbDBClient}
		sbDBClient.Client.Register(notifier)
	}

	go sbDBClient.ovnSbReComputeSchedule()
}

// NewOvnLibClient connect to OVN sorthbound DB and north DB
func NewOvnLibClient() {
	if ovnsb.InitOvnsouthbound(odbc.OvnsbAddr) != nil {
		go OvnSbLibReConnect()
	} else {
		notifier := ovnSbLibNotifier{ovnsb.OvnsouthboundClient.Client}
		ovnsb.OvnsouthboundClient.Client.Register(notifier)
	}

	if ovnnb.InitOvnnorthbound(odbc.OvnnbAddr) != nil {
		go OvnNbLibReConnect()
	} else {
		notifier := ovnNbLibNotifier{ovnnb.OvnnorthboundClient.Client}
		ovnnb.OvnnorthboundClient.Client.Register(notifier)
	}
}

func ovnCentralConnectJob() {
	for {
		time.Sleep(10 * time.Millisecond)
		if OvnCentralSet == true {
			// start ovn lib connection
			NewOvnLibClient()
			// Start OVN SB connection and update Notifier
			NewSbDbClient()
			// Start OVN NB connection and update Notifier
			NewNbDbClient()
			return
		}
	}
}

// OvnCentralConnect vtep controller connect to ovn sb and nb
func OvnCentralConnect() {
	go ovnCentralConnectJob()
}

func vtepGlobalNotifyUpdate(op string, rowUpdate libovsdb.RowUpdate) {
	switch op {
	case odbc.OpInsert:
		vtepGlobalCreate(rowUpdate.New)
	case odbc.OpDelete:
		//vtepGlobalRemove(rowUpdate.Old)
	case odbc.OpUpdate:
		//vtepGlobalUpdate(rowUpdate.New, rowUpdate.Old)
	}
}

func vtepGlobalCreate(row libovsdb.Row) {
	tableGlobal := vtepdb.ConvertRowToGlobal(libovsdb.ResultRow(row.Fields))

	log.Info("ovn target set northbound: %s southbound: %s\n",
		tableGlobal.OvnnbTarget, tableGlobal.OvnsbTarget)

	if tableGlobal.OvnnbTarget != "" && tableGlobal.OvnsbTarget != "" {
		odbc.OvnnbAddr = tableGlobal.OvnnbTarget
		odbc.OvnsbAddr = tableGlobal.OvnsbTarget
		OvnCentralSet = true
	}
}

func vtepGlobalUpdate(newrow libovsdb.Row, oldrow libovsdb.Row) {
	var err error

	for field, oldValue := range oldrow.Fields {
		switch field {
		case vtepdb.GlobalFieldOvnnbTarget:
			err = vtepGlobalUpdateNb(newrow)
		case vtepdb.GlobalFieldOvnsbTarget:
			err = vtepGlobalUpdateSb(newrow)
		default:
			continue
		}

		if err != nil {
			log.Warning("update failed %s old-value %v failed: %v\n", field, oldValue, err)
			return
		}
	}
}

func vtepGlobalUpdateNb(row libovsdb.Row) error {
	tableGlobal := vtepdb.ConvertRowToGlobal(libovsdb.ResultRow(row.Fields))

	log.Warning("update ovn northbound: %s \n", tableGlobal.OvnnbTarget)

	odbc.OvnnbAddr = tableGlobal.OvnnbTarget
	// disconnect origin connection then auto reconnect
	if nil != ovnnb.OvnnorthboundClient.Client {
		ovnnb.OvnnorthboundClient.Client.Disconnect()
	}
	if nil != nbDBClient.Client {
		nbDBClient.Client.Disconnect()
	}

	return nil
}

func vtepGlobalUpdateSb(row libovsdb.Row) error {
	tableGlobal := vtepdb.ConvertRowToGlobal(libovsdb.ResultRow(row.Fields))

	log.Warning("update ovn southbound: %s\n", tableGlobal.OvnsbTarget)

	odbc.OvnsbAddr = tableGlobal.OvnsbTarget
	// disconnect origin connection then auto reconnect
	if nil != ovnsb.OvnsouthboundClient.Client {
		ovnsb.OvnsouthboundClient.Client.Disconnect()
	}
	if nil != sbDBClient.Client {
		sbDBClient.Client.Disconnect()
	}

	return nil
}
