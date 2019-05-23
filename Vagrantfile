# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.define "edge_controller", primary: true do |edge_controller|
    edge_controller.vm.box = "ubuntu/bionic64"
    edge_controller.vm.network "forwarded_port", guest: 5588, host: 5588
    edge_controller.vm.network "forwarded_port", guest: 5577, host: 5577
    edge_controller.vm.provision "shell", path: "scripts/install.sh"
    edge_controller.vm.network "public_network"
  end
end
