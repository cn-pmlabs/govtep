package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// IntMax ...
const IntMax = int(^uint(0) >> 1)

var schemaFile string = "ovsdb.ovsschema"
var libDir string = "../../lib/odbapi/"
var help bool = false

func usage() {
	fmt.Fprintf(os.Stderr, `odbgen
Usage: odbgen [-h] [-f dbSchemaFile]

Options:
`)
	flag.PrintDefaults()
}

func init() {
	flag.StringVar(&schemaFile, "f", schemaFile, "connect to vtep database")
	flag.StringVar(&libDir, "d", libDir, "lib ovsdb api dir")
	flag.BoolVar(&help, "h", false, "display this help message")
	flag.Usage = usage
}

// DatabaseSchema is a database schema according to RFC7047
type DatabaseSchema struct {
	Name    string                 `json:"name"`
	Version string                 `json:"version"`
	Tables  map[string]TableSchema `json:"tables"`
}

// TableSchema is a table schema according to RFC7047
type TableSchema struct {
	Columns map[string]ColumnSchema `json:"columns"`
	Indexes [][]string              `json:"indexes,omitempty"`
	IsRoot  bool                    `json:"isRoot,omitempty"`
	MaxRows int                     `json:"maxRows,omitempty"`
}

// ColumnSchema is a column schema according to RFC7047
type ColumnSchema struct {
	Type      interface{} `json:"type"`
	Ephemeral bool        `json:"ephemeral,omitempty"`
	Mutable   bool        `json:"mutable,omitempty"`
}

func schemaCheck() (string, error) {
	tempDBFileName := "temp.db"
	os.Remove(tempDBFileName)

	cmd := "sudo ovsdb-tool create " + tempDBFileName + " " + schemaFile
	out, err := execShell(cmd)
	os.Remove(tempDBFileName)

	if err != nil {
		fmt.Println("remove file ,err ", err)
	}

	return out, err
}

