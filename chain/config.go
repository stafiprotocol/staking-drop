package chain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shopspring/decimal"
	"github.com/stafiprotocol/go-substrate-rpc-client/submodel"
	"github.com/urfave/cli/v2"
)

const (
	defaultConfigPath   = "./config.json"
	defaultKeystorePath = "./keys"
)

var (
	ConfigFileFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "json configuration file",
		Value: defaultConfigPath,
	}

	KeystorePathFlag = &cli.StringFlag{
		Name:  "keystore",
		Usage: "path to keystore directory",
		Value: defaultKeystorePath,
	}
)

type ConfigOption struct {
	Endpoint       string                        `json:"endpoint"` // url for rpc endpoint
	KeystorePath   string                        `json:"keystorePath"`
	TypesPath      string                        `json:"typesPath"`
	BlockstorePath string                        `json:"blockstorePath"`
	StartBlock     int                           `json:"startBlock"`
	Account        string                        `json:"account"`
	DropInfos      map[submodel.RSymbol]DropInfo `json:"dropInfos"`
}

type DropInfo struct {
	MinBondAmount decimal.Decimal `json:"minBondAmount"`
	DropAmount    decimal.Decimal `json:"dropAmount"`
}

func GetConfig(ctx *cli.Context) (*ConfigOption, error) {
	var cfg ConfigOption
	path := defaultConfigPath
	if file := ctx.String(ConfigFileFlag.Name); file != "" {
		path = file
	}
	err := loadConfig(path, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadConfig(file string, config *ConfigOption) (err error) {
	ext := filepath.Ext(file)
	fp, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	f, err := os.Open(filepath.Clean(fp))
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
	}()

	if ext != ".json" {
		return fmt.Errorf("unrecognized extention: %s", ext)
	}
	return json.NewDecoder(f).Decode(&config)
}
