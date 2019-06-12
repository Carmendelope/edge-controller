package helper

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nalej/derrors"
	"github.com/nalej/grpc-eic-api-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	command = "/usr/bin/vpnclient/vpncmd"
	cmdMode = "/Client"
	hub = "/HUB:DEFAULT"
	cmdCmd = "/cmd"
	nicCreateCmd = "NicCreate"
	nicName ="nicname"
	accountCreateCmd = "AccountCreate"
	accountPasswordSetCmd = "AccountPasswordSet"
	vpnClientAddress = "localhost"
	resolvedFile="/etc/systemd/resolved.conf"
	CredentialsFile = "/etc/edge-controller/credentials.json"
	accountDisconnect = "AccountDisconnect"
	accountDelete = "AccountDelete"
)
const DefaultTimeout = time.Minute

const AuthHeader = "Authorization"


type JoinHelper struct {
	// JoinTokenFile path
	JoinTokenFile string
	// EicToken EICJoinToken (organization_id, edge_controller_id, etc)
	EicToken grpc_inventory_manager_go.EICJoinToken
	// JoinPort with the URL the EIC needs to send the message for starting the join operation.
	JoinPort int
}

// NewJoinHelper returns a JoinHelper to manage all the join and credentials actions
func NewJoinHelper (configFile string, port int) (*JoinHelper, error) {

	var eicToken grpc_inventory_manager_go.EICJoinToken

	jsonFile, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("tokenFile", configFile).Msg("Successfully Opened")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &eicToken)
	if err != nil {
		log.Error().Str("err", conversions.ToDerror(err).DebugReport()).Msg("error Unmarshalling joinTokenFile")
		return nil, err
	}

	return &JoinHelper{
		JoinTokenFile: configFile,
		EicToken: eicToken,
		JoinPort: port,
	}, nil
}

func (j * JoinHelper) LoadTokenFile () derrors.Error{

	if j.JoinTokenFile == "" {
		return derrors.NewFailedPreconditionError("no join file found")
	}

	jsonFile, err :=  os.Open(j.JoinTokenFile)
	if err != nil {
		return conversions.ToDerror(err)
	}
	log.Debug().Str("tokenFile", j.JoinTokenFile).Msg("Successfully Opened")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &j.EicToken)
	if err != nil {
		log.Error().Str("err", conversions.ToDerror(err).DebugReport()).Msg("error Unmarshalling joinTokenFile")
		return conversions.ToDerror(err)
	}

	return  nil
}

// getLabels convert labelsStr (param1=value1,...,paramN=valueN) to a map
func getLabels (labelsStr string) (map[string]string, derrors.Error) {

	labels := make (map[string]string, 0)
	if labelsStr == "" {
		return labels, nil
	}

	if labelsStr != "" {
		labelsList := strings.Split(labelsStr, ",")
		for _, paramStr := range labelsList {
			param := strings.Split(paramStr, ":")
			if len(param) != 2 {
				return nil, derrors.NewInvalidArgumentError("invalid labels format.").WithParams(labelsStr)
			}
			labels[param[0]] = param[1]
		}
	}

	return labels, nil
}

// NeedJoin returns true if the EIC needs to send the join message
func (j * JoinHelper) NeedJoin () (bool, error) {
	_, err := os.Stat(CredentialsFile)
	if os.IsNotExist(err) {
		return true, nil
	}
	if !os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// Join calls eic-api to join de EIC
func (j * JoinHelper) Join (name string, labels string, geolocation string) (*grpc_inventory_manager_go.EICJoinResponse, error){
	log.Info().Msg("Join edge controller")
	ctx, cancel := j.getContext(DefaultTimeout)
	defer cancel()

	conn, err := j.getSecureConnection()
	defer conn.Close()
	if err != nil {
		log.Error().Str("trace", conversions.ToDerror(err).DebugReport()).Msg("cannot create the connection with the Nalej platform")
		return nil, err
	}
	client := grpc_eic_api_go.NewEICClient(conn)

	ips, ipErr := j.getAllIPs()
	if ipErr != nil {
		log.Error().Str("trace", conversions.ToDerror(ipErr).DebugReport()).Msg("cannot get IPs to create CA")
		return nil, ipErr
	}

	joinLabels, err := getLabels(labels)
	if err != nil {
		log.Error().Str("err", conversions.ToDerror(err).DebugReport()).Msg("error getting labels")
		return nil, err
	}

	joinResponse, joinErr := client.Join(ctx, &grpc_inventory_manager_go.EICJoinRequest{
		OrganizationId: j.EicToken.OrganizationId,
		Name: name,
		Labels:joinLabels,
		Geolocation: geolocation,
		Ips: ips,
	})
	if joinErr != nil {
		log.Error().Str("trace", conversions.ToDerror(joinErr).DebugReport()).Msg("error getting credentials")
		return nil, joinErr
	}
	log.Info().Interface("credentials", joinResponse.Credentials.Username).Msg("Join edge controller end")

	return joinResponse, nil
}

// ConfigureDNS adds a new dns entry in /etc/systemd/resolved.conf file
// with the dns.nalej IP
func (j * JoinHelper) ConfigureDNS () error {
	log.Info().Msg("Configuring DNS")

	ips, err := net.LookupHost(j.EicToken.DnsUrl)
	if err != nil {
		return err
	}

	// update resolved.conf
	// [Resolve]
	// DNS=...
	// Cache=no
	cmdStr := fmt.Sprintf("echo \"DNS= %s 8.8.8.8 8.8.4.4\nCache=no\" >> %s", strings.Join(ips," "), resolvedFile)
	cmd :=  exec.Command("/bin/sh", "-c", cmdStr)
	err = cmd.Run()
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error executing")
		return err
	}

	// restart the service
	log.Info().Msg("restart systemd-resolved service")
	cmd =  exec.Command("/bin/sh", "-c", "systemctl restart systemd-resolved")
	err = cmd.Run()
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error restarting service systemd-resolved")
		return err
	}

	return nil
}

