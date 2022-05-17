package v1

import (
	"bufio"
	"fmt"
	"io"
	"log"

	"github.com/adwerx/trimdump/cli"
)

type Processor struct {
	Config *config
}

func NewProcessor() *Processor {
	config := NewConfig()
	return &Processor{Config: config}
}

func (p *Processor) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(p.Config); err != nil {
		return err
	}
	return nil
}

func (p *Processor) Run(opts cli.Options) error {
	log.Printf("DEBUG: %+v\n", p.Config)

	for _, database := range p.Config.Databases {
		dumper, err := NewDumper(opts.Binpath, fmt.Sprintf("%s:%s", opts.Host, opts.Port), opts.User, opts.Password)
		if err != nil {
			return err
		}

		for _, table := range database.IncludedTables {
			dumper.Database = database.Name
			dumper.Tables = []string{table.Name}
			dumper.ExtraOptions = opts.ExtraArgs

			r, w := io.Pipe()

			go func() {
				defer w.Close()
				dumper.Dump(w)
			}()

			scanner := bufio.NewScanner(r)

			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}

			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
		}
	}

	return nil
}
