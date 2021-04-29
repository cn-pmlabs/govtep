package dbname

import (
	"fmt"

	"github.com/ebay/libovsdb"
)

// UpdateRows update db.table row's field with updates
// return updated number
func UpdateRows(table string,
	updates map[string]interface{}, conditions []interface{}) int {
	operation := libovsdb.Operation{
		Op:    opUpdate,
		Table: table,
		Row:   updates,
		Where: conditions,
	}
	results, err := Transact(operation)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 0
	}
	return results[0].Count
}

// MutateRows mutate db.table row's field with conditions
// return modified number
func MutateRows(table string,
	mutations []interface{}, conditions []interface{}) int {
	operation := libovsdb.Operation{
		Op:        opMutate,
		Table:     table,
		Mutations: mutations,
		Where:     conditions,
	}
	results, err := Transact(operation)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 0
	}
	return results[0].Count
}

// DeleteRows delete db.table rows with conditions
// return delete number
func DeleteRows(table string,
	conditions []interface{}) int {
	operation := libovsdb.Operation{
		Op:    opDelete,
		Table: table,
		Where: conditions,
	}
	results, err := Transact(operation)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 0
	}
	return results[0].Count
}

// SelectRows check db.table with conditions existence
// return ResultRow and selected rows number
func SelectRows(table string,
	conditions []interface{}) ([]libovsdb.ResultRow, int) {
	operation := libovsdb.Operation{
		Op:    opSelect,
		Table: table,
		Where: conditions,
	}
	results, err := Transact(operation)
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
func Transact(ops ...libovsdb.Operation) ([]libovsdb.OperationResult, error) {
	// Only support one trans at same time now.
	DbnameClient.Tranmutex.Lock()
	defer DbnameClient.Tranmutex.Unlock()
	if DbnameClient.Client == nil {
		return nil, fmt.Errorf("DbnameClient not connected")
	}
	reply, err := DbnameClient.Client.Transact(DBNAME, ops...)
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
