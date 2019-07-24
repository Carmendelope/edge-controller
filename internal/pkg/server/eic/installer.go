package eic

import (
	"fmt"
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/server/agent"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/edge-controller/internal/pkg/server/connection"
	grpc_inventory_go "github.com/nalej/grpc-inventory-go"
	grpc_inventory_manager_go "github.com/nalej/grpc-inventory-manager-go"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const DefaultSSHPort = "22"

type AgentInstaller struct {
	cfg      config.Config
	notifier *agent.Notifier
}

type AgentInstallOptions struct {
	// AgentBinaryName with the name of the binary
	AgentBinaryName string
	// AgentBinaryPath with the path in the edge controller where the agent to be installed resides.
	AgentBinaryPath string
	// AgentBinarySCPTargetPath with the path where the binary will be copied prior installation.
	AgentBinarySCPTargetPath string
	// CertPath with the path where the certificates will be stored.
	CertPath string
	// CACertTargetPath with the target path to store the CA.
	CACertTargetPath string

	// CreateCertDirCmd with the command to create the certificate directory.
	CreateCertDirCmd string
	// SetExecutionPermissionsCmd with the command to give execution permissions to the executable
	SetExecutionPermissionsCmd string
	// InstallAgentCmd with the command to install the agent.
	InstallAgentCmd string
	// AgentJoinCmd with the command to perform the agent join
	AgentJoinCmd string
	// AgentStartCmd with the command to start the agent
	AgentStartCmd string
}

// NewAgentInstall creates a new installer for agents.
func NewAgentInstaller(cfg config.Config, notifier *agent.Notifier) *AgentInstaller {
	return &AgentInstaller{cfg, notifier}
}

// getAgentInstallOptions obtains the options regarding commands and path required to install an agent.
func (ai *AgentInstaller) getAgentInstallOptions(agentType grpc_inventory_manager_go.AgentType) (*AgentInstallOptions, derrors.Error) {
	if agentType == grpc_inventory_manager_go.AgentType_WINDOWS_AMD64 {
		return nil, derrors.NewUnimplementedError("windows automatic install is not supported")
	}

	agentPath := filepath.Join(ai.cfg.AgentBinaryPath, strings.ToLower(agentType.String()), "service-net-agent")

	_, err := os.Stat(agentPath)
	if err != nil {
		log.Error().Str("agentPath", agentPath).Msg("cannot access agent binary")
		return nil, derrors.AsError(err, "agent binary is not accessible")
	}

	return &AgentInstallOptions{
		AgentBinaryName:          "service-net-agent",
		AgentBinaryPath:          agentPath,
		AgentBinarySCPTargetPath: ".",
		CertPath:                 "/opt/nalej/certs",
		CACertTargetPath:         "/opt/nalej/certs/cacert.pem",
		CreateCertDirCmd:         "mkdir -p /opt/nalej/certs",
		SetExecutionPermissionsCmd: "chmod +x service-net-agent",
		InstallAgentCmd:          "./service-net-agent install",
		AgentJoinCmd:             "/opt/nalej/bin/service-net-agent join --token=%s --address=%s:5588 --cert=/opt/nalej/certs/cacert.pem",
		AgentStartCmd:            "/opt/nalej/bin/service-net-agent start",
	}, nil
}

// getAgentJoinCmd substitutes the missing parameters in the agent join command.
func (aio *AgentInstallOptions) getAgentJoinCmd(agentJoinToken string, edgeControllerIP string) string {
	return fmt.Sprintf(aio.AgentJoinCmd, agentJoinToken, edgeControllerIP)
}

// getBaseResponse returns a base response to send an update on the progress of the install.
func (ai *AgentInstaller) getBaseResponse(operationID string, request *grpc_inventory_manager_go.InstallAgentRequest) *grpc_inventory_manager_go.EdgeControllerOpResponse {
	return &grpc_inventory_manager_go.EdgeControllerOpResponse{
		OrganizationId:   request.OrganizationId,
		EdgeControllerId: request.EdgeControllerId,
		OperationId:      operationID,
		Timestamp:        time.Now().Unix(),
	}
}

/*
  156  ./service-net-agent install
  157  ls
  158  mkdir /opt/nalej/certs
  159  cp cacert.pem /opt/nalej/certs/
  160  /opt/nalej/bin/service-net-agent join --help
  161  /opt/nalej/bin/service-net-agent join --token=d62cc38f-64a7-4d7c-b591-a2032dad0ef5 --address=172.16.17.93:5588 --debug --cert=/opt/nalej/certs/cacert.pem
  162  /opt/nalej/bin/service-net-agent start
*/
// InstallAgent triggers the steps required to install the agent.
func (ai *AgentInstaller) InstallAgent(operationID string, agentJoinToken string, request *grpc_inventory_manager_go.InstallAgentRequest) {
	log.Debug().Interface("request", request).Msg("triggering agent install")

	options, optErr := ai.getAgentInstallOptions(request.AgentType)
	if optErr != nil {
		update := ai.getBaseResponse(operationID, request)
		update.Status = grpc_inventory_go.OpStatus_FAIL
		update.Info = optErr.Error()
		ai.notifier.NotifyECOpResponse(update)
		return
	}
	log.Debug().Interface("options", options).Msg("install options defined")
	start := time.Now()
	edgeControllerIP, err := ai.detectEdgeControllerIP(operationID, request)
	if err != nil {
		log.Debug().Str("trace", err.DebugReport()).Msg("cannot detect edge controller IP")
		return
	}
	log.Debug().Str("IP", edgeControllerIP).Msg("edge controller IP as seen by the asset")
	// First copy the agent binary to the target host
	err = ai.copyBinaryToAsset(options, operationID, request)
	if err != nil {
		log.Debug().Str("trace", err.DebugReport()).Msg("cannot copy agent binary")
		return
	}
	// Copy the CA
	err = ai.createCacert(options, operationID, request)
	if err != nil {
		log.Debug().Str("trace", err.DebugReport()).Msg("cannot create CA cert on target host")
		return
	}
	// Set exec permissions
	err = ai.execSSHCommand(options.SetExecutionPermissionsCmd, operationID, request)
	if err != nil {
		log.Debug().Str("trace", err.DebugReport()).Msg("cannot set execution permissions")
		return
	}
	// Install the agent.
	err = ai.execSSHCommand(options.InstallAgentCmd, operationID, request)
	if err != nil {
		log.Debug().Str("trace", err.DebugReport()).Msg("cannot install agent")
		return
	}
	// Join the agent
	agentJoinCmd := options.getAgentJoinCmd(agentJoinToken, edgeControllerIP)
	err = ai.execSSHCommand(agentJoinCmd, operationID, request)
	if err != nil {
		log.Debug().Str("trace", err.DebugReport()).Msg("cannot join agent")
		return
	}
	// Start the agent
	err = ai.execSSHCommand(options.AgentStartCmd, operationID, request)
	if err != nil {
		log.Debug().Str("trace", err.DebugReport()).Msg("cannot start agent")
		return
	}
	log.Info().Str("targetHost", request.TargetHost).Msg("agent has been installed")
	// Send success update
	update := ai.getBaseResponse(operationID, request)
	update.Status = grpc_inventory_go.OpStatus_SUCCESS
	update.Info = fmt.Sprintf("Agent has been installed, took %s", time.Since(start).String())
	nErr := ai.notifier.NotifyECOpResponse(update)
	if nErr != nil {
		log.Error().Str("trace", nErr.DebugReport()).Msg("notify EC op response failed")
	}
}

// copyBinaryToAsset copies the agent binary to the remote asset so that it can be installed.
func (ai *AgentInstaller) copyBinaryToAsset(options *AgentInstallOptions, operationID string, request *grpc_inventory_manager_go.InstallAgentRequest) derrors.Error {
	log.Debug().Str("agentPath", options.AgentBinaryPath).Msg("copying binary to asset")
	conn, err := connection.NewSSHConnection(
		request.TargetHost, DefaultSSHPort,
		request.Credentials.Username, request.Credentials.GetPassword(), "", request.Credentials.GetClientCertificate())
	if err != nil {
		dErr := derrors.NewInternalError("cannot establish ssh connection", err).WithParams(request.TargetHost)
		ai.notifyResult(operationID, request, dErr, "")
		return dErr
	}
	start := time.Now()

	isSudoer := false
	if request.Credentials != nil && request.Credentials.IsSudoer{
		isSudoer = true
	}
	err = conn.Copy(options.AgentBinaryPath, options.AgentBinarySCPTargetPath, false, isSudoer)
	if err != nil {
		dErr := derrors.NewInternalError("cannot scp agent binary", err).WithParams(request.TargetHost)
		ai.notifyResult(operationID, request, dErr, "")
		return dErr
	}
	msg := fmt.Sprintf("agent binary copied in %s", time.Since(start).String())
	ai.notifyResult(operationID, request, nil, msg)
	return nil
}

// createCacertDir creates the directory to store the certificates for the agent and transfer the appropiate CA.
func (ai *AgentInstaller) createCacert(options *AgentInstallOptions, operationID string, request *grpc_inventory_manager_go.InstallAgentRequest) derrors.Error {
	// Create the directory
	cErr := ai.execSSHCommand(options.CreateCertDirCmd, operationID, request)
	if cErr != nil {
		ai.notifyResult(operationID, request, cErr, "")
		return cErr
	}

	// Create the file locally.
	f, err := ioutil.TempFile("", operationID)
	if err != nil {
		dErr := derrors.AsError(err, "cannot create temp file")
		ai.notifyResult(operationID, request, dErr, "")
		return dErr
	}
	defer os.Remove(f.Name())

	// Copy the content of the ca_cert to the file
	// TODO enable scp from buffer
	err = ioutil.WriteFile(f.Name(), []byte(request.CaCert), os.ModePerm)
	if err != nil {
		dErr := derrors.AsError(err, "cannot write cacert to temp file")
		ai.notifyResult(operationID, request, dErr, "")
		return dErr
	}

	// Now copy the file.
	conn, err := connection.NewSSHConnection(
		request.TargetHost, DefaultSSHPort,
		request.Credentials.Username, request.Credentials.GetPassword(), "", request.Credentials.GetClientCertificate())
	if err != nil {
		dErr := derrors.NewInternalError("cannot establish ssh connection", err).WithParams(request.TargetHost)
		ai.notifyResult(operationID, request, dErr, "")
		return dErr
	}
	start := time.Now()
	isSudoer := false
	if request.Credentials != nil && request.Credentials.IsSudoer{
		isSudoer = true
	}
	err = conn.Copy(f.Name(), options.CACertTargetPath, false, isSudoer)
	if err != nil {
		dErr := derrors.NewInternalError("cannot scp CA Cert", err).WithParams(request.TargetHost)
		ai.notifyResult(operationID, request, dErr, "")
		return dErr
	}
	msg := fmt.Sprintf("CA cert copied in %s", time.Since(start).String())
	ai.notifyResult(operationID, request, nil, msg)
	return nil
}

// execSSHCommand executes an SSH command.
func (ai *AgentInstaller) execSSHCommand(cmd string, operationID string, request *grpc_inventory_manager_go.InstallAgentRequest) derrors.Error {

	conn, err := connection.NewSSHConnection(
		request.TargetHost, DefaultSSHPort,
		request.Credentials.Username, request.Credentials.GetPassword(), "", request.Credentials.GetClientCertificate())
	if err != nil {
		dErr := derrors.AsError(err, "cannot establish ssh connection")
		ai.notifyResult(operationID, request, dErr, "")
		return dErr
	}

	// check if the user is sudoer [NP-1602]
	if request.Credentials != nil && request.Credentials.IsSudoer {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}

	log.Debug().Str("toExecute", cmd).Msg("SSH exec")
	output, err := conn.Execute(cmd)
	if err != nil {
		dErr := derrors.NewInternalError("cannot execute ssh command", err)
		ai.notifyResult(operationID, request, dErr, "")
		return dErr
	}
	ai.notifyResult(operationID, request, nil, string(output))
	return nil
}

// notifyResult sends an update on a given operation.
func (ai *AgentInstaller) notifyResult(operationID string, request *grpc_inventory_manager_go.InstallAgentRequest, err derrors.Error, info string) {
	update := ai.getBaseResponse(operationID, request)
	if err == nil {
		update.Status = grpc_inventory_go.OpStatus_INPROGRESS
		update.Info = info
	} else {
		update.Status = grpc_inventory_go.OpStatus_FAIL
		update.Info = err.Error()
	}
	log.Debug().Interface("update", update).Msg("sending notification on ec operation")
	nErr := ai.notifier.NotifyECOpResponse(update)
	if nErr != nil {
		log.Error().Str("trace", nErr.DebugReport()).Msg("notify EC op response failed")
	}
}

// detectEdgeControllerIP atempts to detect the IP address of the edge controler as seen by the asset. In order to do
// that we use ssh <targetHost> env | grep SSH_CONNECTION
func (ai *AgentInstaller) detectEdgeControllerIP(operationID string, request *grpc_inventory_manager_go.InstallAgentRequest) (string, derrors.Error) {
	envCmd := "env"
	conn, err := connection.NewSSHConnection(
		request.TargetHost, DefaultSSHPort,
		request.Credentials.Username, request.Credentials.GetPassword(), "", request.Credentials.GetClientCertificate())
	if err != nil {
		dErr := derrors.AsError(err, "cannot establish ssh connection")
		ai.notifyResult(operationID, request, dErr, "")
		return "", dErr
	}
	log.Debug().Str("toExecute", envCmd).Msg("SSH exec")
	output, err := conn.Execute(envCmd)
	if err != nil {
		dErr := derrors.NewInternalError("cannot execute ssh command", err)
		ai.notifyResult(operationID, request, dErr, "")
		return "", dErr
	}
	// Now grep the SSH_CONNECTION
	var re = regexp.MustCompile(`.*SSH_CLIENT=.*`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) != 1 {
		dErr := derrors.NewInternalError("cannot find SSH_CLIENT in output", err)
		ai.notifyResult(operationID, request, dErr, "")
		return "", dErr
	}
	withoutVar := strings.Replace(matches[0], "SSH_CLIENT=", "", 1)
	splits := strings.Split(withoutVar, " ")
	return splits[0], nil
}
