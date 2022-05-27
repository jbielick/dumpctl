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
	Port       string `short:"P" long:"port" description:"port of server"`
	User       string `short:"u" long:"user" description:"user for login"`
	Password   string `short:"p" long:"password" description:"password for login"`
	Binpath    string `long:"binpath" description:"Path to mysqldump" default:"mysqldump"`
	Verbose    []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	ExtraArgs  []string
}

var opts Options

func init() {
	log.SetFlags(0)
	extraArgs, err := flags.Parse(&opts)
	if err != nil {
		// PrintError defaults to true
		os.Exit(1)
	}
	opts.ExtraArgs = extraArgs
}

func main() {
	config, err := NewConfig(&opts)

	if err != nil {
		log.Fatal(err.Error())
	}

	sequencer, err := NewDumpSequencer(config)
	err = sequencer.Dump()
	if err != nil {
		log.Fatal(err.Error())
	}
}
