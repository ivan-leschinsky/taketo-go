package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	var overrideCommand string

	flag.StringVar(&overrideCommand, "c", "", "command to run on server")
	flag.Parse()

	cfg, err := readConf("./servers.yml", overrideCommand)
	if err != nil {
		log.Fatalln(err)
	}

	args := []string{fmt.Sprintf("%v@%v", cfg.User, cfg.Host)}
	if len(cfg.Command) > 0 {
		args = append(args, "-t")
		args = append(args, cfg.Command)
	}

	cmd := exec.Command("ssh", args...)

	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	_ = cmd.Run() // TODO: add error checking
}