func fileGetContents(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

func capitalize(str string) string {
	var upperStr string
	vv := []rune(str)
	for i := 0; i < len(vv); i++ {
		if i == 0 {
			if vv[i] >= 97 && vv[i] <= 122 {
				vv[i] -= 32
				upperStr += string(vv[i])
			} else {
				return str
			}
		} else {
			upperStr += string(vv[i])
		}
	}
	return upperStr
}

func capitalizeEachWord(str string) string {
	var processedStr string
	str = strings.ToLower(str)
	words := strings.Split(str, "_")
	for _, word := range words {
		if word == "ip" || word == "id" || word == "acl" || word == "tcp" || word == "udp" || word == "dns" {
			processedStr += strings.ToUpper(word)
		} else {
			processedStr += capitalize(word)
		}
	}
	return processedStr
}

func capitalizeEachWordDelDash(str string) string {
	var processedStr string
	str = strings.ToLower(str)
	words := strings.Split(str, "_")
	for _, word := range words {
		if word == "ip" || word == "id" || word == "acl" || word == "tcp" || word == "udp" || word == "dns" {
			processedStr += strings.ToUpper(word)
		} else {
			processedStr += capitalize(word)
		}
	}
	processedStr = strings.Replace(processedStr, "-", "", -1)
	processedStr = strings.Replace(processedStr, "|", "", -1)
	return processedStr
}

func execShell(s string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", s)
	var out bytes.Buffer
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func copyFile(src, des string) (written int64, err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	fi, _ := srcFile.Stat()
	perm := fi.Mode()

	desFile, err := os.OpenFile(des, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return 0, err
	}
	defer desFile.Close()

	return io.Copy(desFile, srcFile)
}

func odbgen(c DatabaseSchema) {
	var err error
	// Get dbname
	Dbname := c.Name
	fmt.Printf("DB: %v\n", Dbname)
	DbnameLowCase := strings.Replace(strings.ToLower(Dbname), "_", "", -1)
	DbnameUpCase := strings.Replace(strings.ToUpper(Dbname), "_", "", -1)
	dbDir := libDir + DbnameLowCase

	// mkdir
	err = os.RemoveAll(dbDir)
	err = os.MkdirAll(dbDir, os.ModePerm)

	var defineFileStr string
	var tableFileStrs []string

	defineFileStr += "package " + DbnameLowCase + "\n"
	defineFileStr += "// DB name\n"
	defineFileStr += "const (\n\t" + DbnameUpCase + " string = \"" + Dbname + "\"\n)\n"
	defineFileStr += "// Table name\n"
	defineFileStr += "const (\n"

	// gen odbinit.go
	odbinitfileTemp, err := ioutil.ReadFile("dbname/odbinit.go")
	if err != nil {
		return
	}
	odbinitfileStr := string(odbinitfileTemp)
	odbinitfileStr = strings.Replace(odbinitfileStr, "dbname", DbnameLowCase, -1)
	odbinitfileStr = strings.Replace(odbinitfileStr, "DBNAME", DbnameUpCase, -1)
	odbinitfileStr = strings.Replace(odbinitfileStr, "Dbname", capitalize(DbnameLowCase), -1)
	odbinitfile, err := os.Create(dbDir + "/odbinit.go")
	if err != nil {
		fmt.Printf("odbinit create error: %v\n", err)
		return
	}
	defer odbinitfile.Close()
	odbinitfile.WriteString(odbinitfileStr)

	// gen odbop.go
	odbopfileTemp, err := ioutil.ReadFile("dbname/odbop.go")
	if err != nil {
		return
	}
	odbopfileStr := string(odbopfileTemp)
	odbopfileStr = strings.Replace(odbopfileStr, "dbname", DbnameLowCase, -1)
	odbopfileStr = strings.Replace(odbopfileStr, "DBNAME", DbnameUpCase, -1)
	odbopfileStr = strings.Replace(odbopfileStr, "Dbname", capitalize(DbnameLowCase), -1)
	odbopfile, err := os.Create(dbDir + "/odbop.go")
	if err != nil {
		fmt.Printf("odbop create error: %v\n", err)
		return
	}
	defer odbopfile.Close()
	odbopfile.WriteString(odbopfileStr)

	// copy common.go
	commonfileTemp, err := ioutil.ReadFile("dbname/common.go")
	if err != nil {
		return
	}
	commonfileStr := string(commonfileTemp)
	commonfileStr = strings.Replace(commonfileStr, "dbname", DbnameLowCase, -1)
	commonfile, err := os.Create(dbDir + "/common.go")
	if err != nil {
		fmt.Printf("common create error: %v\n", err)
		return
	}
	defer commonfile.Close()
	commonfile.WriteString(commonfileStr)

	tableFiledDefaultMap := make(map[string]map[string]interface{})
	tableUUIDColumns := make(map[string]map[string]int)

	// Gen ovsdb table code
	for tablename, table := range c.Tables {
		fmt.Printf("Gen code for table %v\n", tablename)
		tablenamenosep := capitalizeEachWord(tablename)

		defineFileStr += "\t" + tablenamenosep + " string = " + "\"" + tablename + "\"\n"

		var tablestruct string

		tablestruct += "// Table" + tablenamenosep + " definition\n"
		tablestruct += "type Table" + tablenamenosep + " struct {\n"
		tablestruct += "\tUUID string\n"

		var tableFields string
		tableFields += "// " + tablenamenosep + "Fields name\n"
		tableFields += "const (\n"
		tableFields += "\t" + tablenamenosep + "Field" + "UUID" + " string = " + "\"" + "_uuid" + "\"\n"

		var fieldMapToColumn string
		fieldMapToColumn += "// " + tablenamenosep + "FieldMapToColumn map field name to columns\n"
		fieldMapToColumn += "var " + tablenamenosep + "FieldMapToColumn map[string]string = map[string]string{\n"
		fieldMapToColumn += "\t\"" + "UUID" + "\" : " + tablenamenosep + "Field" + "UUID" + ",\n"

		//var fieldsets []string
		var fieldadddelvalues []string
		var fieldsetdelkeys []string

		// decide table code template file
		tableCodeTmpFile := "dbname/template/tablecode"
		adddelvalTmpFile := "dbname/template/adddelval"
		setdelkeyTmpFile := "dbname/template/setdelkey"
		addreftblTmpFile := "dbname/template/addreftbl"
		rowtostructTmpFile := "dbname/template/rowtostruct"
		if table.MaxRows == 1 {
			tableCodeTmpFile += "_global"
			if table.IsRoot == false {
				tableCodeTmpFile += "_nonroot"
			}
			adddelvalTmpFile += "_global"
			setdelkeyTmpFile += "_global"
			addreftblTmpFile += "_global"
		} else if table.IsRoot == false {
			tableCodeTmpFile += "_nonroot"
		}

		columnKeyType := make(map[string]string)

		// for table field enum and range code
		tableFieldEnum := make(map[string]string)
		tableFieldRange := make(map[string]string)

		filedDefaultMap := make(map[string]interface{})
		fieldDefaultExist := false
		var fieldDefault string
		fieldDefault += "// " + tablenamenosep + "fields default value\n"
		fieldDefault += "const (\n"

		UUIDColumns := make(map[string]int)
		UUIDColumnsExist := false

		for columnname, column := range table.Columns {
			isSetColumn := false
			isMapColumn := false
			min := 1
			max := 1
			var keyType string
			var valType string
			var refTable string

			haveRangeMin := false
			haveRangeMax := false
			rangeMin := 0
			rangeMax := 0
			haveEnum := false
			var enumSet []interface{}

			// TABLENAME fields
			Columnname := capitalizeEachWord(columnname)
			tableFields += "\t" + tablenamenosep + "Field" + Columnname + " string = " + "\"" + columnname + "\"\n"
			// fieldMapToColumn to do reflect
			fieldMapToColumn += "\t\"" + Columnname + "\" : " + tablenamenosep + "Field" + Columnname + ",\n"

			switch column.Type.(type) {
			case string:
				switch column.Type {
				case "string":
					tablestruct += "\t" + Columnname + " string\n"
					keyType = "string"
				case "integer":
					tablestruct += "\t" + Columnname + " int\n"
					keyType = "int"
				case "boolean":
					tablestruct += "\t" + Columnname + " bool\n"
					keyType = "bool"
				default:
					fmt.Printf("type %v unknown\n", Columnname)
					return
				}
			case interface{}:
				for typekey, typevalue := range column.Type.(map[string]interface{}) {
					switch typekey {
					case "key":
						isSetColumn = true

						switch typevalue.(type) {
						case string:
							keyType = typevalue.(string)
							switch keyType {
							case "string":
								keyType = "string"
							case "integer":
								keyType = "int"
							case "boolean":
								keyType = "bool"
							case "uuid":
								// uuid type should have refTable (to make sense)
								keyType = "libovsdb.UUID"
							default:
								fmt.Printf("type %v unknown\n", Columnname)
								return
							}
						case interface{}:
							var defaultValue interface{}
							defV := false
							for k, v := range typevalue.(map[string]interface{}) {
								switch k {
								case "type":
									keyType = v.(string)
									switch keyType {
									case "string":
										keyType = "string"
									case "integer":
										keyType = "int"
									case "boolean":
										keyType = "bool"
									case "uuid":
										keyType = "libovsdb.UUID"
									default:
										fmt.Printf("type %v unknown\n", Columnname)
										return
									}
								case "refTable":
									refTable = v.(string)
								case "minInteger":
									haveRangeMin = true
									rangeMin = int(v.(float64))
								case "maxInteger":
									haveRangeMax = true
									rangeMax = int(v.(float64))
								case "enum":
									haveEnum = true
									enumSet = v.([]interface{})[1].([]interface{})
								case "default":
									fieldDefaultExist = true
									defV = true
									defaultValue = v
								}
							}
							if defV {
								switch keyType {
								case "string":
									fieldDefault += tablenamenosep + "Default" + Columnname + " string = " + "\"" + defaultValue.(string) + "\"" + "\n"
								case "int":
									fieldDefault += tablenamenosep + "Default" + Columnname + " int = " + strconv.Itoa(int(defaultValue.(float64))) + "\n"
								case "bool":
									if defaultValue.(bool) == true {
										fieldDefault += tablenamenosep + "Default" + Columnname + " bool = " + "true" + "\n"
									} else {
										fieldDefault += tablenamenosep + "Default" + Columnname + " bool = " + "false" + "\n"
									}
								}
								filedDefaultMap[columnname] = defaultValue
							}
						}
					case "value":
						isMapColumn = true

						switch typevalue.(type) {
						case string:
							valType = typevalue.(string)
							switch valType {
							case "string":
								valType = "string"
							case "integer":
								valType = "int"
							case "boolean":
								valType = "bool"
							case "uuid":
								valType = "libovsdb.UUID"
							default:
								fmt.Printf("type %v unknown\n", Columnname)
								return
							}
						case interface{}:
							for k, v := range typevalue.(map[string]interface{}) {
								switch k {
								case "type":
									valType = v.(string)
									switch valType {
									case "string":
										valType = "string"
									case "integer":
										valType = "int"
									case "boolean":
										valType = "bool"
									case "uuid":
										valType = "libovsdb.UUID"
									default:
										fmt.Printf("type %v unknown\n", Columnname)
										return
									}
								case "minInteger":
									haveRangeMin = true
									rangeMin = int(v.(float64))
								case "maxInteger":
									haveRangeMax = true
									rangeMax = int(v.(float64))
								case "enum":
									haveEnum = true
									enumSet = v.([]interface{})[1].([]interface{})
								}
							}
						}
					case "min":
						min = int(typevalue.(float64))
						if min > 1 {
							fmt.Printf("min must be exactly 0 or exactly 1")
							return
						}
					case "max":
						switch typevalue.(type) {
						case string:
							if typevalue != "unlimited" {
								fmt.Printf("unsupported max of %v\n", typevalue)
								return
							}
							max = IntMax
						case float64:
							max = int(typevalue.(float64))
							if max < 1 {
								fmt.Printf("max must be at least 1")
								return
							}
						default:
							fmt.Printf("max %v unknown\n", Columnname)
							return
						}
					case "default":
						fieldDefaultExist = true
						filedDefaultMap[columnname] = typevalue
					default:
						fmt.Printf("type %v define error\n", Columnname)
						return
					}
				}
				// detailed key value type
				if isMapColumn == true {
					//tablestruct += "\t" + Columnname + " map[interface{}]interface{}\n"
					tablestruct += "\t" + Columnname + " map[interface{}]interface{}\n"
				} else {
					//tablestruct += "\t" + Columnname + " []interface{}\n"
					if max == 1 && min == 1 {
						tablestruct += "\t" + Columnname + " " + keyType + "\n"
					} else {
						tablestruct += "\t" + Columnname + " []" + keyType + "\n"
					}
				}
			default:
				fmt.Printf("unknown %v\n", Columnname)
				return
			}

			columnKeyType[columnname] = keyType

			/*fieldset, err := ioutil.ReadFile("dbname/template/fieldset.txt")
			if err != nil {
				return
			}
			fieldsetStr := string(fieldset)
			fieldsetStr = strings.Replace(fieldsetStr, "TABLENAME", tablenamenosep, -1)
			fieldsetStr = strings.Replace(fieldsetStr, "FIELD", Columnname, -1)
			fieldsets = append(fieldsets, fieldsetStr)*/

			// set column
			if isSetColumn == true && isMapColumn == false {
				fieldadddelvalue, err := ioutil.ReadFile(adddelvalTmpFile)
				if err != nil {
					fmt.Printf("Read %s failed\n", adddelvalTmpFile)
					return
				}
				fieldadddelvalueStr := string(fieldadddelvalue)
				fieldadddelvalueStr = strings.Replace(fieldadddelvalueStr, "TABLENAME", tablenamenosep, -1)
				fieldadddelvalueStr = strings.Replace(fieldadddelvalueStr, "FIELD", Columnname, -1)
				fieldadddelvalueStr = strings.Replace(fieldadddelvalueStr, "TYPE", keyType, -1)
				fieldadddelvalues = append(fieldadddelvalues, fieldadddelvalueStr)

				if len(refTable) > 0 {
					refTbl := c.Tables[refTable]
					if refTbl.IsRoot == false {
						refTablenosep := capitalizeEachWord(refTable)
						fieldaddreftbl, err := ioutil.ReadFile(addreftblTmpFile)
						if err != nil {
							fmt.Printf("Read %s failed\n", adddelvalTmpFile)
							return
						}
						fieldaddreftblStr := string(fieldaddreftbl)
						fieldaddreftblStr = strings.Replace(fieldaddreftblStr, "TABLENAMEREF", refTablenosep, -1)
						fieldaddreftblStr = strings.Replace(fieldaddreftblStr, "TABLENAME", tablenamenosep, -1)
						fieldaddreftblStr = strings.Replace(fieldaddreftblStr, "FIELD", Columnname, -1)
						fieldadddelvalues = append(fieldadddelvalues, fieldaddreftblStr)
					}
				}
			}

			// map column
			if isSetColumn == true && isMapColumn == true {
				fieldsetdelkey, err := ioutil.ReadFile(setdelkeyTmpFile)
				if err != nil {
					fmt.Printf("Read %s failed\n", setdelkeyTmpFile)
					return
				}
				fieldsetdelkeyStr := string(fieldsetdelkey)
				fieldsetdelkeyStr = strings.Replace(fieldsetdelkeyStr, "TABLENAME", tablenamenosep, -1)
				fieldsetdelkeyStr = strings.Replace(fieldsetdelkeyStr, "FIELD", Columnname, -1)
				fieldsetdelkeyStr = strings.Replace(fieldsetdelkeyStr, "KEY", keyType, -1)
				fieldsetdelkeyStr = strings.Replace(fieldsetdelkeyStr, "VALUE", valType, -1)
				fieldsetdelkeys = append(fieldsetdelkeys, fieldsetdelkeyStr)
			}

			if haveRangeMin || haveRangeMax {
				if haveRangeMin {
					tableFieldRange[Columnname] += tablenamenosep + Columnname + "Min int = " + strconv.Itoa(rangeMin) + "\n"
				}
				if haveRangeMax {
					tableFieldRange[Columnname] += tablenamenosep + Columnname + "Max int = " + strconv.Itoa(rangeMax) + "\n"
				}
			}
			if haveEnum {
				for _, enum := range enumSet {
					val := capitalizeEachWordDelDash(enum.(string))
					tableFieldEnum[Columnname] += tablenamenosep + Columnname + val + " string = " + "\"" + enum.(string) + "\"" + "\n"
				}
			}

			// for uuid columns
			if keyType == "libovsdb.UUID" {
				UUIDColumns[columnname] = 1
				UUIDColumnsExist = true
			}
		}

		tablestruct += "}\n"
		tableFields += ")\n"
		fieldMapToColumn += "}\n"

		var tableindexes []string
		// gen table indexes struct
		for i, index := range table.Indexes {
			var indexname string
			var tableindex string
			if i == 0 {
				indexname = tablenamenosep + "Index"
			} else {
				indexname = tablenamenosep + "Index" + strconv.Itoa(i)
			}

			tableindex += "// " + indexname + " definition\n"
			tableindex += "type " + indexname + " struct {\n"
			for _, name := range index {
				tableindex += "\t" + capitalizeEachWord(name) + " " + columnKeyType[name] + "\n"
			}
			tableindex += "}\n"
			tableindexes = append(tableindexes, tableindex)
		}
		// check global table(maxrow=1) don't have indexes
		if table.MaxRows == 1 && len(tableindexes) > 0 {
			fmt.Printf("Error: global table%v (maxrow=1) don't have indexes\n", tablename)
			return
		}
		// check not global table need indexes
		if table.MaxRows > 1 || table.MaxRows == 0 {
			// add UUID index for all non-global table
			//if 0 == len(tableindexes) {
			indexname := tablenamenosep + "UUIDIndex"
			tableindex := "// " + indexname + " definition\n"
			tableindex += "type " + indexname + " struct {\n"
			tableindex += "\t" + "UUID" + " " + "string" + "\n"
			tableindex += "}\n"
			tableindexes = append(tableindexes, tableindex)
		}

		// table code gen
		tablecode, err := ioutil.ReadFile(tableCodeTmpFile)
		if err != nil {
			fmt.Printf("Read %s failed\n", tableCodeTmpFile)
			return
		}
		tablecodeStr := string(tablecode)
		tablecodeStr = strings.Replace(tablecodeStr, "TABLENAME", tablenamenosep, -1)

		var tableFileStr string
		tableFileStr += "package " + DbnameLowCase + "\n"
		tableFileStr += "import (\n"
		tableFileStr += "\t\"fmt\"\n"
		tableFileStr += "\t\"reflect\"\n\n"
		tableFileStr += "\t\"github.com/ebay/libovsdb\"\n)\n"

		// write tablestruct
		tableFileStr += "\n" + tablestruct + "\n"
		// write tableindex
		for _, indexcode := range tableindexes {
			tableFileStr += "\n" + indexcode + "\n"
		}
		// write fieldname
		tableFileStr += "\n" + tableFields + "\n"
		// write fieldMapToColumn
		tableFileStr += "\n" + fieldMapToColumn + "\n"

		if fieldDefaultExist {
			tableFiledDefaultMap[tablename] = filedDefaultMap
			fieldDefault += ")\n"
			tableFileStr += "\n" + fieldDefault + "\n"
		}

		// for uuid columns
		if UUIDColumnsExist {
			tableUUIDColumns[tablename] = UUIDColumns
		}

		// write field range and enum
		for column, crange := range tableFieldRange {
			tableFileStr += "// " + column + " range\n"
			tableFileStr += "const (\n" + crange + ")\n"
		}
		for column, cnum := range tableFieldEnum {
			tableFileStr += "// " + column + " enum\n"
			tableFileStr += "const (\n" + cnum + ")\n"
		}

		// write tablecode
		tableFileStr += "\n" + tablecodeStr + "\n"
		/*// write field set
		for _, fieldsetcode := range fieldsets {
			tableFileStr += "\n" + fieldsetcode + "\n"
		}*/
		// write add del value code
		for _, fieldadddelvaluecode := range fieldadddelvalues {
			tableFileStr += "\n" + fieldadddelvaluecode + "\n"
		}
		// wirte set del key code
		for _, fieldsetdelkeycode := range fieldsetdelkeys {
			tableFileStr += "\n" + fieldsetdelkeycode + "\n"
		}

		// convert row to struct
		rowToStruct, err := ioutil.ReadFile(rowtostructTmpFile)
		if err != nil {
			fmt.Printf("Read %s failed\n", rowtostructTmpFile)
			return
		}
		rowToStructStr := string(rowToStruct)
		rowToStructStr = strings.Replace(rowToStructStr, "TABLENAME", tablenamenosep, -1)
		tableFileStr += "\n" + rowToStructStr + "\n"

		tableFile, err := os.Create(dbDir + "/table_" + strings.ToLower(tablename) + ".go")
		if err != nil {
			fmt.Printf("%v", err)
		}
		defer tableFile.Close()
		tableFile.WriteString(tableFileStr)

		tableFileStrs = append(tableFileStrs, tableFileStr)
	}

	defineFileStr += ")\n"
	// write define code
	defineFile, _ := os.Create(dbDir + "/define.go")
	defer defineFile.Close()
	defineFile.WriteString(defineFileStr)

	// for FiledsDefaultMap
	FiledDefaultMap := capitalize(DbnameLowCase) + "FiledsDefaultMap"
	tableFiledDefaultMapStr := "// " + FiledDefaultMap + " table fileds default value map\n"
	tableFiledDefaultMapStr += "var " + FiledDefaultMap + " = " + "make(map[string]map[string]interface{})\n"
	defineFile.WriteString(tableFiledDefaultMapStr)

	var FiledDefaultMapStr string
	for tb, tbDef := range tableFiledDefaultMap {
		tbDefaultMap := capitalizeEachWord(tb) + "FiledsDefaultMap"
		tbDefaultMapStr := "var " + tbDefaultMap + " = " + "make(map[string]interface{})\n"
		for field, fieldDef := range tbDef {
			var tmpStr string
			if fieldStr, ok := fieldDef.(string); ok {
				tmpStr = fmt.Sprintf(tbDefaultMap + "[\"" + field + "\"] = " + "\"" + fieldStr + "\"\n")
			} else if _, ok := fieldDef.(float64); ok {
				tmpStr = fmt.Sprintf(tbDefaultMap+"[\""+field+"\"] = %v\n", fieldDef)
			} else if _, ok := fieldDef.(bool); ok {
				tmpStr = fmt.Sprintf(tbDefaultMap+"[\""+field+"\"] = %v\n", fieldDef)
			} else {
				kMapStr := "var " + capitalizeEachWord(field) + "Map = " + "make(map[interface{}]interface{})\n"
				var kMapVal string
				for k, v := range fieldDef.(map[string]interface{}) {
					kMapVal += capitalizeEachWord(field) + "Map[\"" + k + "\"] = "
					if va, ok := v.(string); ok {
						kMapVal += "\"" + va + "\"\n"
					} else {
						kMapVal += fmt.Sprintf("%v\n", v)
					}
				}

				tmpStr += kMapStr
				tmpStr += kMapVal
				tmpStr += fmt.Sprintf(tbDefaultMap + "[\"" + field + "\"] = " + capitalizeEachWord(field) + "Map" + "\n")
			}
			tbDefaultMapStr += tmpStr
		}
		FiledDefaultMapStr += tbDefaultMapStr
		FiledDefaultMapStr += FiledDefaultMap + "[\"" + tb + "\"] = " + tbDefaultMap + "\n"
	}

	defaultInitFunc := "// " + capitalizeEachWord(DbnameLowCase) + "FiledsDefaultMapInit init fields default value mapping\n"
	defaultInitFunc += "func " + capitalizeEachWord(DbnameLowCase) + "FiledsDefaultMapInit() {"
	defaultInitFunc += FiledDefaultMapStr + "}\n"
	defineFile.WriteString(defaultInitFunc)

	// for uuid columns map
	UUIDColumnsStr := "// " + "TableUUIDColumns mapping\n"
	UUIDColumnsStr += "var " + "TableUUIDColumns " + " = " + "map[string]map[string]int {\n"
	for table, UUIDColumns := range tableUUIDColumns {
		UUIDColumnsStr += "\"" + table + "\"" + " : {\n"
		for field := range UUIDColumns {
			UUIDColumnsStr += "\"" + field + "\"" + " : 1,\n"
		}
		UUIDColumnsStr += " },\n"
	}
	UUIDColumnsStr += " }\n"
	defineFile.WriteString(UUIDColumnsStr)

	// gofmt
	gofmtCmd := "go fmt " + dbDir + "/*.go"
	execShell(gofmtCmd)

	fmt.Println("Done!")
}

func main() {
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}

	var c DatabaseSchema
	var content []byte
	var err error

	out, err := schemaCheck()
	if err != nil {
		fmt.Printf("Something error in %s\n", schemaFile)
		fmt.Printf("Detail: %s", out[12:])
		return
	}

	content, err = fileGetContents(schemaFile)
	if err != nil {
		fmt.Println("open file error: " + err.Error())
		return
	}
	err = json.Unmarshal([]byte(content), &c)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return
	}
	odbgen(c)
	//ovsdb.InitOvsdb("aaa")
}
