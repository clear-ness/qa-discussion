package commands

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// if you want to run jobs on jobservers, use this command
var JobserverCmd = &cobra.Command{
	Use:   "jobserver",
	Short: "run the qa-discussion job server",
	RunE:  jobserverCmdF,
}

func init() {
	RootCmd.AddCommand(JobserverCmd)
}

func jobserverCmdF(command *cobra.Command, args []string) error {
	if len(args) < 1 || (args[0] != model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR && args[0] != model.NOTIFICATION_INBOX_INTERVAL_DAY && args[0] != model.NOTIFICATION_INBOX_INTERVAL_WEEK) {
		return errors.New("need interval argument")
	}

	interruptChan := make(chan os.Signal, 1)

	a, err := initContext(&args[0])
	if err != nil {
		return err
	}

	defer func() {
		a.Shutdown()
	}()

	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interruptChan

	return nil
}

func initContext(interval *string) (*app.App, error) {
	var options []app.Option
	options = append(options, app.InitEmailBatching(interval))

	server, err := app.NewServer(options...)
	if err != nil {
		return nil, err
	}

	a := server.FakeApp()

	return a, nil
}
