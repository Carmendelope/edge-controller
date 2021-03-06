#!/bin/bash

mkdir -p /etc/edge-controller/

VAGRANT_INIT_PATH=/vagrant/init
INIT_PATH="${INIT_PATH:-$VAGRANT_INIT_PATH}"
EDGE_CONTROLLER_SERVICE="${EDGE_CONTROLLER_SERVICE:-edge-controller.vagrant.service}"

# Agent binaries
cp -R /tmp/agents /opt
rm -rf /tmp/agents

# Dependencies installation
apt-get update
apt-get install -y build-essential wget

# InfluxDB installation
wget -qO- https://repos.influxdata.com/influxdb.key | apt-key add -
source /etc/lsb-release
echo "deb https://repos.influxdata.com/${DISTRIB_ID,,} ${DISTRIB_CODENAME} stable" | tee /etc/apt/sources.list.d/influxdb.list
apt-get update
apt-get install -y influxdb
systemctl unmask influxdb.service
systemctl enable influxdb.service
systemctl start influxdb.service

# SoftEther VPNClient installation
curl -sLo /tmp/softether-vpnclient-v4.29-9680-rtm-2019.02.28-linux-x64-64bit.tar.gz https://github.com/SoftEtherVPN/SoftEtherVPN_Stable/releases/download/v4.29-9680-rtm/softether-vpnclient-v4.29-9680-rtm-2019.02.28-linux-x64-64bit.tar.gz
cd /tmp
tar zxvf softether-vpnclient-v4.29-9680-rtm-2019.02.28-linux-x64-64bit.tar.gz
cd /tmp/vpnclient
make i_read_and_agree_the_license_agreement
rm -rf /usr/bin/vpnclient
cp -r /tmp/vpnclient /usr/bin/vpnclient
rm -rf /tmp/softether-vpnclient-v4.29-9680-rtm-2019.02.28-linux-x64-64bit.tar.gz
rm -rf /tmp/vpnclient

# SoftEther VPNClient service
cp $INIT_PATH/vpnclient.service /etc/systemd/system/vpnclient.service
chmod 644 /etc/systemd/system/vpnclient.service
systemctl enable /etc/systemd/system/vpnclient.service
systemctl daemon-reload
systemctl stop vpnclient.service
systemctl start vpnclient.service

# edge-controller service
cp $INIT_PATH/$EDGE_CONTROLLER_SERVICE /etc/systemd/system/edge-controller.service
chmod 644 /etc/systemd/system/edge-controller.service
systemctl enable /etc/systemd/system/edge-controller.service
systemctl daemon-reload
systemctl stop edge-controller.service
systemctl start edge-controller.service
