package driver

import (
	"fmt"

	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"

	"github.com/ebay/libovsdb"
)

type aclAPI struct {
	moduleID int
}

var aclAPIs = aclAPI{
	moduleID: tai.ObjectIDACL,
}

func (v aclAPI) CreateObject(obj interface{}) error {
	objACL := obj.(tai.ACLObj)

	aclIndex := cdb.ACLIndex{
		ACLName: objACL.ACLName,
	}
	_, err := cdb.ACLGetByIndex(aclIndex)
	if err == nil {
		log.Info("[Driver] ACL %s already exist", objACL.ACLName)
		return nil
	}

	aclCfg := cdb.TableACL{
		ACLName: objACL.ACLName,
		Type:    cdb.ACLTypeL2,
		Stage:   cdb.ACLStageIngress,
	}

	_, err = cdb.ACLAdd(aclCfg)
	if err != nil {
		return err
	}
	return nil
}

func (v aclAPI) RemoveObject(obj interface{}) error {
	objACL := obj.(tai.ACLObj)

	aclIndex := cdb.ACLIndex{
		ACLName: objACL.ACLName,
	}
	err := cdb.ACLDelByIndex(aclIndex)
	if err != nil {
		return err
	}

	return nil
}

func (v aclAPI) AddObjectAttr(obj interface{}, attrs map[interface{}]interface{}) error {
	objACL := obj.(tai.ACLObj)

	aclIndex := cdb.ACLIndex{
		ACLName: objACL.ACLName,
	}
	_, err := cdb.ACLGetByIndex(aclIndex)
	if err != nil {
		return fmt.Errorf("[Driver] ACL %s not exist", objACL.ACLName)
	}

	for attr, attrValue := range attrs {
		switch attr {
		case tai.ACLAttrStage:
			cdb.ACLSetField(aclIndex, cdb.ACLFieldStage, attrValue)
		case tai.ACLAttrType:
			cdb.ACLSetField(aclIndex, cdb.ACLFieldType, attrValue)
		case tai.ACLAttrRules:
			aclRules, ok := attrValue.([]int)
			if !ok {
				log.Warning("[Driver] ACLRule get failed for add ACLAttrRules")
				continue
			}
			for _, aclRuleSeqnum := range aclRules {
				aclRuleIndex := cdb.ACLRuleIndex{
					ACLName:  objACL.ACLName,
					Sequence: aclRuleSeqnum,
				}
				tableACLRule, err := cdb.ACLRuleGetByIndex(aclRuleIndex)
				if err != nil {
					log.Warning("[Driver] ACLRule %d not exist", aclRuleSeqnum)
					continue
				}

				ruleUpdate := []libovsdb.UUID{{GoUUID: tableACLRule.UUID}}
				err = cdb.ACLUpdateRuleNameAddvalue(aclIndex, ruleUpdate)
				if err != nil {
					continue
				}
			}
		}
	}

	return nil
}

func (v aclAPI) DelObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v aclAPI) SetObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v aclAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v aclAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
