/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package commands

import (
	"github.com/nalej/edge-controller/internal/pkg/server"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var config = server.Config{}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Launch the Edge Controller API",
	Long:  `Launch the Edge Controller API`,
	Run: func(cmd *cobra.Command, args []string) {
		SetupLogging()
		log.Info().Msg("Launching API!")
		config.Debug = debugLevel
		server := server.NewService(config)
		server.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().IntVar(&config.Port, "port", 5555, "Port to receive management communications")
	runCmd.Flags().IntVar(&config.AgentPort, "agentPort", 5556, "Port to receive agent messages")
}