func (j *JoinHelper) ExecuteDhClient () error {
	dhclientCmd := fmt.Sprintf("dhclient vpn_%s", nicName)
	cmd := exec.Command("/bin/sh", "-c", dhclientCmd)
	err := cmd.Run()
	if err != nil {
		log.Warn().Str("command", dhclientCmd).Str("error", err.Error()).Msg("error executing")
		return err
	}
	return nil

}

// GetIP enable IP4 forwarding and alter IP Table
func (j * JoinHelper) GetIP () error{
	// get IP
	cmds := []string{"echo \"net.ipv4.ip_forward=1\" >> /etc/sysctl.conf",
		"sysctl -p"}//,
		//fmt.Sprintf("dhclient vpn_%s", nicName)}
	for _, command := range cmds {
		cmd := exec.Command("/bin/sh", "-c", command)
		err := cmd.Run()
		if err != nil {
			log.Warn().Str("command", command).Str("error", err.Error()).Msg("error executing")
			return err
		}
	}
	return j.ExecuteDhClient()
}

// ConfigureLocalVPN connects to VPN server the user indicated in credentials and executes dhclient to get IP
func (j * JoinHelper) ConfigureLocalVPN (credentials *grpc_inventory_manager_go.VPNCredentials) error {

	log.Info().Str("user", credentials.Username).Msg("Configuring Local VPN")

	// NicCreate
	cmd := exec.Command(command, cmdMode, vpnClientAddress, cmdCmd, nicCreateCmd, nicName)
	err := cmd.Run()
	if err != nil {
		log.Info().Str("error", err.Error()).Msg("error creating nicName")
	}
	vpnServer := fmt.Sprintf("/SERVER:%s", credentials.Hostname)
	vpnUserName := fmt.Sprintf("/USERNAME:%s", credentials.Username)
	vpnNicName :=  fmt.Sprintf("/NICNAME:%s", nicName)

	// Account Create
	cmd = exec.Command(command, cmdMode, vpnClientAddress,cmdCmd, accountCreateCmd, credentials.Username, vpnServer, hub, vpnUserName, vpnNicName)
	err = cmd.Run()
	if err != nil {
		log.Warn().Str("error", err.Error()).Msg("error creating account")
	}

	// Account PasswordSet
	pass := fmt.Sprintf("/PASSWORD:%s", credentials.Password)
	cmd = exec.Command(command, cmdMode, vpnClientAddress,cmdCmd, accountPasswordSetCmd, credentials.Username, pass, "/TYPE:standard")
	err = cmd.Run()
	if err != nil {
		log.Warn().Str("error", err.Error()).Msg("error creating password")
	}

	cmd = exec.Command(command, cmdMode, vpnClientAddress,cmdCmd, "accountConnect", credentials.Username)
	err = cmd.Run()
	if err != nil {
		log.Warn().Str("error", err.Error()).Msg("error connecting account")
		return err
	}

	log.Info().Str("user", credentials.Username).Msg("connected")

	return nil
}

// DeleteLocalVPN disconnect the account and delete it
func (j * JoinHelper) DeleteLocalVPN () error {

	credentials, err := j.LoadCredentials()
	if err != nil {
		return err
	}

	// Disconnect
	cmd := exec.Command(command, cmdMode, vpnClientAddress, cmdCmd, accountDisconnect, credentials.Credentials.Username)
	err = cmd.Run()
	if err != nil {
		log.Info().Str("error", err.Error()).Msg("error disconnecting account")
	}

	// AccountDelete
	cmd = exec.Command(command, cmdMode, vpnClientAddress, cmdCmd, accountDelete, credentials.Credentials.Username)
	err = cmd.Run()
	if err != nil {
		log.Info().Str("error", err.Error()).Msg("error deleting account")
	}

	// RemoveCredentialsFile
	err = j.RemoveCredentials()
	if err != nil {
		log.Info().Str("error", err.Error()).Msg("error deleting credentials file")
	}


	return nil
}

