package helper

/*
network:
    version: 2
    ethernets:
        enp0s3:
            dhcp4: true
            match:
                macaddress: 02:35:4a:03:dd:91
            set-name: enp0s3
            nameservers:
                addresses: [51.144.230.81]
 */

type NameServers struct {
	Addresses []string  `json:"addresses,omitempty"`
}

type Enp0s3 struct {
	Nameservers NameServers `json:"nameservers,omitempty"`
}

type Ethernets struct {
	Enp0s3 Enp0s3 `json:"enp0s3,omitempty"`
}

type Network struct {
	//Version int `json:"version,omitempty"`
	Ethernets Ethernets  `json:"ethernets,omitempty"`

}

type NetPlan struct{
	Network Network `json:"network,omitempty"`
}



func GetInitNetPlan() * NetPlan{

	return &NetPlan{
		Network: Network{
			Ethernets: Ethernets{
				Enp0s3: Enp0s3{
					Nameservers: NameServers{
						Addresses: []string{},
					},
				},
			},
		},
	}

}