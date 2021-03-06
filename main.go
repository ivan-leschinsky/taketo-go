package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"errors"
	"github.com/fatih/color"
)

func exit(err error) {
	color.Set(color.FgRed)
	log.Fatalln(err)
	os.Exit(1)
}

func parseArguments() (string, string) {
	if (len(os.Args) < 2) {
		exit(errors.New("Expected at least one argument as server"))
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
	log.SetFlags(0)

	serverAlias, overrideCommand := parseArguments()

	cfg, err := readConf(fmt.Sprintf("%s/.taketo.yml", os.Getenv("HOME")), serverAlias, overrideCommand)
	if err != nil {
		exit(err)
	}

	args := []string{fmt.Sprintf("%s@%s", cfg.User, cfg.Host)}
	if len(cfg.Command) > 0 {
		args = append(args, "-t")
		args = append(args, cfg.Command)
	}

	cmd := exec.Command("ssh", args...)

	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	_ = cmd.Run() // TODO: add error checking
}
