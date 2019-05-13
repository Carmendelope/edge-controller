# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.define "edge_controller", primary: true do |edge_controller|
    edge_controller.vm.box = "ubuntu/bionic64"
    edge_controller.vm.network "forwarded_port", guest: 5555, host: 5555
    edge_controller.vm.network "forwarded_port", guest: 5556, host: 5556
    edge_controller.vm.provision "shell", path: "scripts/install.sh"
  end
end
