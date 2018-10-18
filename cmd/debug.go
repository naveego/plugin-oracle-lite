package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/naveego/plugin-oracle/internal"
	"github.com/naveego/plugin-oracle/internal/pub"
	"github.com/spf13/cobra"
	"os"
)

var log hclog.Logger
var server pub.PublisherServer

var debugCmd = &cobra.Command{
	Use:   "debug",
	Args:cobra.ExactArgs(2),
	Hidden:true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log = hclog.New(&hclog.LoggerOptions{
			Level:      hclog.Trace,
			Output:     os.Stderr,
			JSONFormat: false,
			Name: "plugin-oracle",
		})

		server = internal.NewServer(log)
	},
}

var connectCmd = &cobra.Command{
	Use:   "connect {json-settings}",
	Args:cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		req := &pub.ConnectRequest{
			SettingsJson:args[0],
		}

		resp, err := server.Connect(context.Background(), req)
		if err != nil {
			return err
		}

		j, _ := json.Marshal(resp)

		fmt.Println(j)

		return nil
	},
}

func init(){
	debugCmd.AddCommand(connectCmd)
	RootCmd.AddCommand(debugCmd)
}