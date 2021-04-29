package driver

import (
	"fmt"

	cdb "github.com/cn-pmlabs/govtep/lib/odbapi/unosconfig"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"

	"github.com/ebay/libovsdb"
)

type aclRuleAPI struct {
	moduleID int
}

var aclRuleAPIs = aclRuleAPI{
	moduleID: tai.ObjectIDACLRule,
}

func (v aclRuleAPI) CreateObject(obj interface{}) error {
	objACLRule := obj.(tai.ACLRuleObj)

	aclIndex := cdb.ACLIndex{
		ACLName: objACLRule.ACLName,
	}
	_, err := cdb.ACLGetByIndex(aclIndex)
	if err != nil {
		log.Warning("[Driver] ACL %s not exist", aclIndex.ACLName)
		return nil
	}

	aclRuleIndex := cdb.ACLRuleIndex{
		ACLName:  objACLRule.ACLName,
		Sequence: objACLRule.Sequence,
	}
	_, err = cdb.ACLRuleGetByIndex(aclRuleIndex)
	if err == nil {
		log.Info("[Driver] ACLRule %d already exist", objACLRule.Sequence)
		return nil
	}

	aclRuleCfg := cdb.TableACLRule{
		ACLName:  objACLRule.ACLName,
		Sequence: objACLRule.Sequence,
	}

	err = cdb.ACLUpdateAddRuleName(aclIndex, aclRuleCfg)
	if err != nil {
		return err
	}
	return nil
}

func (v aclRuleAPI) RemoveObject(obj interface{}) error {
	objACLRule := obj.(tai.ACLRuleObj)

	aclIndex := cdb.ACLIndex{
		ACLName: objACLRule.ACLName,
	}
	_, err := cdb.ACLGetByIndex(aclIndex)
	if err != nil {
		return fmt.Errorf("[Driver] ACL %s not exist", aclIndex.ACLName)
	}

	aclRuleIndex := cdb.ACLRuleIndex{
		ACLName:  objACLRule.ACLName,
		Sequence: objACLRule.Sequence,
	}
	tableACLRule, err := cdb.ACLRuleGetByIndex(aclRuleIndex)
	if err != nil {
		return fmt.Errorf("[Driver] ACLRule %d not exist", objACLRule.Sequence)
	}
	ruleUpdate := []libovsdb.UUID{{GoUUID: tableACLRule.UUID}}

	err = cdb.ACLUpdateRuleNameDelvalue(aclIndex, ruleUpdate)
	if err != nil {
		return err
	}

	return nil
}

func (v aclRuleAPI) AddObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v aclRuleAPI) DelObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v aclRuleAPI) SetObjectAttr(interface{}, map[interface{}]interface{}) error {
	return nil
}

func (v aclRuleAPI) GetObjectAttr(interface{}, []interface{}) (map[interface{}]interface{}, error) {
	return nil, nil
}

func (v aclRuleAPI) ListObject() ([]interface{}, error) {
	return nil, nil
}
