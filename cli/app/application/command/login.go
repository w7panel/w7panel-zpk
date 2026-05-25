package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/cli/app/application/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
)

type Login struct {
	console.Abstract
}

func (c Login) GetName() string {
	return "login"
}

func (c Login) GetDescription() string {
	return "login to zpk registry"
}

func (c Login) Configure(cmd *cobra.Command) {
	cmd.Flags().String("host", "", "registry host, for example registry.example.com")
	cmd.Flags().StringP("username", "u", "", "registry username")
	cmd.Flags().StringP("password", "p", "", "registry password")
	_ = cmd.MarkFlagRequired("host")
	_ = cmd.MarkFlagRequired("username")
	_ = cmd.MarkFlagRequired("password")
}

func (c Login) Handle(cmd *cobra.Command, args []string) {
	host, _ := cmd.Flags().GetString("host")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")

	host = strings.TrimRight(strings.TrimSpace(host), "/")
	username = strings.TrimSpace(username)
	if host == "" || username == "" || password == "" {
		panic("host, username and password are required")
	}
	if strings.Contains(host, "://") || strings.Contains(host, "/") {
		panic("host must be a domain or host:port without protocol")
	}
	encryptPwd, err := logic.EncryptPassword(password)
	if err != nil {
		panic(err)
	}

	session, err := logic.LoadSession()
	if err != nil {
		panic(err)
	}
	session.Host = host
	session.Username = username
	session.Password = encryptPwd
	session.Artifact = ""
	if err := logic.SaveSession(session); err != nil {
		panic(err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "login saved for %s\n", host)
}
