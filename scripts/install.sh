#!/bin/bash

# Dependencies installation
apt-get update
apt-get install -y build-essential

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
cp /vagrant/init/vpnclient.service /etc/systemd/system/vpnclient.service
chmod 644 /etc/systemd/system/vpnclient.service
systemctl enable /etc/systemd/system/vpnclient.service
systemctl daemon-reload
systemctl stop vpnclient.service
systemctl start vpnclient.service

# edge-controller service
cp /vagrant/init/edge-controller.vagrant.service /etc/systemd/system/edge-controller.service
chmod 644 /etc/systemd/system/edge-controller.service
systemctl enable /etc/systemd/system/edge-controller.service
systemctl daemon-reload
systemctl stop edge-controller.service
systemctl start edge-controller.service
