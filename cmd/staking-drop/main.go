package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/stafiprotocol/staking-drop/chain"
	"github.com/urfave/cli/v2"
)

var app = cli.NewApp()

var mainFlags = []cli.Flag{
	chain.ConfigFileFlag,
}

var generateFlags = []cli.Flag{
	KeystorePathFlag,
	NetworkFlag,
}

// init initializes CLI
func init() {
	app.Action = run
	app.Copyright = "Copyright 2022 Stafi Protocol Authors"
	app.Name = "staking-dropd"
	app.Usage = "staking-dropd"
	app.Authors = []*cli.Author{{Name: "Stafi Protocol 2022"}}
	app.Version = "0.0.1"
	app.EnableBashCompletion = true
	app.Commands = []*cli.Command{
		{
			Action:      handleGenerateSubCmd,
			Name:        "gensub",
			Usage:       "generate subsrate keystore",
			Flags:       generateFlags,
			Description: "The generate subcommand is used to generate the substrate keystore.",
		},
	}

	app.Flags = append(app.Flags, mainFlags...)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {

	cfg, err := chain.GetConfig(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("config: %+v\n", cfg)

	// Used to signal core shutdown due to fatal error
	sysErr := make(chan error)

	stafiChain := chain.NewChain()
	// logger := log.Root()
	err = stafiChain.Initialize(cfg, log, sysErr)
	if err != nil {
		return err
	}

	// =============== start
	err = stafiChain.Start()
	if err != nil {
		return err
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)

	// Block here and wait for a signal
	select {
	case err := <-sysErr:
		log.Error("FATAL ERROR. Shutting down.", "err", err)
	case <-sigc:
		log.Warn("Interrupt received, shutting down now.")
	}
	stafiChain.Stop()
	return nil
}
