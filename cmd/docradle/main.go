package main

import (
	"fmt"
	"os"

	"github.com/future-architect/docradle"
	"github.com/gookit/color"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	runCommand  = kingpin.Command("run", "Execute commands")
	configFlag  = runCommand.Flag("config", "Config filename").Default(`docradle.cue,docradle.json,docradle.yaml,docradle.yml`).Short('c').String()
	dryRunFlag  = runCommand.Flag("dryrun", "Check EnvVar/Files only").Short('d').Bool()
	dotEnvFlag  = runCommand.Flag("dotenv", ".env filename").Short('e').Default(".env").String()
	command     = runCommand.Arg("command", "Command name to run").Required().String()
	args        = runCommand.Arg("args", "Arguments").Strings()
	initCommand = kingpin.Command("init", "Generate config file")
	format      = initCommand.Flag("format", "Config file format").Short('f').Default("json").Enum("cue", "json", "yaml")
)

func main() {
	color.IsSupportColor()
	switch kingpin.Parse() {
	case runCommand.FullCommand():
		wd, err := os.Getwd()
		if err != nil {
			color.Fprintf(os.Stderr, "<red>Cannot get current folder: %v</>\n", err)
			os.Exit(1)
		}
		config, envvar, err := docradle.ParseAndVerifyConfig(wd, os.Stdout, os.Stderr, *configFlag, *dotEnvFlag)
		if err != nil {
			os.Exit(1)
		}
		docradle.DumpCommand(os.Stdout, *command, *args, *dryRunFlag)
		if !(*dryRunFlag) {
			err = docradle.Exec(os.Stdout, os.Stderr, config, *command, *args, envvar)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fails to run command: %v", err)
				os.Exit(1)
			}
		}
	case initCommand.FullCommand():
		err := docradle.Generate(os.Stdout, *format)
		if err != nil {
			color.Fprintf(os.Stderr, "<red>%s</>\n", err.Error())
			os.Exit(1)
		}
	}
}