func (j * JoinHelper) getContext(timeout ...time.Duration) (context.Context, context.CancelFunc) {
	md := metadata.New(map[string]string{AuthHeader: fmt.Sprintf("%s#%s", j.EicToken.Token, j.EicToken.OrganizationId)})
	log.Debug().Interface("md", md).Msg("metadata has been created")
	if len(timeout) == 0 {
		baseContext, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
		return metadata.NewOutgoingContext(baseContext, md), cancel
	}
	baseContext, cancel := context.WithTimeout(context.Background(), timeout[0])
	return metadata.NewOutgoingContext(baseContext, md), cancel
}

// GetSecureConnection returns a secure connection.
func (j * JoinHelper) getSecureConnection() (*grpc.ClientConn, derrors.Error) {

	var creds credentials.TransportCredentials
	rootCAs := x509.NewCertPool()
	//caCert:= []byte(j.Cacert)
	caCert:= []byte(j.EicToken.Cacert)

	added := rootCAs.AppendCertsFromPEM(caCert)
	if !added {
		return nil, derrors.NewInternalError("cannot add CA certificate to the pool")
	}

	creds = credentials.NewClientTLSFromCert(rootCAs, "")
	log.Debug().Interface("creds", creds.Info()).Msg("Secure credentials")

	//targetAddress := fmt.Sprintf("%s:%d", j.JoinUrl, j.JoinPort)
	targetAddress := fmt.Sprintf("%s:%d", j.EicToken.JoinUrl, j.JoinPort)
	log.Info().Str("address", targetAddress).Msg("creating connection")

	sConn, dErr := grpc.Dial(targetAddress, grpc.WithTransportCredentials(creds))
	if dErr != nil {
		return nil, derrors.AsError(dErr, "cannot create connection with the eic-api service")
	}
	return sConn, nil
}
// SaveCredentials save VPN credentials in a file
func (j * JoinHelper) SaveCredentials(edge grpc_inventory_manager_go.EICJoinResponse) error {

	log.Info().Msg("saving credentials")

	edgeJson, _ := json.Marshal(edge)
	err := ioutil.WriteFile(CredentialsFile, edgeJson, 0644)

	return err
}

// LoadCredentials load vpn credentials from a file
func (j * JoinHelper) LoadCredentials() (* grpc_inventory_manager_go.EICJoinResponse, error) {

	log.Info().Msg("loading credentials")

	credentialsFile, err := ioutil.ReadFile(CredentialsFile)
	if err != nil {
		return nil, err
	}

	credentials := &grpc_inventory_manager_go.EICJoinResponse{}

	err = json.Unmarshal(credentialsFile, &credentials)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}

// RemoveCredentials removes credentials file
func (j *JoinHelper) RemoveCredentials() error {
	remCmd := fmt.Sprintf("rm %s", CredentialsFile)
	cmd := exec.Command("/bin/sh", "-c", remCmd)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// getAllIPs return a list of IPs where edge-controller accepts connections (except VPN Address)
func (j *JoinHelper) getAllIPs () ([]string, error){

	vpnName := j.getVPNNicName()
	ips := make ([]string, 0)

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips, err
	}
	for _, iface := range interfaces {
		if iface.Name != vpnName {
			addresses, err := iface.Addrs()
			if err != nil {
				return ips, err
			}
			for _, addr := range addresses {
				netIP, ok := addr.(*net.IPNet)
				if ok && !netIP.IP.IsLoopback() && netIP.IP.To4() != nil {
					ip := netIP.IP.String()
					ips = append(ips, ip)
				}
			}
		}
	}

	return ips, nil
}

func (j * JoinHelper) getVPNNicName() string{
	return fmt.Sprintf("vpn_%s", nicName)
}

func (j * JoinHelper) GetVPNAddress() (*string, error){
	iface, err := net.InterfaceByName(j.getVPNNicName())
	if err != nil{
		return nil, err
	}

	addresses, err := iface.Addrs()
	if err != nil{
		return nil, err
	}
	for _, addr := range addresses{
		netIP, ok := addr.(*net.IPNet)
		if ok && !netIP.IP.IsLoopback() && netIP.IP.To4() != nil{
			ip := netIP.IP.String()
			return &ip, nil
		}
	}

	return nil, errors.New("cannot retrieve address list")
}
