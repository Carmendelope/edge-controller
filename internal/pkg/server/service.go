/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package server

import (
	"fmt"
	assetProvider "github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/nalej/edge-controller/internal/pkg/server/agent"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/edge-controller/internal/pkg/server/helper"
	"github.com/nalej/grpc-edge-controller-go"
	"github.com/nalej/grpc-edge-inventory-proxy-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io/ioutil"
	"net"
)

// Service structure containing the configuration and gRPC server.
type Service struct {
	Configuration config.Config
}

// NewService creates a new system model service.
func NewService(conf config.Config) *Service {
	return &Service{
		conf,
	}
}

type Providers struct{
	assetProvider assetProvider.Provider
}

type Clients struct{
	agentClient grpc_edge_inventory_proxy_go.EdgeInventoryProxyClient
}

// Name of the service.
func (s *Service) Name() string {
	return "Edge Controller."
}

// Description of the service.
func (s *Service) Description() string {
	return "Edge controller in charge of managing a set of agents."
}

// CreateInMemoryProviders creates MockupProviders.
func (s*Service) CreateInMemoryProviders() * Providers{
	return &Providers{
		assetProvider: assetProvider.NewMockupAssetProvider(),
	}
}

// CreateBBoltProviders creates Bboltroviders.
func (s*Service) CreateBBoltProviders() * Providers{
	return &Providers{
		assetProvider: assetProvider.NewBboltAssetProvider(s.Configuration.BboltPath),
	}
}

func (s*Service) GetProviders() * Providers{
	if s.Configuration.UseInMemoryProviders{
		return s.CreateInMemoryProviders()
	}else {
		if s.Configuration.UseBBoltProviders{
			return s.CreateBBoltProviders()
		}
	}

	log.Fatal().Msg("unsupported type of provider")
	return nil
}

func (s*Service) GetClients() * Clients{
	// TODO Update type of connection
	mngtConn, err := grpc.Dial(s.Configuration.ProxyURL, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Str("error", err.Error()).Msg("cannot create connection with Edge Management URL")
	}

	return &Clients{
		agentClient: grpc_edge_inventory_proxy_go.NewEdgeInventoryProxyClient(mngtConn),
	}
}

func (s *Service) saveHostName (hostname string) error {
	return ioutil.WriteFile("/etc/edge-controller/proxyname", []byte(hostname), 0644)
}

func (s *Service) readHostName () (string, error) {
	dat, err := ioutil.ReadFile("/etc/edge-controller/proxyname")
	if err != nil {
		return "", err
	}

	return string(dat), nil
}

// TODO: check the error control. When the service can not be launching the VM restart it and a lot of users can be added into VPN
// Run the service, launch the REST service handler.
func (s *Service) Run() error {

	proxyURL:= ""

	valErr := s.Configuration.Validate()
	if valErr != nil{
		log.Fatal().Str("error", valErr.DebugReport()).Msg("Invalid configuration")
	}
	s.Configuration.Print()

	//If the controller has not done the join yet, it will have to be done
	joinHelper, err := helper.NewJoinHelper(s.Configuration.JoinTokenPath, s.Configuration.EicApiPort, s.Configuration.Name, s.Configuration.Labels)
	if err != nil {
		log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("Error creating joinHelper")
	}

	needJoin, err := joinHelper.NeedJoin(s.Configuration)
	if err != nil {
		log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("Error asking for join")
	}

	log.Info().Bool("need join", needJoin).Msg("Join")

	if needJoin{
		log.Info().Msg("Join needed!")
		credentials, err := joinHelper.Join()
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("Error in join")
		}
		proxyURL = credentials.Proxyname
		// save the hostname in a file to get it if no join is need
		s.saveHostName(proxyURL)

		// configureDNS
		err = joinHelper.ConfigureDNS()
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("enable to configure DNS")
		}

		// ConfigureLocalVPN
		err = joinHelper.ConfigureLocalVPN(credentials)
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("enable to configure VPN")
		}

		err = joinHelper.GetIP()
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("getting IP")
		}
	}

	if proxyURL == "" {
		proxyURL, err = s.readHostName()
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error getting proxy URL")
		}
	}

	// get port 443 by default
	s.Configuration.ProxyURL = proxyURL
	log.Info().Str("vpn_proxy", s.Configuration.ProxyURL).Msg("ProxyURL")

	providers := s.GetProviders()
	clients := s.GetClients()
	return s.LaunchAgentServer(providers, clients)
}

func (s*Service) LaunchEICServer() error{
	// TODO Launch EIC Server
	return nil
}

// TODO:
func (s*Service) LaunchAgentServer(providers * Providers, clients * Clients) error{
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Configuration.AgentPort))
	if err != nil {
		log.Fatal().Errs("failed to listen: %v", []error{err})
	}
	notifier := agent.NewNotifier(s.Configuration.NotifyPeriod, providers.assetProvider, clients.agentClient)
	go notifier.LaunchNotifierLoop()

	agentManager := agent.NewManager(s.Configuration, providers.assetProvider, *notifier, clients.agentClient)
	agentHandler := agent.NewHandler(agentManager)
	grpcServer := grpc.NewServer()
	grpc_edge_controller_go.RegisterAgentServer(grpcServer, agentHandler)

	if s.Configuration.Debug{
		log.Info().Msg("Enabling gRPC server reflection")
		// Register reflection service on gRPC server.
		reflection.Register(grpcServer)
	}

	log.Info().Int("port", s.Configuration.AgentPort).Msg("Launching Agent gRPC server")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal().Errs("failed to serve: %v", []error{err})
	}
	return nil
}