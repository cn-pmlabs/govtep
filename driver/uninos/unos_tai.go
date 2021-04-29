package driver

import (
	"fmt"

	"github.com/cn-pmlabs/govtep/lib/log"
	"github.com/cn-pmlabs/govtep/tai"
)

func (d *unosDriver) TaiCreateObject(objID tai.ObjID, obj interface{}) error {
	log.Info("[Driver] TaiCreateObject %v => %+v\n", tai.ObjectOrder[objID], obj)

	if unosDriverHandler.ModuleAPIs[objID] == nil {
		return fmt.Errorf("[Driver] unspported object %v", tai.ObjectOrder[objID])
	}

	return unosDriverHandler.ModuleAPIs[objID].CreateObject(obj)
}

func (d *unosDriver) TaiRemoveObject(objID tai.ObjID, obj interface{}) error {
	if unosDriverHandler.ModuleAPIs[objID] == nil {
		return fmt.Errorf("[Driver] unspported object %v", tai.ObjectOrder[objID])
	}
	return unosDriverHandler.ModuleAPIs[objID].RemoveObject(obj)
}

func (d *unosDriver) TaiAddObjectAttr(objID tai.ObjID, obj interface{},
	attr map[interface{}]interface{}) error {
	if unosDriverHandler.ModuleAPIs[objID] == nil {
		return fmt.Errorf("[Driver] unspported object %v", tai.ObjectOrder[objID])
	}
	return unosDriverHandler.ModuleAPIs[objID].AddObjectAttr(obj, attr)
}

func (d *unosDriver) TaiDelObjectAttr(objID tai.ObjID, obj interface{},
	attr map[interface{}]interface{}) error {
	if unosDriverHandler.ModuleAPIs[objID] == nil {
		return fmt.Errorf("[Driver] unspported object %v", tai.ObjectOrder[objID])
	}
	return unosDriverHandler.ModuleAPIs[objID].DelObjectAttr(obj, attr)
}

func (d *unosDriver) TaiSetObjectAttr(objID tai.ObjID, obj interface{},
	attr map[interface{}]interface{}) error {
	if unosDriverHandler.ModuleAPIs[objID] == nil {
		return fmt.Errorf("[Driver] unspported object %v", tai.ObjectOrder[objID])
	}
	return unosDriverHandler.ModuleAPIs[objID].SetObjectAttr(obj, attr)
}

func (d *unosDriver) TaiGetObjectAttr(objID tai.ObjID, obj interface{},
	attr []interface{}) (map[interface{}]interface{}, error) {
	if unosDriverHandler.ModuleAPIs[objID] == nil {
		return nil, fmt.Errorf("[Driver] unspported object %v", tai.ObjectOrder[objID])
	}
	return unosDriverHandler.ModuleAPIs[objID].GetObjectAttr(obj, attr)
}

func (d *unosDriver) TaiListObject(objID tai.ObjID) ([]interface{}, error) {
	if unosDriverHandler.ModuleAPIs[objID] == nil {
		return nil, fmt.Errorf("[Driver] unspported object %v", tai.ObjectOrder[objID])
	}
	return unosDriverHandler.ModuleAPIs[objID].ListObject()
}
