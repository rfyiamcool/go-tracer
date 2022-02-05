#!/usr/bin/env bash

min_num=1
max_num=100000000000

# get random number
function rand() {
    min=$1
    max=$(($2 - $min + 1))
    num=$(cat /dev/urandom | head -n 10 | cksum | awk -F ' ' '{print $1}')
    echo $(($num % $max + $min))
}

# convert 10 -> 16
function dec2hex() {
    printf "%x" $1
}

# make x-trace-id
function make_xid() {
    # trace high
    rnd=$(rand $min_num $max_num)
    traceHigh=$(dec2hex $rnd)

    # trace low
    rnd=$(rand $min_num $max_num)
    traceLow=$(dec2hex $rnd)

    # traceID = high + low
    traceID=$traceHigh$traceLow

    # x-trace-id: $traceID:$spanID:$parentSpanID:flags
    echo $traceID:$traceID:0000000000000000:1
}

function main() {
    xid=`make_xid`
    echo "x-trace-id ===> " $xid

    curl -vvv -H "x-trace-id: $xid" 127.0.0.1:8080/ping
}

main
