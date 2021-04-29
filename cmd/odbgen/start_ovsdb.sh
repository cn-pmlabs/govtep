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

export OVSDB_DBDIR=/usr/local/etc/openvswitch
export OVSDB_RUNDIR=/usr/local/var/run/openvswitch
export OVSDB_LOGDIR=/usr/local/var/log/openvswitch

if [ ! -f "OVSDB_DBDIR" ]; then
    mkdir -p $OVSDB_DBDIR
fi
if [ ! -f "OVSDB_RUNDIR" ]; then
    mkdir -p $OVSDB_RUNDIR
fi
if [ ! -f "OVSDB_LOGDIR" ]; then
    mkdir -p $OVSDB_LOGDIR
fi

if [ ! -f "$OVSDB_DBDIR/ovsdb.db" ];then
    ovsdb-tool create $OVSDB_DBDIR/ovsdb.db ovsdb.ovsschema
elif [ $INIT == "Y" ]; then
    rm $OVSDB_DBDIR/ovsdb.db
    ovsdb-tool create $OVSDB_DBDIR/ovsdb.db ovsdb.ovsschema
fi

ovsdb-server \
    -vconsole:emer \
    -vsyslog:err \
    -vfile:info \
    --monitor \
    --detach \
    --pidfile=$OVSDB_RUNDIR/ovsdb.pid \
    --log-file=$OVSDB_LOGDIR/ovsdb.log \
    --remote punix:$OVSDB_RUNDIR/ovsdb.sock \
    --remote ptcp:6688:0.0.0.0 \
    $OVSDB_DBDIR/ovsdb.db
