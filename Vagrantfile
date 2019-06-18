# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.define "edge_controller", primary: true do |edge_controller|
    edge_controller.vm.box = "ubuntu/bionic64"
    edge_controller.vm.network "forwarded_port", guest: 5588, host: 5588
    edge_controller.vm.network "forwarded_port", guest: 5577, host: 5577
    edge_controller.vm.provision "file", source: "../service-net-agent/bin/linux_amd64", destination: "/tmp/agents/linux_amd64"
    edge_controller.vm.provision "file", source: "../service-net-agent/bin/windows_amd64", destination: "/tmp/agents/windows_amd64"
    edge_controller.vm.provision "file", source: "../service-net-agent/bin/darwin_amd64", destination: "/tmp/agents/darwin_amd64"
    edge_controller.vm.provision "shell", path: "scripts/install.sh"
    edge_controller.vm.provision "shell", inline: "apt-get install -y dkms virtualbox-guest-dkms virtualbox-guest-utils" 
    edge_controller.vm.network "public_network"
  end
end
