package helper

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
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
	nicName ="nicName"
	nicUser ="/NICNAME:nicname"
	accountCreateCmd = "AccountCreate"
	accountPasswordSetCmd = "AccountPasswordSet"
	vpnClientAddress = "localhost"
)
const DefaultTimeout = time.Minute

const AuthHeader = "Authorization"


type JoinHelper struct {
	// JoinTokenFile path
	JoinTokenFile string
	// OrganizationID with the organization identifier
	OrganizationId string
	// Token to be used by the agent.
	Token string
	// Cacert with the CA certificate.
	Cacert string
	// JoinURL with the URL the EIC needs to send the message for starting the join operation.
	JoinUrl string
	// JoinPort with the URL the EIC needs to send the message for starting the join operation.
	JoinPort int
	// Name with the edge controller name
	Name string
	// labels with the edge controller labels
	Labels map[string]string
	DnsUrl string
}

func NewJoinHelper (configFile string, port int, name string, labels string ) (*JoinHelper, error) {

	jsonFile, err :=  os.Open(configFile)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("tokenFile", configFile).Msg("Successfully Opened")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var eicToken grpc_inventory_manager_go.EICJoinToken
	err = json.Unmarshal(byteValue, &eicToken)
	if err != nil {
		log.Error().Str("err", conversions.ToDerror(err).DebugReport()).Msg("error Unmarshalling joinTokenFile")
		return nil, err
	}

	joinLabels, err := getLabels(labels)
	if err != nil {
		log.Error().Str("err", conversions.ToDerror(err).DebugReport()).Msg("error getting labels")
		return nil, err
	}

	return &JoinHelper{
		JoinPort: port,
		Name: name,
		OrganizationId:eicToken.OrganizationId,
		Token: eicToken.Token,
		Cacert: eicToken.Cacert,
		JoinUrl: eicToken.JoinUrl,
		Labels: joinLabels,
		DnsUrl: eicToken.DnsUrl,
	}, nil
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
func (j * JoinHelper) NeedJoin (config  config.Config) (bool, error) {
	_, err := os.Stat(config.BboltPath)
	if os.IsNotExist(err) {
		return true, nil
	}

	return err != nil, err

}

func (j * JoinHelper) Join () (*grpc_inventory_manager_go.VPNCredentials, error){
	ctx, cancel := j.getContext(DefaultTimeout)
	defer cancel()

	conn, err := j.getSecureConnection()
	defer conn.Close()
	if err != nil {
		log.Error().Str("trace", conversions.ToDerror(err).DebugReport()).Msg("cannot create the connection with the Nalej platform")
		return nil, err
	}
	client := grpc_eic_api_go.NewEICClient(conn)

	joinResponse, joinErr := client.Join(ctx, &grpc_inventory_manager_go.EICJoinRequest{
		OrganizationId: j.OrganizationId,
		Name: j.Name,
		Labels: j.Labels,
	})
	if joinErr != nil {
		log.Error().Str("trace", conversions.ToDerror(joinErr).DebugReport()).Msg("error getting credentials")
		return nil, joinErr
	}

	return joinResponse.Credentials, nil
}

func (j * JoinHelper) writeDNSNameServer (ip string) error {
	log.Info().Str("dns", ip).Msg("adding dns entry	")
	auxFile := "/home/vagrant/aux.conf"
	resolveConf := "/etc/resolv.conf"
	name := fmt.Sprintf("nameserver %s", ip)

	cmds := []string{
		fmt.Sprintf("cat %s >> %s", resolveConf, auxFile),
		fmt.Sprintf("echo '%s' >> %s",name,auxFile),
		fmt.Sprintf("cp %s %s",auxFile,resolveConf),
		fmt.Sprintf("rm %s", auxFile),
	}
	for _, inst := range cmds {
		cmd := exec.Command("/bin/sh", "-c", inst)
		err := cmd.Run()
		if err != nil {
			log.Error().Str("error", err.Error()).Str("inst", inst).Msg("error executing")
			return err
		}
	}
	//log.Fatal().Msg("stop!")

	log.Info().Msg("dns added in /etc/resolv.conf file")

	return nil
}

func (j * JoinHelper) Test() error  {


	return nil
}

func (j * JoinHelper) ConfigureDNS () error {
	log.Info().Msg("Configuring DNS")

	ips, err := net.LookupIP(j.DnsUrl)
	if err != nil {
		return err
	}
	for _, ip := range ips {

		j.writeDNSNameServer(ip.String())
	}

	return nil
}

func (j * JoinHelper) ConfigureLocalVPN (credentials *grpc_inventory_manager_go.VPNCredentials) error {

	log.Info().Str("user", credentials.Username).Msg("Configuring Local VPN")

	// NicCreate
	cmd := exec.Command(command, cmdMode, vpnClientAddress, cmdCmd, nicCreateCmd, nicName)
	err := cmd.Run()
	if err != nil {
		log.Info().Str("error", err.Error()).Msg("error creating nicName")
		//return err
	}
	vpnServer := fmt.Sprintf("/SERVER:vpn-server.%s:5555", credentials.Hostname)
	vpnUserName := fmt.Sprintf("/USERNAME:%s", credentials.Username)
	// Account Create
	cmd = exec.Command(command, cmdMode, vpnClientAddress,cmdCmd, accountCreateCmd, credentials.Username, vpnServer, hub, vpnUserName, nicUser)
	err = cmd.Run()
	if err != nil {
		log.Warn().Str("error", err.Error()).Msg("error creating account")
		//return err
	}
	// Account PasswordSet
	pass := fmt.Sprintf("/PASSWORD:%s", credentials.Password)
	cmd = exec.Command(command, cmdMode, vpnClientAddress,cmdCmd, accountPasswordSetCmd, credentials.Username, pass, "/TYPE:standard")
	err = cmd.Run()
	if err != nil {
		log.Warn().Str("error", err.Error()).Msg("error creating password")
		//return err
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

func (j * JoinHelper) getContext(timeout ...time.Duration) (context.Context, context.CancelFunc) {
	md := metadata.New(map[string]string{AuthHeader: fmt.Sprintf("%s#%s", j.Token, j.OrganizationId)})
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
	caCert:= []byte(j.Cacert)

	added := rootCAs.AppendCertsFromPEM(caCert)
	if !added {
		return nil, derrors.NewInternalError("cannot add CA certificate to the pool")
	}

	creds = credentials.NewClientTLSFromCert(rootCAs, "")
	log.Debug().Interface("creds", creds.Info()).Msg("Secure credentials")

	targetAddress := fmt.Sprintf("%s:%d", j.JoinUrl, j.JoinPort)
	log.Debug().Str("address", targetAddress).Msg("creating connection")

	sConn, dErr := grpc.Dial(targetAddress, grpc.WithTransportCredentials(creds))
	if dErr != nil {
		return nil, derrors.AsError(dErr, "cannot create connection with the eic-api service")
	}
	return sConn, nil
}