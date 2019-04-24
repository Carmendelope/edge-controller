/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package server

import "github.com/rs/zerolog/log"

// Service structure containing the configuration and gRPC server.
type Service struct {
	Configuration Config
}

// NewService creates a new system model service.
func NewService(conf Config) *Service {
	return &Service{
		conf,
	}
}

// Name of the service.
func (s *Service) Name() string {
	return "Edge Controller."
}

// Description of the service.
func (s *Service) Description() string {
	return "Edge controller in charge of managing a set of agents."
}

// Run the service, launch the REST service handler.
func (s *Service) Run() error {
	err := s.Configuration.Validate()
	if err != nil{
		log.Fatal().Str("error", err.DebugReport()).Msg("Invalid configuration")
	}
	s.Configuration.Print()
	return nil
}