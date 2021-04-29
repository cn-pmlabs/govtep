package ovsdbclient

import (
	"encoding/hex"
	"reflect"

	"github.com/ebay/libovsdb"
	"github.com/google/uuid"
)

// defeault ovsdb server listening socket addr
var (
	OvnnbAddr    string = "tcp:172.171.12.13:6641"
	OvnsbAddr    string = "tcp:172.171.12.13:6642"
	VtepdbAddr   string = "tcp:0.0.0.0:6644"
	ConfigdbAddr string = "tcp:0.0.0.0:6645"
)

// operation set
const (
	OpInsert string = "insert"
	OpMutate string = "mutate"
	OpDelete string = "delete"
	OpSelect string = "select"
	OpUpdate string = "update"
)

// DB name
const (
	OVNNB    string = "OVN_Northbound"
	OVNSB    string = "OVN_Southbound"
	VTEPDB   string = "CONTROLLER_VTEP"
	CONFIGDB string = "UNOS_config"
)

// NB Table name
const (
	NB_NBGlobal                    string = "NB_Global"
	NB_Logical_Switch              string = "Logical_Switch"
	NB_Logical_Switch_Port         string = "Logical_Switch_Port"
	NB_Forwarding_Group            string = "Forwarding_Group"
	NB_Address_Set                 string = "Address_Set"
	NB_Port_Group                  string = "Port_Group"
	NB_Load_Balancer               string = "Load_Balancer"
	NB_Load_Balancer_Health_Check  string = "Load_Balancer_Health_Check"
	NB_ACL                         string = "ACL"
	NB_QoS                         string = "QoS"
	NB_Meter                       string = "Meter"
	NB_Meter_Band                  string = "Meter_Band"
	NB_Logical_Router              string = "Logical_Router"
	NB_Logical_Router_Port         string = "Logical_Router_Port"
	NB_Logical_Router_Static_Route string = "Logical_Router_Static_Route"
	NB_Logical_Router_Policy       string = "Logical_Router_Policy"
	NB_NAT                         string = "NAT"
	NB_DHCP_Options                string = "DHCP_Options"
	NB_Connection                  string = "Connection"
	NB_DNS                         string = "DNS"
	NB_SSL                         string = "SSL"
	NB_Gateway_Chassis             string = "Gateway_Chassis"
	NB_HA_Chassis                  string = "HA_Chassis"
	NB_HA_Chassis_Group            string = "HA_Chassis_Group"
)

// NBTablesOrder ...
var NBTablesOrder = []string{
	NB_NBGlobal,
	NB_Logical_Switch,
	NB_Logical_Switch_Port,
	NB_Forwarding_Group,
	NB_Port_Group,
	NB_Load_Balancer,
	NB_Load_Balancer_Health_Check,
	NB_ACL,
	NB_QoS,
	NB_Meter,
	NB_Meter_Band,
	NB_Logical_Router,
	NB_Logical_Router_Port,
	NB_Logical_Router_Static_Route,
	NB_Logical_Router_Policy,
	NB_NAT,
	NB_DHCP_Options,
	NB_Connection,
	NB_DNS,
	NB_SSL,
	NB_Gateway_Chassis,
	NB_HA_Chassis,
	NB_HA_Chassis_Group,
}

// SB Table name
const (
	SB_SB_Global        string = "SB_Global"
	SB_Chassis          string = "Chassis"
	SB_Encap            string = "Encap"
	SB_Address_Set      string = "Address_Set"
	SB_Port_Group       string = "Port_Group"
	SB_Logical_Flow     string = "Logical_Flow"
	SB_Multicast_Group  string = "Multicast_Group"
	SB_Meter            string = "Meter"
	SB_Meter_Band       string = "Meter_Band"
	SB_Datapath_Binding string = "Datapath_Binding"
	SB_Port_Binding     string = "Port_Binding"
	SB_MAC_Binding      string = "MAC_Binding"
	SB_DHCP_Options     string = "DHCP_Options"
	SB_DHCPv6_Options   string = "DHCPv6_Options"
	SB_Connection       string = "Connection"
	SB_SSL              string = "SSL"
	SB_DNS              string = "DNS"
	SB_RBAC_Role        string = "RBAC_Role"
	SB_RBAC_Permission  string = "RBAC_Permission"
	SB_Gateway_Chassis  string = "Gateway_Chassis"
	SB_HA_Chassis       string = "HA_Chassis"
	SB_HA_Chassis_Group string = "HA_Chassis_Group"
	SB_Controller_Event string = "Controller_Event"
	SB_IP_Multicast     string = "IP_Multicast"
	SB_IGMP_Group       string = "IGMP_Group"
	SB_Service_Monitor  string = "Service_Monitor"
)

