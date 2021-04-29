package driver

import (
	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"
	"github.com/cn-pmlabs/govtep/tai"
)

type unosDriver struct {
	DriverName string
	ModuleAPIs map[tai.ObjID]moduleAPI
}

type moduleAPI interface {
	CreateObject(interface{}) error
	RemoveObject(interface{}) error
	AddObjectAttr(interface{}, map[interface{}]interface{}) error
	DelObjectAttr(interface{}, map[interface{}]interface{}) error
	SetObjectAttr(interface{}, map[interface{}]interface{}) error
	GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error)
	ListObject() ([]interface{}, error)
}

var unosDriverHandler = unosDriver{
	DriverName: "UNOS",
	ModuleAPIs: make(map[tai.ObjID]moduleAPI),
}

// Init UNOS TAI driver and unosconfig db
func Init() {
	cdb.InitUnosconfig(odbc.ConfigdbAddr)
	tai.RegisterTaiDriverHandler(unosDriverHandler.DriverName, &unosDriverHandler)

	unosDriverHandler.ModuleAPIs = map[tai.ObjID]moduleAPI{
		tai.ObjectIDBridge:          bdAPIs,
		tai.ObjectIDVrf:             vrfAPIs,
		tai.ObjectIDL2Port:          l2portAPIs,
		tai.ObjectIDL3Port:          l3portAPIs,
		tai.ObjectIDFDB:             fdbAPIs,
		tai.ObjectIDNeighbour:       neighbourAPIs,
		tai.ObjectIDRoute:           routeAPIs,
		tai.ObjectIDTunnel:          tunnelAPIs,
		tai.ObjectIDACL:             aclAPIs,
		tai.ObjectIDACLRule:         aclRuleAPIs,
		tai.ObjectIDPBR:             pbrAPIs,
		tai.ObjectIDAutoGatewayConf: autoGatewayConfAPIs,
	}

	return
}
