/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/nalej/authx-interceptors/pkg/interceptor/apikey"
	interceptorConfig "github.com/nalej/authx-interceptors/pkg/interceptor/config"
	assetProvider "github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage"
	"github.com/nalej/edge-controller/internal/pkg/server/agent"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/edge-controller/internal/pkg/server/eic"
	"github.com/nalej/edge-controller/internal/pkg/server/helper"
	"github.com/nalej/grpc-edge-controller-go"
	"github.com/nalej/grpc-edge-inventory-proxy-go"
	"github.com/nalej/grpc-inventory-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	plugin "github.com/nalej/infra-net-plugin"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"net"
	"strings"
	"time"
)

const DefaultTimeout = 30 * time.Second

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
	metricStorageProvider metricstorage.Provider
}

type Clients struct{
	inventoryProxyClient grpc_edge_inventory_proxy_go.EdgeInventoryProxyClient
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
	var providers *Providers = nil

	if s.Configuration.UseInMemoryProviders{
		providers = s.CreateInMemoryProviders()
	} else if s.Configuration.UseBBoltProviders{
		providers = s.CreateBBoltProviders()
	} else {
		log.Fatal().Msg("unsupported type of provider")
		return nil
	}

	// In theory we can have different configurations for the metric
	// storage and retrieval - but not now
	metricConf, derr := metricstorage.NewConnectionConfig(getSubConfig(s.Configuration.PluginConfig, plugin.DefaultPluginPrefix).Sub("metrics"))
	if derr != nil {
		log.Fatal().Err(derr).Str("trace", derr.DebugReport()).Msg("unable to create metric storage provider configuration")
	}
	providers.metricStorageProvider, derr = metricstorage.NewProvider(metricConf)
	if derr != nil {
		log.Fatal().Err(derr).Str("trace", derr.DebugReport()).Msg("unable to create metric storage provider")
	}

	derr = providers.metricStorageProvider.Connect()
	if derr != nil {
		log.Fatal().Err(derr).Str("trace", derr.DebugReport()).Msg("unable to connect to metric storage provider")
	}

	return providers
}

func (s*Service) GetClients() * Clients{
	mngtConn, err := grpc.Dial(s.Configuration.ProxyURL, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Str("error", err.Error()).Msg("cannot create connection with Edge Management URL")
	}

	return &Clients{
		inventoryProxyClient: grpc_edge_inventory_proxy_go.NewEdgeInventoryProxyClient(mngtConn),
	}
}


// TODO: check the error control. When the service can not be launching the VM restart it and a lot of users can be added into VPN
// Run the service, launch the REST service handler.
func (s *Service) Run() error {

	var joinResponse *grpc_inventory_manager_go.EICJoinResponse
	joinResponse = nil

	valErr := s.Configuration.Validate()
	if valErr != nil{
		log.Fatal().Str("error", valErr.DebugReport()).Msg("Invalid configuration")
	}
	s.Configuration.Print()

	// Start plugins
	derr := startRegisteredPlugins(getSubConfig(s.Configuration.PluginConfig, plugin.DefaultPluginPrefix))
	if derr != nil {
		log.Fatal().Str("error", derr.DebugReport()).Msg("error starting plugins")
	}

	//If the controller has not done the join yet, it will have to be done
	joinHelper, err := helper.NewJoinHelper(s.Configuration.JoinTokenPath, s.Configuration.EicApiPort, s.Configuration.Name,
		s.Configuration.Labels, s.Configuration.Geolocation)
	if err != nil {
		log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("Error creating joinHelper")
	}

	needJoin, err := joinHelper.NeedJoin(s.Configuration)
	if err != nil {
		log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("Error asking for join")
	}

	log.Info().Bool("need join", needJoin).Msg("Join")

	if needJoin{
		joinResponse, err = joinHelper.Join()
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("Error in join")
		}

		err = joinHelper.SaveCredentials(*joinResponse)
		if err != nil {
			log.Info().Str("error", conversions.ToDerror(err).DebugReport()).Msg("Error saving cedentials")
		}


		// configureDNS
		err = joinHelper.ConfigureDNS()
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("enable to configure DNS")
		}

		// ConfigureLocalVPN
		err = joinHelper.ConfigureLocalVPN(joinResponse.Credentials)
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("enable to configure VPN")
		}

		err = joinHelper.GetIP()
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("getting IP")
		}
	}

	if joinResponse == nil {
		joinResponse, err = joinHelper.LoadCredentials()
		if err != nil {
			log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error getting credentials")
		}
	}

	log.Info().Str("VpnUser", joinResponse.Credentials.Username).Str("pass", strings.Repeat("*", len(joinResponse.Credentials.Password))).
		Msg("VPN credentials")

	// Store organization_id, edge_controller_id and proxyName
	s.Configuration.OrganizationId = joinResponse.OrganizationId
	s.Configuration.EdgeControllerId = joinResponse.EdgeControllerId
	s.Configuration.ProxyURL = joinResponse.Credentials.Proxyname
	s.Configuration.CaCert.Certificate = joinResponse.Certificate.Certificate
	s.Configuration.CaCert.PrivateKey = joinResponse.Certificate.PrivateKey

	providers := s.GetProviders()
	clients := s.GetClients()

	// GetVPNIP
	ip, err := joinHelper.GetVPNAddress()
	if err != nil {
		log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error getting vpnAddress")
	}

	// EICStart
	log.Info().Msg("EIC Start")
	_, err = clients.inventoryProxyClient.EICStart(context.Background(), &grpc_inventory_manager_go.EICStartInfo{
		OrganizationId: joinResponse.OrganizationId,
		EdgeControllerId: joinResponse.EdgeControllerId,
		Ip: *ip,
	})

	// launch the alive loop
	go s.aliveLoop()

	if err != nil {
		log.Fatal().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error starting EIC")
	}

	go s.LaunchEICServer(providers, clients)

	return s.LaunchAgentServer(providers, clients)
}

