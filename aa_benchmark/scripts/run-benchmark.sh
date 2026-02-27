#!/bin/bash
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 MBition GmbH

IP_SERVICE="REPLACE"
IP_CLIENT="REPLACE"
SSH_PATH="REPLACE"
PEM_FILE="REPLACE"
USERNAME="REPLACE"
GROUP="REPLACE"

for ((SC=50; SC<=3200; SC*=2)); do
	sed -i "114s/.*/int l_j = $SC;/" /home/developer/vrte/project/AraCM_Method_Benchmark/subscriber/subscriber.cpp

	for RUN in $(seq 1 10); do

	# Compile service
	rvbuild_service -o Linux -w aarch64 -sd AraCM_Method_Benchmark Graviton
	sleep 5
	/usr/bin/ssh -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_SERVICE "cd /opt/vrte/usr/bin; sudo nohup ./exmd.sh 1>/dev/null 2>&1 &"

	# Compile the client
	rvbuild_client -o Linux -w aarch64 -ld AraCM_Method_Benchmark Graviton
	sleep 5
	/usr/bin/ssh -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_CLIENT "cd /opt/vrte/usr/bin; sudo nohup ./exmd.sh 1>/dev/null 2>&1 &"

	for ((i=1;i<=100*30;i++)); do
		if ssh -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_CLIENT "test -e /opt/vrte/usr/bin/wu_done"; then
			/usr/bin/ssh -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_SERVICE "cd /opt/vrte/usr/bin; nohup ./collect-stats.sh 1>/dev/null 2>&1 &"	
			/usr/bin/ssh -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_CLIENT "cd /opt/vrte/usr/bin; nohup ./collect-stats.sh 1>/dev/null 2>&1 &"	
		
			echo "warm up done..."
			
			break
		fi
		sleep 1
	done


	for ((i=1;i<=100*30;i++)); do
		if ssh -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_CLIENT "test -e /opt/vrte/usr/bin/done"; then
			break
		fi
		sleep 1
	done

	mkdir -p "$SC/$RUN/service"
	mkdir -p "$SC/$RUN/client"

	/usr/bin/ssh -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_CLIENT "sudo chown $USERNAME:$GROUP /opt/vrte/usr/bin/times.stats"	
	scp -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_CLIENT:/opt/vrte/usr/bin/times.stats "$SC/$RUN/client"
	scp -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_CLIENT:/opt/vrte/usr/bin/stats.stats "$SC/$RUN/client"

	scp -i $SSH_PATH/$PEM_FILE $USERNAME@$IP_SERVICE:/opt/vrte/usr/bin/stats.stats "$SC/$RUN/service"

	echo "done"

	done

done
