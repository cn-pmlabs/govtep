package govtep

import (
	"github.com/cn-pmlabs/govtep/lib/log"

	"github.com/ebay/libovsdb"
)

type ovnNbNotifier struct {
	onbi *ovsdbc
}

func (notify ovnNbNotifier) Update(context interface{}, updates libovsdb.TableUpdates) {
	notify.onbi.ovnNbNotifyUpdate(updates)
}

func (notify ovnNbNotifier) Locked([]interface{}) {
}

func (notify ovnNbNotifier) Stolen([]interface{}) {
}

func (notify ovnNbNotifier) Echo([]interface{}) {
}

func (notify ovnNbNotifier) Disconnected(client *libovsdb.OvsdbClient) {
	if notify.onbi.Reconn {
		log.Warning("ovsdb %s[%s] disconnected, try reconnect\n", notify.onbi.Db, notify.onbi.Addr)
		go notify.onbi.ovnNbReConnect()
	}
}

type ovnSbNotifier struct {
	osbi *ovsdbc
}

func (notify ovnSbNotifier) Update(context interface{}, updates libovsdb.TableUpdates) {
	notify.osbi.ovnSbNotifyUpdate(updates)
}

func (notify ovnSbNotifier) Locked([]interface{}) {
}

func (notify ovnSbNotifier) Stolen([]interface{}) {
}

func (notify ovnSbNotifier) Echo([]interface{}) {
}

func (notify ovnSbNotifier) Disconnected(client *libovsdb.OvsdbClient) {
	if notify.osbi.Reconn {
		log.Warning("ovsdb %s[%s] disconnected, try reconnect\n", notify.osbi.Db, notify.osbi.Addr)
		go notify.osbi.ovnSbReConnect()
	}
}

type vtepDbNotifier struct {
	vdbi *ovsdbc
}

func (notify vtepDbNotifier) Update(context interface{}, updates libovsdb.TableUpdates) {
	notify.vdbi.vtepDbNotifyUpdate(updates)
}

func (notify vtepDbNotifier) Locked([]interface{}) {
}

func (notify vtepDbNotifier) Stolen([]interface{}) {
}

func (notify vtepDbNotifier) Echo([]interface{}) {
}

func (notify vtepDbNotifier) Disconnected(client *libovsdb.OvsdbClient) {
	if notify.vdbi.Reconn {
		log.Warning("ovsdb %s[%s] disconnected, try reconnect\n", notify.vdbi.Db, notify.vdbi.Addr)
		go notify.vdbi.vtepDBReConnect()
	}
}

type ovnNbLibNotifier struct {
	nbLibClient *libovsdb.OvsdbClient
}

func (notify ovnNbLibNotifier) Update(context interface{}, updates libovsdb.TableUpdates) {
}

func (notify ovnNbLibNotifier) Locked([]interface{}) {
}

func (notify ovnNbLibNotifier) Stolen([]interface{}) {
}

func (notify ovnNbLibNotifier) Echo([]interface{}) {
}

func (notify ovnNbLibNotifier) Disconnected(client *libovsdb.OvsdbClient) {
	log.Warning("ovnnb lib disconnected, try reconnect\n")
	go OvnNbLibReConnect()
}

type ovnSbLibNotifier struct {
	sbLibClient *libovsdb.OvsdbClient
}

func (notify ovnSbLibNotifier) Update(context interface{}, updates libovsdb.TableUpdates) {
}

func (notify ovnSbLibNotifier) Locked([]interface{}) {
}

func (notify ovnSbLibNotifier) Stolen([]interface{}) {
}

func (notify ovnSbLibNotifier) Echo([]interface{}) {
}

func (notify ovnSbLibNotifier) Disconnected(client *libovsdb.OvsdbClient) {
	log.Warning("ovnsb lib disconnected, try reconnect\n")
	go OvnSbLibReConnect()
}
