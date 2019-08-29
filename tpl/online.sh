#!/bin/bash
#
#使用gunicorn启动, 要求系统安装了gunicorn模块
#如果要自定义进程名，还需要安装setproctitle模块
#pip install gunicorn setproctitle
#

dir=$(
    cd $(dirname $0)
    pwd
)
cd $dir

if [ -f online_preboot.sh ]; then
    source online_preboot.sh
fi

host=$(rtfd cfg api:host)
port=$(rtfd cfg api:port)
basedir=$(rtfd cfg g:base_dir)
procname=rtfd
cpu_count=$(cat /proc/cpuinfo | grep "processor" | wc -l)
[ -d ${basedir}/logs ] || mkdir -p ${basedir}/logs
logfile=${basedir}/logs/gunicorn.log
pidfile=${basedir}/logs/rtfd.pid

case $1 in
start)
    if [ -f $pidfile ]; then
        echo "Has pid($(cat $pidfile)) in $pidfile, please check, exit."
        exit 1
    else
        gunicorn -w $cpu_count -b ${host}:${port} app:app --daemon --pid $pidfile --log-file $logfile -n $procname --max-requests 250
        sleep 1
        pid=$(cat $pidfile)
        [ "$?" != "0" ] && exit 1
        echo "$procname start over with pid ${pid}"
    fi
    ;;

run)
    gunicorn -w $cpu_count -b ${host}:${port} app:app --max-requests 250 --name $procname
    ;;

stop)
    if [ ! -f $pidfile ]; then
        echo "$pidfile does not exist, process is not running"
    else
        echo "Stopping ${procname}..."
        pid=$(cat $pidfile)
        kill $pid
        while [ -x /proc/${pid} ]; do
            echo "Waiting to shutdown ..."
            kill $pid
            sleep 1
        done
        echo "${procname} stopped"
        rm -f $pidfile
    fi
    ;;

status)
    if [ -f $pidfile ]; then
        ps aux | grep -v grep | grep -E "$(cat $pidfile)|${procname}" --color=auto
    fi
    ;;

reload)
    if [ -f $pidfile ]; then
        kill -HUP $(cat $pidfile)
    fi
    ;;

restart)
    bash $(basename $0) stop
    bash $(basename $0) start
    ;;

*)
    echo "Usage: $0 start|run|stop|reload|restart|status"
    ;;
esac
