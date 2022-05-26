package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/pingcap/parser/ast"
)

type Config struct {
	Databases map[string]*Database
	Options   *Options
	File      *hcl.File
}

var configSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "database",
			LabelNames: []string{"name"},
		},
	},
}

type Column struct {
	Name      string
	Position  int64
	Type      string
	MaxLength sql.NullInt64
	Table     *Table
}

func (c *Column) String() string {
	return fmt.Sprintf(c.Table.Name, c.Name)
}

type Row struct {
	Table  *Table
	Values *[]ast.ExprNode
}

func NewConfig(opts *Options) (config *Config, err error) {
	parser := hclparse.NewParser()
	f, diags := parser.ParseHCLFile(opts.ConfigFile)

	config = &Config{
		Databases: make(map[string]*Database),
		Options:   opts,
		File:      f,
	}

	moreDiags := config.Read()
	diags = append(diags, moreDiags...)

	if diags.HasErrors() {
		var sb strings.Builder
		wr := hcl.NewDiagnosticTextWriter(&sb, parser.Files(), 78, true)
		wr.WriteDiagnostics(diags)
		err = fmt.Errorf("Some errors were encountered reading the config: \n%s", sb.String())
	}
	return
}

func (c *Config) Read() (diags hcl.Diagnostics) {
	configContent, diags := c.File.Body.Content(configSchema)
	if diags.HasErrors() {
		return
	}
	for _, dbBlock := range configContent.Blocks {
		database, moreDiags := c.AddDatabase(dbBlock.Labels[0], dbBlock)
		if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
			continue
		}
		moreDiags = database.ReadSchema()
		if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
			continue
		}
		moreDiags = database.ReadRules()
		if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
			continue
		}
	}
	return
}

// make some of this NewDatabase
func (c *Config) AddDatabase(name string, block *hcl.Block) (db *Database, diags hcl.Diagnostics) {
	db, diags = NewDatabase(name, block, c)
	c.Databases[name] = db
	return
}
