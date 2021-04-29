package dbname

import (
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/ebay/libovsdb"
	"github.com/google/uuid"
)

// operation set
const (
	opInsert string = "insert"
	opMutate string = "mutate"
	opDelete string = "delete"
	opSelect string = "select"
	opUpdate string = "update"
)

// InvalidUUID used to select all rows in table
const InvalidUUID string = "00000000-0000-0000-0000-000000000000"

// Float64ToInt libovsdb get interger by by float64
func float64ToInt(row map[string]interface{}) {
	for field, value := range row {
		if v, ok := value.(float64); ok {
			n := int(v)
			if float64(n) == v {
				row[field] = n
			}
		}
	}
}

// StringToGoUUID convert uuid string to libovsdb.UUID
func stringToGoUUID(uuid string) libovsdb.UUID {
	return libovsdb.UUID{GoUUID: uuid}
}

func encodeHex(dst []byte, id uuid.UUID) {
	hex.Encode(dst, id[:4])
	dst[8] = '_'
	hex.Encode(dst[9:13], id[4:6])
	dst[13] = '_'
	hex.Encode(dst[14:18], id[6:8])
	dst[18] = '_'
	hex.Encode(dst[19:23], id[8:10])
	dst[23] = '_'
	hex.Encode(dst[24:], id[10:])
}

// NewRowUUID generate a random UUID
func newRowUUID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	var buf [36 + 3]byte
	copy(buf[:], "row")
	encodeHex(buf[3:], id)
	return string(buf[:]), nil
}

// convertGoSetToArray get string array from OvsSet
func convertGoSetToArray(oset libovsdb.OvsSet) []interface{} {
	var ret []interface{}
	for _, s := range oset.GoSet {
		value, ok := s.(interface{})
		if ok {
			ret = append(ret, value)
		}
	}
	return ret
}

// ConvertTableToRow table struct to row map
func ConvertTableToRow(table interface{}, fieldMap map[string]string) (map[string]interface{}, error) {
	row := make(map[string]interface{})

	typ := reflect.TypeOf(table)
	val := reflect.ValueOf(table)

	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Name == "UUID" {
			continue
		}
		switch val.Field(i).Interface().(type) {
		case libovsdb.UUID:
			if uuid, ok := val.Field(i).Interface().(libovsdb.UUID); ok {
				if uuid.GoUUID == "" {
					continue
				}
			}
			row[fieldMap[typ.Field(i).Name]] = val.Field(i).Interface()
		case string, int, bool:
			row[fieldMap[typ.Field(i).Name]] = val.Field(i).Interface()
		case []interface{}, []string, []int, []bool, []libovsdb.UUID:
			if val.Field(i).Len() == 0 {
				continue
			}
			oSet, err := libovsdb.NewOvsSet(val.Field(i).Interface())
			if err != nil {
				return nil, fmt.Errorf("OvsSet trans error for %s", typ.Field(i).Name)
			}
			row[fieldMap[typ.Field(i).Name]] = oSet
		case map[interface{}]interface{}, map[int]int, map[int]string, map[int]bool,
			map[string]int, map[string]string, map[string]bool,
			map[bool]int, map[bool]string, map[bool]bool:
			if val.Field(i).Len() == 0 {
				continue
			}
			oMap, err := libovsdb.NewOvsMap(val.Field(i).Interface())
			if err != nil {
				return nil, fmt.Errorf("OvsMap trans error for %s", typ.Field(i).Name)
			}
			row[fieldMap[typ.Field(i).Name]] = oMap
		default:
			return nil, fmt.Errorf("Unsupported type for %s", typ.Field(i).Name)
		}
	}

	return row, nil
}

func convertFieldToRow(fieldName string, field interface{}) (map[string]interface{}, error) {
	row := make(map[string]interface{})

	switch field.(type) {
	case string, int, bool, libovsdb.UUID:
		row[fieldName] = field
	case []interface{}, []string, []int, []bool, []libovsdb.UUID:
		oSet, err := libovsdb.NewOvsSet(field)
		if err != nil {
			return nil, fmt.Errorf("OvsSet trans error for %s", fieldName)
		}
		row[fieldName] = oSet
	case map[interface{}]interface{}, map[int]int, map[int]string, map[int]bool,
		map[string]int, map[string]string, map[string]bool,
		map[bool]int, map[bool]string, map[bool]bool:
		oMap, err := libovsdb.NewOvsMap(field)
		if err != nil {
			return nil, fmt.Errorf("OvsMap trans error for %s", fieldName)
		}
		row[fieldName] = oMap
	default:
		return nil, fmt.Errorf("Unsupported type for %s", fieldName)
	}

	return row, nil
}

func convertIndexToConditions(tableIndex interface{}, fieldMap map[string]string) ([]interface{}, error) {
	var conditions []interface{}
	typ := reflect.TypeOf(tableIndex)
	val := reflect.ValueOf(tableIndex)

	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Name == "UUID" {
			conditions = append(conditions, libovsdb.
				NewCondition(fieldMap[typ.Field(i).Name], "==",
					stringToGoUUID(val.Field(i).Interface().(string))))
			continue
		}
		conditions = append(conditions, libovsdb.
			NewCondition(fieldMap[typ.Field(i).Name], "==", val.Field(i).Interface()))
	}
	return conditions, nil
}

// convertOvsSetToStringArray get string array from OvsSet
func convertOvsSetToStringArray(oset libovsdb.OvsSet) []string {
	var ret = []string{}
	for _, s := range oset.GoSet {
		value, ok := s.(string)
		if ok {
			ret = append(ret, value)
		}
	}
	return ret
}

// convertOvsSetToIntArray get int array from OvsSet
func convertOvsSetToIntArray(oset libovsdb.OvsSet) []int {
	var ret = []int{}
	for _, s := range oset.GoSet {
		value, ok := s.(float64)
		if ok {
			ret = append(ret, int(value))
		}
	}
	return ret
}

// convertOvsSetToBoolArray get bool array from OvsSet
func convertOvsSetToBoolArray(oset libovsdb.OvsSet) []bool {
	var ret = []bool{}
	for _, s := range oset.GoSet {
		value, ok := s.(bool)
		if ok {
			ret = append(ret, value)
		}
	}
	return ret
}

// convertOvsSetToUUIDArray get uuid array from OvsSet
func convertOvsSetToUUIDArray(oset libovsdb.OvsSet) []libovsdb.UUID {
	var ret = []libovsdb.UUID{}
	for _, s := range oset.GoSet {
		value, ok := s.(libovsdb.UUID)
		if ok {
			ret = append(ret, value)
		}
	}
	return ret
}
