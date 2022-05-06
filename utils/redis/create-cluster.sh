#!/bin/bash

[[ $VERBOSE == 1 ]] && set -x
[[ $IGNERR == 1 ]] || set -e

CLUSTER=${CLUSTER:-0}
CLUSTER_HOST=${CLUSTER_HOST:-"127.0.0.1"}
REDIS_SERVER_BIN=${REDIS_SERVER_BIN:-"redis-server"}
REDIS_CLI_BIN=${REDIS_CLI_BIN:-"redis-cli"}
NODES=${NODES:-3}
# use the HOSTS env var to create a multi-VM cluster in the format of ip:port pair.
# HOSTS should contain the remote ip:port pairs
# example HOSTS="host2-ip:port1" ./create-cluster.sh create
HOSTS=${HOSTS:-""}
PORT=${PORT:-30000}
REPLICAS=${REPLICAS:-0}
PROTECTED_MODE=${PROTECTED_MODE:-"no"}
ADDITIONAL_OPTIONS=${ADDITIONAL_OPTIONS:-""}
TIMEOUT=2000
AOF=${AOF:-0}
CORE_START=${CORE_START:-0}

# Computed vars
ENDPORT=$((PORT+NODES))

if [ "$1" == "start" ]
then
    while [ $((PORT < ENDPORT)) != "0" ]; do
        PORT=$((PORT+1))
        echo "Starting redis-server on core $CORE port $PORT with extra args: \"$ADDITIONAL_OPTIONS\""
        AOF_PROPS="--appendonly no"
        RDB_PROPS="--dbfilename dump-${PORT}.rdb"
        [[ $AOF == 1 ]] && AOF_PROPS="--appendonly yes --appendfilename appendonly-${PORT}.aof"
        taskset -c $CORE_START $REDIS_SERVER_BIN --port $PORT  --protected-mode $PROTECTED_MODE --cluster-enabled yes --cluster-config-file nodes-${PORT}.conf --cluster-node-timeout $TIMEOUT ${RDB_PROPS} ${AOF_PROPS} --logfile ${PORT}.log --daemonize yes ${ADDITIONAL_OPTIONS}
        CORE_START=$((CORE_START+1))
    done
    exit 0
fi

if [ "$1" == "create" ]
then
    while [ $((PORT < ENDPORT)) != "0" ]; do
        PORT=$((PORT+1))
        HOSTS="$HOSTS $CLUSTER_HOST:$PORT"
    done
    OPT_ARG="--cluster-yes"
    $REDIS_CLI_BIN --cluster create $HOSTS --cluster-replicas $REPLICAS $OPT_ARG
    exit 0
fi

if [ "$1" == "hosts" ]
then
    while [ $((PORT < ENDPORT)) != "0" ]; do
        PORT=$((PORT+1))
        HOSTS="$HOSTS $CLUSTER_HOST:$PORT"
    done
    echo "HOSTS=\"$HOSTS\""
    exit 0
fi

if [ "$1" == "stop" ]
then
    while [ $((PORT < ENDPORT)) != "0" ]; do
        PORT=$((PORT+1))
        echo "Stopping $PORT"
        $REDIS_CLI_BIN -p $PORT shutdown nosave
    done
    exit 0
fi

if [ "$1" == "watch" ]
then
    PORT=$((PORT+1))
    while [ 1 ]; do
        clear
        date
        $REDIS_CLI_BIN -p $PORT cluster nodes | head -30
        sleep 1
    done
    exit 0
fi

if [ "$1" == "tail" ]
then
    INSTANCE=$2
    PORT=$((PORT+INSTANCE))
    tail -f ${PORT}.log
    exit 0
fi

if [ "$1" == "tailall" ]
then
    tail -f *.log
    exit 0
fi

if [ "$1" == "call" ]
then
    while [ $((PORT < ENDPORT)) != "0" ]; do
        PORT=$((PORT+1))
        $REDIS_CLI_BIN -p $PORT $2 $3 $4 $5 $6 $7 $8 $9
    done
    exit 0
fi

if [ "$1" == "clean" ]
then
    rm -rf *.log
    rm -rf appendonly*.aof
    rm -rf dump*.rdb
    rm -rf nodes*.conf
    exit 0
fi

if [ "$1" == "clean-logs" ]
then
    rm -rf *.log
    exit 0
fi

echo "Usage: $0 [start|create|stop|watch|tail|clean|call]"
echo "start       -- Launch Redis Cluster instances."
echo "create [-f] -- Create a cluster using redis-cli --cluster create."
echo "                  Note: to create a multi-node cluster pass the remote host:port pairs via the HOSTS env var."
echo "                  Example: HOSTS=\"remote-ip:port1 remote-ip:port2\" ./create-cluster.sh create"
echo "stop        -- Stop Redis Cluster instances."
echo "hosts       -- Print the HOSTS env var."
echo "watch       -- Show CLUSTER NODES output (first 30 lines) of first node."
echo "tail <id>   -- Run tail -f of instance at base port + ID."
echo "tailall     -- Run tail -f for all the log files at once."
echo "clean       -- Remove all instances data, logs, configs."
echo "clean-logs  -- Remove just instances logs."
echo "call <cmd>  -- Call a command (up to 7 arguments) on all nodes."
