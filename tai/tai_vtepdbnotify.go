package tai

import (
	"github.com/ebay/libovsdb"
)

type vtepdbNotifier struct {
	tdbi *ovsdbc
}

func (notify vtepdbNotifier) Update(context interface{}, updates libovsdb.TableUpdates) {
	notify.tdbi.taiNotifyUpdate(updates)
}

func (notify vtepdbNotifier) Locked([]interface{}) {
}
func (notify vtepdbNotifier) Stolen([]interface{}) {
}
func (notify vtepdbNotifier) Echo([]interface{}) {
}
func (notify vtepdbNotifier) Disconnected(client *libovsdb.OvsdbClient) {
	notify.tdbi.reConnect()
}
