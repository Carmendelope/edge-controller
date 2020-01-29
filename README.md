
# Edge-Controller
Edge Controller for the Service Network. The Edge Controller is a component that will be deployed on the user premises and 
it's in charge of managing a collection of agents. 

The EC connects to the management cluster through a VPN, receives _Alive Messages_ from agents and can determine their IP. It can receive orders from the management cluster and send them to agents, and works with a plugin system.

## Getting Started
The EC runs in a virtual machine. The component includes an installation of a VM with `vagrant`. To run it, you need
to follow the steps bellow:

1) Generate the join token. To join an EC, a `join token` is required. This token is generated in `authx` and we can ask for it with a public-api command:
```
./bin/public-api-cli edgecontroller create-join-token --outputPath=_token_file_path
```
2) Configure the EC updating the file `configs/config.yaml`.
```
joinTokenPath: /tmp/joinToken.json
useBBoltProviders: true
bboltpath: /home/vagrant/database.db
name: <EC_name>
labels: "name:test"
geolocation: "Madrid, Madrid, Spain" 
```
3) Run the VM executing `make vagrant`.

_The edge-controller is started!!_

### Some commands that can help...

To enter the VM:
`vagrant ssh`

To see/edit the credentials information:
`vi /etc/edge-controller/credentials.json`

To see the edge controller logs:
`sudo journalctl -u edge-controller.service -f`

**Set debug on the vagrant environment**

```
vagrant@ubuntu-bionic:~$ sudo systemctl stop edge-controller
vagrant@ubuntu-bionic:~$ sudo vim /etc/systemd/system/edge-controller.service
vagrant@ubuntu-bionic:~$ sudo systemctl daemon-reload
vagrant@ubuntu-bionic:~$ sudo systemctl start edge-controller
```


### Build and compile

In order to build and compile this repository use the provided Makefile:

```
make all
```

This operation generates the binaries for this repo, downloads the required dependencies, runs existing tests and generates ready-to-deploy Kubernetes files.

### Run tests

Tests are executed using Ginkgo. To run all the available tests:

```
make test
```

### Update dependencies

Dependencies are managed using Godep. For an automatic dependencies download use:

```
make dep
```

In order to have all dependencies up-to-date run:

```
dep ensure -update -v
```

### Installing the VM

As commented above, you can install a VM executing 

```
make vagrant
```

### Managing the VM

There are more commands to manage the virtual machine:

To stop it:
```
make vagrant-stop
```
To destroy it:
```
make vagrant-destroy
```
To start it:
```
make vagrant-up
```
To restart it:
```
make vagrant-restart-service
```
And to rebuild it:
```
vagrant-rebuild
```

## Contributing

Please read [contributing.md](contributing.md) for details on our code of conduct, and the process for submitting pull requests to us.


## Versioning

We use [SemVer](http://semver.org/) for versioning. For the available versions, see the [tags on this repository](https://github.com/nalej/edge-controller/tags). 

## Authors

See also the list of [contributors](https://github.com/nalej/edge-controller/contributors) who participated in this project.

## License
This project is licensed under the Apache 2.0 License - see the [LICENSE-2.0.txt](LICENSE-2.0.txt) file for details.


