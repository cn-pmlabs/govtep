package tai

// ObjID define
type ObjID int

// ObjAttrID define
type ObjAttrID string

// TAI objects id
const (
	_ = iota
	ObjectIDBridge
	ObjectIDVrf
	ObjectIDL2Port
	ObjectIDL3Port
	ObjectIDFDB
	ObjectIDNeighbour
	ObjectIDRoute
	ObjectIDTunnel
	ObjectIDMcastFDB
	ObjectIDACL
	ObjectIDACLRule
	ObjectIDPBR
	ObjectIDAutoGatewayConf
)

// ObjectOrder is object name order
var ObjectOrder = []string{
	ObjectIDBridge:                    "Bridge",
	ObjectIDVrf:                       "Vrf",
	ObjectIDL2Port:                    "L2Port",
	ObjectIDL3Port:                    "L3Port",
	ObjectIDFDB:                       "FDB",
	ObjectIDNeighbour:                 "Neighbour",
	ObjectIDRoute:                     "Route",
	ObjectIDTunnel:                    "Tunnel",
	ObjectIDMcastFDB:                  "McastFDB",
	ObjectIDACL:                       "ACL",
	ObjectIDACLRule:                   "ACLRule",
	ObjectIDPBR:                       "PBR",
	ObjectIDAutoGatewayConf:           "AutoGatewayConf",
}
