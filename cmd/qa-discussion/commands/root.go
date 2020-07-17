package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Command = cobra.Command

func Run(args []string) error {
	RootCmd.SetArgs(args)
	return RootCmd.Execute()
}

var RootCmd = &cobra.Command{
	Use:   "qa-discussion",
	Short: "fundamental modern questions & answers system.",
}

func init() {
	RootCmd.PersistentFlags().StringP("config", "c", "config.json", "config file name.")

	viper.SetEnvPrefix("qa-discussion")
	viper.BindEnv("config")

	viper.BindPFlag("config", RootCmd.PersistentFlags().Lookup("config"))
}
