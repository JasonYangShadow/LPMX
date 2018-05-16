#!/bin/bash

ROOT=/tmp/x86_64/terminal
BINARY="$(dirname $(dirname $(dirname `pwd`)))"
CURRENT=`pwd`


echo "cleanup"
rm -rf $ROOT/.lpmx
rm -rf $BINARY/.lpmxsys

echo "Automatically create exmaple folder under /tmp with $ROOT"
mkdir -p $ROOT
mkdir -p $ROOT/bin
mkdir -p $ROOT/lib
cp -n pid $ROOT/bin
cp -n getpid.so $ROOT/lib
echo "checking if there is memcached instance running on your os..."
MEM_PID=`ps -aux|grep memcached|grep -v "grep"|awk '{print $2}'`
if [ -n "$MEM_PID" ];then
  echo "memcached instance with pid $MEM_PID will be killed"
  kill -9 $MEM_PID
fi
echo "restarting memcached server"
export LD_PRELOAD=$BINARY/libevent.so
cd $BINARY
./memcached -d
NEW_MEM_PID=`ps -aux|grep memcached|grep -v "grep"|awk '{print $2}'`
if [ -n "$NEW_MEM_PID" ];then
  echo "memcached instance is restarted with new pid $NEW_MEM_PID"
else
  echo "restarting memcached instace encountered error"
  exit 1
fi
if [ -f "readme" ];then
  cat readme
fi
./lpmx init
./lpmx run -c $CURRENT/setting.yml -s $ROOT
