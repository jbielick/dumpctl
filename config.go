package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/pingcap/parser/ast"
)

type Config struct {
	Databases map[string]*Database
	Options   *Options
	File      *hcl.File
	Started   time.Time
	Conn      *sql.DB
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
	conn, err := NewConnection(opts)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	config = &Config{
		Databases: make(map[string]*Database),
		Options:   opts,
		File:      f,
		Started:   time.Now(),
		Conn:      conn,
	}

	moreDiags := config.Read()
	diags = append(diags, moreDiags...)

	var sb strings.Builder
	wr := hcl.NewDiagnosticTextWriter(&sb, parser.Files(), 78, true)
	wr.WriteDiagnostics(diags)
	if diags.HasErrors() {
		err = fmt.Errorf("Could not read config")
	}
	log.Println(sb.String())

	return
}

func NewConnection(opts *Options) (*sql.DB, error) {
	var dsn string
	if len(opts.Socket) != 0 {
		dsn = fmt.Sprintf(
			"%s:%s@unix(%s)/",
			opts.User,
			opts.Password,
			opts.Socket,
		)
	} else {
		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/",
			opts.User,
			opts.Password,
			opts.Host,
			opts.Port,
		)
	}
	return sql.Open("mysql", dsn)
}

func (c *Config) Read() (diags hcl.Diagnostics) {
	configContent, diags := c.File.Body.Content(configSchema)
	if diags.HasErrors() {
		return
	}
	for _, dbBlock := range configContent.Blocks {
		name := dbBlock.Labels[0]
		database, moreDiags := NewDatabase(name, dbBlock, c)
		c.Databases[name] = database
		if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
			continue
		}
		moreDiags = database.ReadSchema()
		if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
			continue
		}
		moreDiags = database.ReadDynamicConfig()
		if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
			continue
		}
	}

	return
}
