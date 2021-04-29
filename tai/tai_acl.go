package tai

import (
	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"

	"github.com/cn-pmlabs/govtep/lib/log"

	"github.com/ebay/libovsdb"
)

// ACLAttrName ...
const (
	ACLAttrName  = "acl_name"
	ACLAttrPorts = "acl_ports"
	ACLAttrStage = "acl_stage"
	ACLAttrType  = "acl_type"
	ACLAttrRules = "acl_rules"
)

// ACLObj ...
type ACLObj struct {
	ACLName string
}

func rowToACLObj(row libovsdb.Row) (interface{}, map[interface{}]interface{}) {
	tableACL := vtepdb.ConvertRowToACL(libovsdb.ResultRow(row.Fields))

	obj := ACLObj{
		ACLName: tableACL.Name,
	}

	var aclRulesSequence []int
	if len(tableACL.ACLRules) != 0 {
		for _, aclRuleUUID := range tableACL.ACLRules {
			tableACLRule, err := vtepdb.ACLRuleGetByUUID(aclRuleUUID.GoUUID)
			if err != nil {
				log.Warning("ACLRule %s for ACL %s not exist", aclRuleUUID.GoUUID, tableACL.Name)
				continue
			}
			aclRulesSequence = append(aclRulesSequence, tableACLRule.Sequence)
		}
	}

	attrs := map[interface{}]interface{}{
		ACLAttrName:  tableACL.Name,
		ACLAttrPorts: tableACL.Ports,
		ACLAttrStage: tableACL.Stage,
		ACLAttrType:  tableACL.Type,
		ACLAttrRules: aclRulesSequence,
	}
	return obj, attrs
}

func rowToACLAttrs(row libovsdb.Row) map[interface{}]interface{} {
	attrs := make(map[interface{}]interface{})

	for attr, value := range row.Fields {
		switch attr {
		// acl field name & type & stage can't be updated
		case vtepdb.ACLFieldPorts:
			if port, ok := value.(string); ok {
				attrs[ACLAttrPorts] = port
			}
		case vtepdb.ACLFieldACLRules:
			if aclRules, ok := value.([]libovsdb.UUID); ok {
				var aclRulesSequence []int
				if len(aclRules) != 0 {
					for _, aclRuleUUID := range aclRules {
						tableACLRule, err := vtepdb.ACLRuleGetByUUID(aclRuleUUID.GoUUID)
						if err != nil {
							log.Warning("ACLRule %s not exist", aclRuleUUID.GoUUID)
							continue
						}
						aclRulesSequence = append(aclRulesSequence, tableACLRule.Sequence)
					}
				}
				attrs[ACLAttrRules] = aclRulesSequence
			}
		}
	}

	return attrs
}
