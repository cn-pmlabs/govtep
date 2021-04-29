package ovsdbclient

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/ebay/libovsdb"
)

// OvsdbC db connection configure and status
type OvsdbC struct {
	Db            string
	Addr          string
	TLSConfig     *tls.Config
	Client        *libovsdb.OvsdbClient
	Tranmutex     sync.Mutex
	Reconn        bool
	MonitorAll    bool
	MonitorTables []string

	//Cache        map[string]map[string]libovsdb.Row
	//Cachemutex   sync.RWMutex
	//signalCB     OVNSignal
	//disconnectCB OVNDisconnectedCallback
}

// UpdateRows update db.table row's field with updates
// return updated number
func (c *OvsdbC) UpdateRows(db string, table string,
	updates map[string]interface{}, conditions []interface{}) int {
	operation := libovsdb.Operation{
		Op:    OpUpdate,
		Table: table,
		Row:   updates,
		Where: conditions,
	}
	results, err := c.Transact(c.Db, operation)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 0
	}
	return results[0].Count
}

// MutateRows mutate db.table row's field with conditions
// return modified number
func (c *OvsdbC) MutateRows(db string, table string,
	mutations []interface{}, conditions []interface{}) int {
	operation := libovsdb.Operation{
		Op:        OpMutate,
		Table:     table,
		Mutations: mutations,
		Where:     conditions,
	}
	results, err := c.Transact(c.Db, operation)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 0
	}
	return results[0].Count
}

// DeleteRows delete db.table rows with conditions
// return delete number
func (c *OvsdbC) DeleteRows(db string, table string,
	conditions []interface{}) int {
	operation := libovsdb.Operation{
		Op:    OpDelete,
		Table: table,
		Where: conditions,
	}
	results, err := c.Transact(c.Db, operation)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 0
	}
	return results[0].Count
}

// SelectRows check db.table with conditions existence
// return ResultRow and selected rows number
func (c *OvsdbC) SelectRows(db string, table string,
	conditions []interface{}) ([]libovsdb.ResultRow, int) {
	operation := libovsdb.Operation{
		Op:    OpSelect,
		Table: table,
		Where: conditions,
	}
	results, err := c.Transact(c.Db, operation)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return []libovsdb.ResultRow{}, 0
	}

	if len(results[0].Rows) > 0 {
		return results[0].Rows, len(results[0].Rows)
	}
	return []libovsdb.ResultRow{}, 0
}

// Transact with mutex and error check
func (c *OvsdbC) Transact(db string, ops ...libovsdb.Operation) ([]libovsdb.OperationResult, error) {
	// Only support one trans at same time now.
	c.Tranmutex.Lock()
	defer c.Tranmutex.Unlock()
	reply, err := c.Client.Transact(db, ops...)
	if err != nil {
		return reply, err
	}

	for i, o := range reply {
		if o.Error != "" {
			if i < len(ops) {
				return nil, fmt.
					Errorf("Transaction Failed due to an error : %v details: %v in %v", o.Error, o.Details, ops[i])
			}
			return nil, fmt.
				Errorf("Transaction Failed due to an error : %v details: %v", o.Error, o.Details)
		}
	}
	if len(reply) < len(ops) {
		return reply, fmt.
			Errorf("Number of Replies should be atleast equal to number of operations")
	}

	return reply, nil
}

// MonitorDbTables for specific tables monitor
func (c *OvsdbC) MonitorDbTables(db string, all bool, tables []string,
	jsonContext string) (*libovsdb.TableUpdates, error) {
	if all {
		return c.Client.MonitorAll(db, jsonContext)
	}

	requests := make(map[string]libovsdb.MonitorRequest)
	schema, _ := c.Client.Schema[db]
	for _, table := range tables {
		var columns []string
		for column := range schema.Tables[table].Columns {
			columns = append(columns, column)
		}
		request := libovsdb.MonitorRequest{
			Columns: columns,
			Select: libovsdb.MonitorSelect{
				Initial: true,
				Insert:  true,
				Delete:  true,
				Modify:  true,
			},
		}
		requests[table] = request
	}
	return c.Client.Monitor(db, jsonContext, requests)
}

// NewOvsDbClient ovsdb connection
func (c *OvsdbC) NewOvsDbClient() error {
	client, err := libovsdb.Connect(c.Addr, c.TLSConfig)
	if err != nil {
		return fmt.Errorf("NewOvsDbClient: Fail to connect %s", c.Db)
	}
	c.Client = client
	return nil
}