// SBTablesOrder ...
var SBTablesOrder = []string{
	SB_SB_Global,
	SB_Chassis,
	SB_Encap,
	SB_Address_Set,
	SB_Port_Group,
	SB_Logical_Flow,
	SB_Multicast_Group,
	SB_Meter,
	SB_Meter_Band,
	SB_Datapath_Binding,
	SB_Port_Binding,
	SB_MAC_Binding,
	SB_DHCP_Options,
	SB_DHCPv6_Options,
	SB_Connection,
	SB_SSL,
	SB_DNS,
	SB_RBAC_Role,
	SB_RBAC_Permission,
	SB_Gateway_Chassis,
	SB_HA_Chassis,
	SB_HA_Chassis_Group,
	SB_Controller_Event,
	SB_IP_Multicast,
	SB_IGMP_Group,
	SB_Service_Monitor,
}

// VTEPDB Table name
const (
	VTEP_Physical_Switch_Group string = "Physical_Switch_Group"
	VTEP_Physical_Switch       string = "Physical_Switch"
	VTEP_Locator               string = "Locator"
	VTEP_Bridge_Domain         string = "Bridge_Domain"
	VTEP_VRF                   string = "VRF"
	VTEP_L2Port                string = "L2Port"
	VTEP_L3Port                string = "L3Port"
	VTEP_Route                 string = "Route"
	VTEP_RemoteFDB             string = "RemoteFdb"
	VTEP_LocalFDB              string = "LocalFdb"
	VTEP_RemoteNeigh           string = "RemoteNeigh"
	VTEP_LocalNeigh            string = "LocalNeigh"
	VTEP_RemoteMcastfdb        string = "RemoteMcastfdb"
	VTEP_LocalMcastfdb         string = "LocalMcastfdb"
	VTEP_ACL                   string = "ACL"
	VTEP_ACL_Rule              string = "ACL_Rule"
)

// VTEPTablesOrder ...
var VTEPTablesOrder = []string{
	VTEP_Physical_Switch,
	VTEP_Locator,
	VTEP_Bridge_Domain,
	VTEP_VRF,
	VTEP_L2Port,
	VTEP_L3Port,
	VTEP_Route,
	VTEP_RemoteFDB,
	VTEP_LocalFDB,
	VTEP_RemoteNeigh,
	VTEP_LocalNeigh,
	VTEP_RemoteMcastfdb,
	VTEP_LocalMcastfdb,
	VTEP_ACL,
	VTEP_ACL_Rule,
}

// Float64ToInt libovsdb get interger by by float64
func Float64ToInt(row libovsdb.Row) {
	for field, value := range row.Fields {
		if v, ok := value.(float64); ok {
			n := int(v)
			if float64(n) == v {
				row.Fields[field] = n
			}
		}
	}
}

// RowUpdateOptimize convert float64 and save uuid to row field
func RowUpdateOptimize(rowUpdate libovsdb.RowUpdate, uuid string) libovsdb.RowUpdate {
	Float64ToInt(rowUpdate.New)
	Float64ToInt(rowUpdate.Old)

	if rowUpdate.New.Fields != nil {
		rowUpdate.New.Fields["_uuid"] = libovsdb.UUID{GoUUID: uuid}
	}
	if rowUpdate.Old.Fields != nil {
		rowUpdate.Old.Fields["_uuid"] = libovsdb.UUID{GoUUID: uuid}
	}

	return rowUpdate
}

// StringToGoUUID convert uuid string to libovsdb.UUID
func StringToGoUUID(uuid string) libovsdb.UUID {
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
func NewRowUUID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	var buf [36 + 3]byte
	copy(buf[:], "row")
	encodeHex(buf[3:], id)
	return string(buf[:]), nil
}

// NewUUIDString generate a random UUID string
func NewUUIDString() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

// ConvertGoSetToStringArray get string array from OvsSet
func ConvertGoSetToStringArray(oset libovsdb.OvsSet) []string {
	var ret = []string{}
	for _, s := range oset.GoSet {
		value, ok := s.(string)
		if ok {
			ret = append(ret, value)
		}
	}
	return ret
}

// GetRowUpdateOp get the update operation
func GetRowUpdateOp(rowUpdate libovsdb.RowUpdate) string {
	var op string
	empty := libovsdb.Row{}
	if !reflect.DeepEqual(rowUpdate.New, empty) {
		if reflect.DeepEqual(rowUpdate.Old, empty) {
			op = OpInsert
		} else {
			op = OpUpdate
		}
	} else {
		if !reflect.DeepEqual(rowUpdate.Old, empty) {
			op = OpDelete
		}
	}
	return op
}
