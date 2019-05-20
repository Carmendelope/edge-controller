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
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	agentClient grpc_inventory_manager_go.AgentClient
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
	mngtConn, err := grpc.Dial(s.Configuration.EdgeManagementURL, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Str("error", err.Error()).Msg("cannot create connection with Edge Management URL")
	}

	return &Clients{
		agentClient: grpc_inventory_manager_go.NewAgentClient(mngtConn),
	}
}

// Run the service, launch the REST service handler.
func (s *Service) Run() error {
	err := s.Configuration.Validate()
	if err != nil{
		log.Fatal().Str("error", err.DebugReport()).Msg("Invalid configuration")
	}
	s.Configuration.Print()

	//If the controller has not done the join yet, it will have to be done
	joinHelper, jErr := helper.NewJoinHelper(s.Configuration.JoinTokenPath, s.Configuration.EicApiPort, s.Configuration.Name, s.Configuration.Labels)
	if jErr != nil {
		log.Fatal().Str("error", conversions.ToDerror(jErr).DebugReport()).Msg("Error creating joinHelper")
	}

	//joinHelper.Test()


	needJoin, nErr := joinHelper.NeedJoin(s.Configuration)
	if nErr != nil {
		log.Fatal().Str("error", conversions.ToDerror(nErr).DebugReport()).Msg("Error asking for join")
	}

	if needJoin{
		credentials, jErr := joinHelper.Join()
		if jErr != nil {
			log.Fatal().Str("error", conversions.ToDerror(jErr).DebugReport()).Msg("Error in join")
		}

		// configureDNS
		errJoin := joinHelper.ConfigureDNS()
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(errJoin).DebugReport()).Msg("enable to configure DNS")
		}

		// ConfigureLocalVPN
		errVpn := joinHelper.ConfigureLocalVPN(credentials)
		if errVpn != nil {
			log.Fatal().Str("error", conversions.ToDerror(errVpn).DebugReport()).Msg("enable to configure VPN")
		}
	}


// TODO: get credentials.Hostname and load ManagementCluster

	providers := s.GetProviders()
	clients := s.GetClients()
	return s.LaunchAgentServer(providers, clients)
}

func (s*Service) LaunchEICServer() error{
	// TODO Launch EIC Server
	return nil
}

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