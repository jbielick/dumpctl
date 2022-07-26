package main

import (
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	ConfigFile string `short:"c" long:"config" description:"Path to config file" required:"true"`
	Host       string `short:"h" long:"host" description:"hostname of server" default:"127.0.0.1"`
	Port       string `short:"P" long:"port" description:"port of server" default:"3306"`
	Socket     string `short:"S" long:"socket"`
	User       string `short:"u" long:"user" description:"user for login"`
	Password   string `short:"p" long:"password" description:"password for login"`
	Binpath    string `long:"binpath" description:"Path to mysqldump" default:"mysqldump"`
	Help       bool   `long:"help" description:"Display this (help) message"`
	Verbose    []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	ExtraArgs  []string
}

var opts Options

func init() {
	log.SetFlags(0)
}

func main() {
	parser := flags.NewParser(&opts, flags.PassDoubleDash)
	extraArgs, err := parser.Parse()

	if opts.Help {
		parser.WriteHelp(os.Stderr)
		os.Exit(0)
	}

	if err != nil {
		log.Printf("Error: %s\n\n", err.Error())
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}
	opts.ExtraArgs = extraArgs

	log.Printf("DEBUG: reading config")

	config, err := NewConfig(&opts)

	if err != nil {
		log.Fatal(err.Error())
	}

	sequencer := NewDumpSequencer(config)
	log.Printf("DEBUG: starting dump")
	err = sequencer.Dump()
	if err != nil {
		log.Fatal(err.Error())
	}
}
