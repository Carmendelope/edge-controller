# edge-controller
Egde controller for the Service Net

# Introduction

The Edge Controller is a component that will be deployed on the user premises and it is in charge of managing a collection of
agents. In terms of how that component will be deployed, we envision a VM deployment that could potentially be transformed
into an appliance.

# Sync and Async operations

The EIC uses both sync and async communication with the management cluster. The following table describes
the different types of operations and how those are performed

 | Operation  | Type | Local persistence | Description |
 | ------------- | ------------- |------------- |------------- |
 | Agent Join  | SYNC | No | An agent wants to join the EIC |
 | Agent Start | ASYNC | Yes | An agent starts |
 | Agent Callback | ASYNC | Yes | An agent sends a callback from an operation |


# How to run the edge-controller
 The run command is in the file `/Users/cdelope/go/src/github.com/nalej/edge-controller/init/edge-controller.vagrant.service`
  
 ` ...
 
 ExecStart=/vagrant/bin/linux_amd64/edge-controller run --joinTokenPath=_joinTokenPath_ --useBBoltProviders --bboltpath=_databasePath_ --name=_ec_name_ --labels=_labels_ --geolocation=_location_
 
  ...
`

- __joinTokenPath:__
 
we need a token to join an edge-controller. We can obtain it executing public-api

Once the login is done:

`./bin/public-api-cli login --email=_user_ --password=*****`

we can execute the command to obtain it:

`./bin/public-api-cli edgecontroller create-join-token --outputPath=_token_file_path`

we already have the token, we can do the join of the edge-controller

- __bbolpath:__ path where the database file is going to saved. 
- __name:__ is required
- __labels and geolocation:__ no required


once configured the parameters indicated above we can run the VM executing ` make vagrant`

**The edge-controller is started!!**

### Some commands that can help...

`vagrant ssh`: command to entry to VM

`vi /etc/edge-controller/credentials.json` : file with credentials info

`sudo journalctl -u edge-controller.service -f`: command to see the edge-controller logs

### Set debug on the vagrant environment

```
vagrant@ubuntu-bionic:~$ sudo systemctl stop edge-controller
vagrant@ubuntu-bionic:~$ sudo vim /etc/systemd/system/edge-controller.service
vagrant@ubuntu-bionic:~$ sudo systemctl daemon-reload
vagrant@ubuntu-bionic:~$ sudo systemctl start edge-controller
```