package commands

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/clear-ness/qa-discussion/api"
	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/web"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run the qa-discussion server",
	RunE:  serverCmdF,
}

func init() {
	RootCmd.AddCommand(serverCmd)
	RootCmd.RunE = serverCmdF
}

func serverCmdF(command *cobra.Command, args []string) error {
	return runServer()
}

func runServer() error {
	interruptChan := make(chan os.Signal, 1)

	server, err := app.NewServer()
	if err != nil {
		return err
	}
	defer server.Shutdown()

	api.Init(server.AppOptions, server.Router)
	web.New(server, server.AppOptions, server.Router)

	serverErr := server.Start()
	if serverErr != nil {
		return serverErr
	}

	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGPIPE)
	<-interruptChan

	return nil
}
