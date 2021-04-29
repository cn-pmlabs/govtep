#!/bin/bash
set -o errexit

INIT='N'

print_usage_and_exit()
{
	echo $0 "-i/--init"
	exit 1
}

if [ $# == 1 ] && [ $1 == "-i" ] ; then
    INIT='Y'
fi

export VTEPDB_DBDIR=/usr/local/etc/openvswitch
export VTEPDB_RUNDIR=/usr/local/var/run/openvswitch
export VTEPDB_LOGDIR=/usr/local/var/log/openvswitch

if [ ! -f "VTEPDB_DBDIR" ]; then
    mkdir -p $VTEPDB_DBDIR
fi
if [ ! -f "VTEPDB_RUNDIR" ]; then
    mkdir -p $VTEPDB_RUNDIR
fi
if [ ! -f "VTEPDB_LOGDIR" ]; then
    mkdir -p $VTEPDB_LOGDIR
fi

if [ ! -f "$VTEPDB_DBDIR/controller_vtep.db" ];then
    ovsdb-tool create $VTEPDB_DBDIR/controller_vtep.db \
        /root/code/unos-schema/controller/controller_vtep.ovsschema
elif [ $INIT == "Y" ]; then
    rm $VTEPDB_DBDIR/controller_vtep.db
    ovsdb-tool create $VTEPDB_DBDIR/controller_vtep.db \
        /root/code/unos-schema/controller/controller_vtep.ovsschema
fi


ovsdb-server \
    -vconsole:emer \
    -vsyslog:err \
    -vfile:info \
    --monitor \
    --detach \
    --pidfile=$VTEPDB_RUNDIR/vtep_db.pid \
    --log-file=$VTEPDB_LOGDIR/vtep_db.log \
    --remote punix:$VTEPDB_RUNDIR/vtep_db.sock \
    --remote ptcp:6644:0.0.0.0 \
    --remote=db:CONTROLLER_VTEP,Global,managers \
    $VTEPDB_DBDIR/controller_vtep.db
