package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bpstr/rootmark/internal/config"
	"github.com/bpstr/rootmark/internal/site"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "rootmark: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "build":
		return runBuild(args[1:])
	case "version", "--version", "-v":
		fmt.Printf("rootmark %s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runBuild(args []string) error {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	configPath := flags.String("config", ".github/rootmark.yml", "configuration file")
	source := flags.String("source", "", "content source directory")
	output := flags.String("output", "", "generated site directory")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("build does not accept positional arguments")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if *source != "" {
		cfg.Build.Source = *source
	}
	if *output != "" {
		cfg.Build.Output = *output
	}

	result, err := site.Build(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("Built %d page(s) into %s\n", result.Pages, result.Output)
	return nil
}

func printUsage() {
	fmt.Print(`Rootmark turns Markdown repositories into static sites.

Usage:
  rootmark build [--config .github/rootmark.yml] [--source DIR] [--output DIR]
  rootmark version
`)
}
