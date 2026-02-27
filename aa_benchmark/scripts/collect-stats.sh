#!/bin/bash
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 MBition GmbH

rm -f stats.stats

while true; do
	CPU=$(awk '{u=$2+$4; t=$2+$4+$5; if (NR==1){u1=u; t1=t;} else print ($2+$4-u1) * 100 / (t-t1) "%"; }' <(grep 'cpu ' /proc/stat) <(sleep 1;grep 'cpu ' /proc/stat))
	MEM=$(awk 'NR==1{T=$2} NR==2{F=$2; print(T-F); exit}' /proc/meminfo)
	
	echo "$CPU//$MEM" >>stats.stats
	
	sleep 0.001
done
