#!/bin/bash

set -evx

# Variables
RANCHER_LOG_COLLECTER="https://raw.githubusercontent.com/rancherlabs/support-tools/master/collection/rancher/v2.x/logs-collector/rancher2_logs_collector.sh"

# Create directory to store logs
mkdir -p -m 755 logs
cd logs

# Download and run script
curl -L ${RANCHER_LOG_COLLECTER} -o rancherlogcollector.sh
chmod +x rancherlogcollector.sh
sudo ./rancherlogcollector.sh -d ../logs

# Done!
exit 0
