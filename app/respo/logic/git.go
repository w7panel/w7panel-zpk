package logic

import (
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/err_handler"
	"log"
	"os/exec"
)

type git struct {
	depot *Depot
}

func (self git) runCmd(arg ...string) (string, error) {
	cmd := exec.Command("git", arg...)
	cmd.Dir = self.depot.basePath
	msg, err := cmd.CombinedOutput()
	cmd.Run()
	if string(msg) != "" {
		log.Println(string(msg))
	}
	if err_handler.Found(err) {
		log.Println(err.Error())
	}
	return string(msg), err
}
