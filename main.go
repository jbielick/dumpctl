package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	cli "github.com/adwerx/trimdump/cli"
	v1 "github.com/adwerx/trimdump/v1"
	hclsimple "github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
)

type Processor interface {
	Run(cli.Options) error
}

type Api struct {
	Version   string `yaml:"apiVersion"`
	Processor Processor
}

var versionReader struct {
	Version string `yaml:"apiVersion" hcl:"version"`
}

func (api *Api) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var versionReader struct {
		Version string `yaml:"apiVersion"`
	}
	if err := unmarshal(&versionReader); err != nil {
		return err
	}
	if len(versionReader.Version) > 0 {
		api.Version = versionReader.Version
	} else {
		api.Version = "v1"
	}
	versionMap := map[string]Processor{
		"v1": v1.NewProcessor(),
	}
	processor := versionMap[api.Version]
	if err := unmarshal(processor); err != nil {
		return err
	}
	api.Processor = processor
	return nil
}

func main() {
	var opts cli.Options
	extraArgs, err := flags.Parse(&opts)
	opts.ExtraArgs = extraArgs
	if err != nil {
		log.Fatalf("failed to parse options: %v\n", err)
	}
	var api = new(Api)
	if len(opts.ConfigFile) > 0 {
		if strings.HasSuffix(opts.ConfigFile, ".hcl") {
			err := hclsimple.DecodeFile(opts.ConfigFile, nil, api)
			if err != nil {
				log.Fatalf("Failed to load configuration: %s", err)
			}
			log.Printf("DEBUG: %+v\n", api)
			os.Exit(0)
		} else {
			data, err := os.ReadFile(opts.ConfigFile)
			if err != nil {
				log.Fatalf("Failed to read config: %v\n", err)
			}
			if err := yaml.Unmarshal(data, &api); err != nil {
				log.Fatal(err)
			}
		}
	}
	err = api.Processor.Run(opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%v\n", api)
	os.Exit(0)

	// var in io.Reader

	// info, err := os.Stdin.Stat()
	// if err != nil {
	// 	log.Fatalf("ERROR: %v\n", err)
	// }
	// if info.Size() > 0 {
	// 	in = os.Stdin
	// } else {
	// 	in, err = NewDumpReader(dumpExtraArgs)
	// 	if err != nil {
	// 		log.Fatalf("ERROR: failed to create dump: %v\n", err)
	// 	}
	// }

	// scanner := bufio.NewScanner(in)

	// for scanner.Scan() {
	// 	fmt.Println(scanner.Text())
	// }

	// if err := scanner.Err(); err != nil {
	// 	log.Fatal(err)
	// }
}
