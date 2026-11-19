package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	clicommon "github.com/w7panel/w7panel-zpk/cli/app/application/logic/common"
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
	encryptPwd, err := clicommon.EncryptPassword(password)
	if err != nil {
		panic(err)
	}

	session, err := clicommon.LoadSession()
	if err != nil {
		panic(err)
	}
	if session.Credentials == nil {
		session.Credentials = map[string]clicommon.RegistryCredential{}
	}
	if session.Host != "" && session.Username != "" && session.Password != "" {
		session.Credentials[session.Host] = clicommon.RegistryCredential{Username: session.Username, Password: session.Password}
	}
	session.Credentials[host] = clicommon.RegistryCredential{Username: username, Password: encryptPwd}
	session.Host = host
	session.Username = username
	session.Password = encryptPwd
	if err := clicommon.SaveSession(session); err != nil {
		panic(err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "login saved for %s\n", host)
}
