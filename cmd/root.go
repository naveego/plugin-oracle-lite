// Copyright © 2017 Naveego

package cmd

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/lestrrat-go/file-rotatelogs"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/naveego/dataflow-contracts/plugins"
	"github.com/naveego/plugin-oracle/internal"
	"github.com/naveego/plugin-oracle/internal/pub"
	"github.com/naveego/plugin-oracle/version"
	"github.com/spf13/cobra"
)

var verbose *bool

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "plugin-pub-mssql",
	Short: "A publisher that pulls data from SQL.",
	Long: fmt.Sprintf(`Version %s
Runs the publisher in externally controlled mode.`, version.Version.String()),
	Run: func(cmd *cobra.Command, args []string)  {

		logf, err := rotatelogs.New(
			"./log.%Y%m%d%H%M%S",
			// rotatelogs.WithLinkName("./log"),
			rotatelogs.WithMaxAge(7 * 24 * time.Hour),
			rotatelogs.WithRotationTime(time.Hour),
		)
		if err != nil {
			recordCrashFile( fmt.Sprintf("Could not create log file: %s", err))
		}

		log := hclog.New(&hclog.LoggerOptions{
			Level:      hclog.Trace,
			Output:     io.MultiWriter(os.Stderr, logf),
			JSONFormat: true,
		})

		plugin.Serve(&plugin.ServeConfig{
			HandshakeConfig: plugin.HandshakeConfig{
				ProtocolVersion: plugins.PublisherProtocolVersion,
				MagicCookieKey:plugins.PublisherMagicCookieKey,
				MagicCookieValue:plugins.PublisherMagicCookieValue,
			},
			Plugins: map[string]plugin.Plugin{
				"publisher": pub.NewServerPlugin(internal.NewServer(log)),
			},
			Logger: log,

			// A non-nil value here enables gRPC serving for this plugin...
			GRPCServer: plugin.DefaultGRPCServer,
		})
	}}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}



func init() {
	verbose = RootCmd.Flags().BoolP("verbose", "v", false, "enable verbose logging")
}

func recordCrashFile(message string) {
	startupFailureErrorPath := fmt.Sprintf("./crash-%d-%s.log", time.Now().Unix(), uuid.New().String())
	_ = ioutil.WriteFile(startupFailureErrorPath, []byte(message), 0666)
}