// aliveLoop sends alive message to proxy
func (s*Service) aliveLoop() {

	proxyClient := s.GetClients().inventoryProxyClient
	ticker := time.NewTicker(s.Configuration.AlivePeriod)

	for {
		select {
			case <- ticker.C:
				log.Info().Msg("alive")
				// Send alive message to proxy
				ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
				_, err := proxyClient.EICAlive(ctx, &grpc_inventory_go.EdgeControllerId{
					OrganizationId: s.Configuration.OrganizationId,
					EdgeControllerId: s.Configuration.EdgeControllerId,
				})
				if err != nil {
					log.Warn().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error sending the alive message")
				}
				cancel()
		}
	}
}

func (s*Service) LaunchEICServer(providers * Providers, clients * Clients) error{

	EICLis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Configuration.Port))
	if err != nil {
		log.Fatal().Errs("failed to listen: %v", []error{err})
	}

	eicManager := eic.NewManager(s.Configuration, providers.assetProvider, providers.metricStorageProvider)
	eicHandler := eic.NewHandler(eicManager)

	grpcEICServer := grpc.NewServer()
	grpc_edge_controller_go.RegisterEICServer(grpcEICServer,eicHandler)
	if s.Configuration.Debug{
		log.Info().Msg("Enabling gRPC server reflection")
		// Register reflection service on gRPC server.
		reflection.Register(grpcEICServer)
	}

	log.Info().Int("port", s.Configuration.Port).Msg("Launching gRPC server")
	if err := grpcEICServer.Serve(EICLis); err != nil {
		log.Fatal().Errs("failed to serve: %v", []error{err})
	}

	return nil
}

func (s*Service) LaunchAgentServer(providers * Providers, clients * Clients) error{
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Configuration.AgentPort))
	if err != nil {
		log.Fatal().Errs("failed to listen: %v", []error{err})
	}


	notifier := agent.NewNotifier(s.Configuration.NotifyPeriod, providers.assetProvider, clients.inventoryProxyClient,
		s.Configuration.OrganizationId, s.Configuration.EdgeControllerId)
	go notifier.LaunchNotifierLoop()

	agentManager := agent.NewManager(s.Configuration, providers.assetProvider, *notifier, clients.inventoryProxyClient)
	agentHandler := agent.NewHandler(agentManager)

	apiKeyAccess := NewAgentTokenInterceptor(providers.assetProvider)

	cfg := interceptorConfig.NewConfig(&interceptorConfig.AuthorizationConfig{
		AllowsAll: false,
		Permissions: map[string]interceptorConfig.Permission{
			"/edge_controller.Agent/AgentJoin": {Must: []string{"APIKEY"}},
			"/edge_controller.Agent/AgentCheck": {Must: []string{"APIKEY"}},
		}}, "not-used", "authorization")

	x509Cert, err := tls.X509KeyPair([]byte(s.Configuration.CaCert.Certificate), []byte(s.Configuration.CaCert.PrivateKey))
	creds :=  credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{x509Cert}})
	if err != nil {
		log.Fatal().Errs("Failed to generate credentials: %v", []error{err})
	}

	// server with apiKeyAccess and caCert
	options :=[]grpc.ServerOption{apikey.WithAPIKeyInterceptor(apiKeyAccess, cfg), grpc.Creds(creds)}
	grpcServer := grpc.NewServer(options...)
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
