package helper

type Match struct {
	Macaddress string `json:"macaddress,omitempty"`
}

type NameServers struct {
	Addresses []string  `json:"addresses,omitempty"`
}

type Enp0s3 struct {
	Dhcp4 bool `json:"dhcp4,omitempty"`
	Match Match `json:"match,omitempty"`
	SetName string `json:"set-name,omitempty"`
	Nameservers NameServers `json:"nameservers,omitempty"`
}

type Ethernets struct {
	Enp0s3 Enp0s3 `json:"enp0s3,omitempty"`
}

type Network struct {
	Version int `json:"version,omitempty"`
	Ethernets Ethernets  `json:"ethernets,omitempty"`

}

type Neplan struct{
	Network Network `json:"network,omitempty"`
}
