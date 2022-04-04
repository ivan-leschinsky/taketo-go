package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func parseArguments() (string, string) {
	if (len(os.Args) < 2) {
		fmt.Println("Expected at least one argument as server")
		os.Exit(1)
	}
	var overrideCommand string
	var server string
	server = os.Args[1]

	if (len(os.Args) > 2) {
		mySet := flag.NewFlagSet("",flag.ExitOnError)
		mySet.StringVar(&overrideCommand, "c", "", "command to run on server")
		mySet.Parse(os.Args[2:])
	}
	return server, overrideCommand
}

func main() {
	server, overrideCommand := parseArguments()
	fmt.Println("server to run: ", server)

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
