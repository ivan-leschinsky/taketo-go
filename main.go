package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	cfg, err := readConf("./servers.yml")
	if err != nil {
		log.Fatalln(err)
	}

	args := []string{fmt.Sprintf("%v@%v", cfg.User, cfg.Host)}
	if len(cfg.Command) > 0 {
		args = append(args, fmt.Sprintf("-t %s", cfg.Command))
	}

	cmd := exec.Command("ssh", args...)

	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	_ = cmd.Run() // TODO: add error checking
}
