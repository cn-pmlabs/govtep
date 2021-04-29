package dbname

import (
	"fmt"
	"sync"

	"github.com/ebay/libovsdb"
)

// OvsdbC db connection configure and status
type odbc struct {
	Client    *libovsdb.OvsdbClient
	Tranmutex sync.Mutex
}

// DbnameClient ovsdb connection and transaction
var DbnameClient odbc

// InitDbname init db operation
func InitDbname(addr string) error {
	c, err := libovsdb.Connect(addr, nil)
	if err != nil {
		return fmt.Errorf("InitDbname: Fail to connect %s", DBNAME)
	}
	DbnameClient.Client = c
	return err
}

// RegisterDbnameClient init db operation
func RegisterDbnameClient(c *libovsdb.OvsdbClient) error {
	if c == nil {
		return fmt.Errorf("RegisterDbnameClient: invalid nil client")
	}

	DbnameClient.Client = c
	return nil
}
