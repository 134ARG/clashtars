package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"clashtars/internal"
)

const defaultConfigPath = "clash.conf"
const defaultTemplatePath = "template.yaml"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}

	switch args[0] {
	case "prepare":
		fs := flag.NewFlagSet("prepare", flag.ContinueOnError)
		configPath := fs.String("config", defaultConfigPath, "path to clash.conf")
		templatePath := fs.String("template", defaultTemplatePath, "path to template.yaml")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if err := internal.Prepare(context.Background(), *configPath, *templatePath); err != nil {
			fmt.Fprintf(os.Stderr, "clashtars prepare: %v\n", err)
			return 1
		}
		return 0
	case "start":
		fs := flag.NewFlagSet("start", flag.ContinueOnError)
		configPath := fs.String("config", defaultConfigPath, "path to clash.conf")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if err := internal.Start(*configPath); err != nil {
			fmt.Fprintf(os.Stderr, "clashtars start: %v\n", err)
			return 1
		}
		return 0
	default:
		usage()
		return 2
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: clashtars prepare [--config clash.conf] [--template template.yaml]")
	fmt.Fprintln(os.Stderr, "       clashtars start [--config clash.conf]")
}
