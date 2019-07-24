/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package commands

import (
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/server"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/infra-net-plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"time"
)

// DefaultNotificationPeriod defines how often by default the EIC sends data back to the management.
const DefaultNotificationPeriod = "30s"
const DefaultAlivePeriod = "5m"

var cfg = config.Config{
	PluginConfig: viper.New(),
}
var configFile string
var configHelper *viper.Viper


var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Launch the Edge Controller API",
	Long:  `Launch the Edge Controller API`,
	Run: func(cmd *cobra.Command, args []string) {
		SetupLogging()
		err := ReadConfigFile()
		if err != nil{
			log.Fatal().Str("error", err.DebugReport()).Msg("error reading configFile")
		}
		log.Info().Msg("Launching API!")
		cfg.Debug = debugLevel
		server := server.NewService(cfg)
		server.Run()
	},
}


func init() {

	configHelper = viper.New()

	d, _ := time.ParseDuration(DefaultNotificationPeriod)
	a, _ := time.ParseDuration(DefaultAlivePeriod)

	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&configFile, "configFile", "config.yaml", "configuration file")
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
	runCmd.Flags().StringVar(&cfg.Geolocation, "geolocation", "", "Edge Controller Geolocation")
	runCmd.Flags().StringVar(&cfg.AgentBinaryPath, "agentBinaryPath", "/opt/agents", "Agents binary path as <os_arch>/service-net-agent")

	configHelper.BindPFlag("port", runCmd.Flags().Lookup("port"))
	configHelper.BindPFlag("agentPort", runCmd.Flags().Lookup("agentPort"))
	configHelper.BindPFlag("notifyPeriod", runCmd.Flags().Lookup("notifyPeriod"))
	configHelper.BindPFlag("useInMemoryProviders", runCmd.Flags().Lookup("useInMemoryProviders"))
	configHelper.BindPFlag("useBBoltProviders", runCmd.Flags().Lookup("useBBoltProviders"))
	configHelper.BindPFlag("bboltpath", runCmd.Flags().Lookup("bboltpath"))
	configHelper.BindPFlag("joinTokenPath", runCmd.Flags().Lookup("joinTokenPath"))
	configHelper.BindPFlag("eicapiPort", runCmd.Flags().Lookup("eicapiPort"))
	configHelper.BindPFlag("name", runCmd.Flags().Lookup("name"))
	configHelper.BindPFlag("labels", runCmd.Flags().Lookup("labels"))
	configHelper.BindPFlag("alivePeriod", runCmd.Flags().Lookup("alivePeriod"))
	configHelper.BindPFlag("geolocation", runCmd.Flags().Lookup("geolocation"))
	configHelper.BindPFlag("agentBinaryPath", runCmd.Flags().Lookup("agentBinaryPath"))

	// Add plugin-specific flags
	plugin.SetCommandFlags(runCmd, cfg.PluginConfig, plugin.DefaultPluginPrefix)
}

// ReadConfigFile reads config in config.yaml per default and
//  fills the config values with the values viper has
func ReadConfigFile() derrors.Error{
	log.Info().Str("configFile", configFile).Msg("reading config file")

	// Check if file exists.
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return derrors.NewInvalidArgumentError("configFile does not exist").WithParams("configFile", configFile)
	}

	configHelper.SetConfigFile(configFile)
	configHelper.ReadInConfig()

	if configHelper.IsSet("agentPort"){
		cfg.AgentPort = configHelper.GetInt("agentPort")
	}
	if configHelper.IsSet("useInMemoryProviders"){
		cfg.UseInMemoryProviders = configHelper.GetBool("useInMemoryProviders")
	}
	if configHelper.IsSet("useBBoltProviders"){
		cfg.UseBBoltProviders = configHelper.GetBool("useBBoltProviders")
	}
	if configHelper.IsSet("bboltpath"){
		cfg.BboltPath = configHelper.GetString("bboltpath")
	}
	if configHelper.IsSet("joinTokenPath"){
		cfg.JoinTokenPath = configHelper.GetString("joinTokenPath")
	}
	if configHelper.IsSet("eicapiPort"){
		cfg.EicApiPort = configHelper.GetInt("eicapiPort")
	}
	if configHelper.IsSet("name"){
		cfg.Name = configHelper.GetString("name")
	}
	if configHelper.IsSet("labels"){
		cfg.Labels = configHelper.GetString("labels")
	}
	if configHelper.IsSet("geolocation"){
		cfg.Geolocation = configHelper.GetString("geolocation")
	}
	if configHelper.IsSet("notifyPeriod"){
		cfg.NotifyPeriod = configHelper.GetDuration("notifyPeriod")
	}
	if configHelper.IsSet("alivePeriod"){
		cfg.AlivePeriod = configHelper.GetDuration("alivePeriod")
	}
	return nil
}
