package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/urfave/cli/v2"
)

const (
	rpcServerKey   = "rpcserver"
	tlsCertPathKey = "tls_cert_path"
	insecureKey    = "insecure"
)

var config = cli.Command{
	Name:   "config",
	Usage:  "Print local configuration of the taxi CLI",
	Action: configAction,
	Subcommands: []*cli.Command{
		{
			Name:   "set",
			Usage:  "set a <key> <value> in the local state",
			Action: configSetAction,
		},
		{
			Name:   "init",
			Usage:  "initialize the local state with flags",
			Action: configInitAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  rpcServerKey,
					Usage: "taxi daemon address host:port",
					Value: "localhost:8000",
				},
				&cli.StringFlag{
					Name:  tlsCertPathKey,
					Usage: "the path of the TLS certificate file to use",
				},
				&cli.BoolFlag{
					Name:  insecureKey,
					Usage: "to connect to a daemon with TLS disabled",
					Value: false,
				},
			},
		},
	},
}

func configAction(ctx *cli.Context) error {
	state, err := getState()
	if err != nil {
		return err
	}

	for key, value := range state {
		fmt.Println(key + ": " + value)
	}

	return nil
}

func configInitAction(c *cli.Context) error {
	return setState(map[string]string{
		rpcServerKey:   c.String(rpcServerKey),
		tlsCertPathKey: cleanAndExpandPath(c.String(tlsCertPathKey)),
		insecureKey:    c.String(insecureKey),
	})
}

func configSetAction(c *cli.Context) error {
	if c.NArg() < 2 {
		return errors.New("key and value are missing")
	}

	key := c.Args().Get(0)
	value := c.Args().Get(1)

	partialState := map[string]string{key: value}

	if key == insecureKey {
		if val, _ := strconv.ParseBool(value); val {
			partialState[tlsCertPathKey] = ""
		}
	}
	if key == tlsCertPathKey {
		partialState[insecureKey] = "false"
	}

	if err := setState(partialState); err != nil {
		return err
	}

	fmt.Printf("%s %s has been set\n", key, value)

	return nil
}
