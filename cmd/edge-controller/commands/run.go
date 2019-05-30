/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package commands

import (
	"github.com/nalej/edge-controller/internal/pkg/server"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/service-net-agent/pkg/plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"
)

// DefaultNotificationPeriod defines how often by default the EIC sends data back to the management.
const DefaultNotificationPeriod = "30s"
const DefaultAlivePeriod = "5m"

var cfg = config.Config{
	PluginConfig: viper.New(),
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Launch the Edge Controller API",
	Long:  `Launch the Edge Controller API`,
	Run: func(cmd *cobra.Command, args []string) {
		SetupLogging()
		log.Info().Msg("Launching API!")
		cfg.Debug = debugLevel
		server := server.NewService(cfg)
		server.Run()
	},
}

func init() {

	d, _ := time.ParseDuration(DefaultNotificationPeriod)
	a, _ := time.ParseDuration(DefaultAlivePeriod)

	rootCmd.AddCommand(runCmd)
	runCmd.Flags().IntVar(&cfg.Port, "port", 5577, "Port to receive management communications")
	runCmd.Flags().IntVar(&cfg.AgentPort, "agentPort", 5588, "Port to receive agent messages")
	runCmd.Flags().DurationVar(&cfg.NotifyPeriod, "notifyPeriod", d, "Notification period to the management cluster")
	runCmd.Flags().BoolVar(&cfg.UseInMemoryProviders, "useInMemoryProviders", false,"Use InMemory providers")
	runCmd.Flags().BoolVar(&cfg.UseBBoltProviders, "useBBoltProviders", false,"Use Bbolt providers")
	runCmd.Flags().StringVar(&cfg.BboltPath, "bboltpath", "", "Database path")
	runCmd.Flags().StringVar(&cfg.JoinTokenPath, "joinTokenPath", "", "Token Path")
	runCmd.Flags().IntVar(&cfg.EicApiPort, "eicapiPort", 443, "Port to send the join message")
	runCmd.Flags().StringVar(&cfg.Name, "name", "", "Edge controller name")
	runCmd.Flags().StringVar(&cfg.Labels, "labels", "", "Edge controller labels")
	runCmd.Flags().DurationVar(&cfg.AlivePeriod, "alivePeriod", a,"Notification period to the management cluster")

	// Add plugin-specific flags
	plugin.SetCommandFlags(runCmd, cfg.PluginConfig, plugin.DefaultPluginPrefix)
}
