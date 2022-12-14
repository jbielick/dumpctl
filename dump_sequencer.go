package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/heimdalr/dag"
	"github.com/pingcap/tidb/parser"
	_ "github.com/pingcap/tidb/types/parser_driver"
)

type DumpSequencer struct {
	Config *Config
}

func NewDumpSequencer(config *Config) *DumpSequencer {
	return &DumpSequencer{Config: config}
}

func (s *DumpSequencer) Dump() error {
	for _, database := range s.Config.Databases {
		if err := s.DumpDatabase(database); err != nil {
			return err
		}
	}
	return nil
}

func (s *DumpSequencer) DumpDatabase(database *Database) error {
	dumper, err := NewDumper(s.Config.Options)
	if err != nil {
		return err
	}
	for _, table := range database.Tables {
		outFile, err := ioutil.TempFile("", table.Name)
		if err != nil {
			log.Fatal(err)
		}
		table.OutFile = outFile
		defer os.Remove(table.OutFile.Name())
	}
	var diags hcl.Diagnostics
	visitor := &Rewriter{
		Database:    database,
		Dumper:      dumper,
		Parser:      parser.New(),
		Diagnostics: diags,
		Wg:          &sync.WaitGroup{},
	}

	dag.BFSWalk(database.DAG, visitor)
	var sb strings.Builder
	wr := hcl.NewDiagnosticTextWriter(&sb, nil, 78, true)
	wr.WriteDiagnostics(visitor.Diagnostics)
	if diags.HasErrors() {
		err = fmt.Errorf("Could not read config")
	}
	log.Println(sb.String())
	return err
}
