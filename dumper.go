package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Dumper struct {
	Binpath          string
	Addr             string
	User             string
	Password         string
	Tables           []string
	Database         string
	Destination      string
	Where            string
	Charset          string
	ExtraOptions     []string
	ErrOut           io.Writer
	maxAllowedPacket int
}

func NewDumper(options *Options) (*Dumper, error) {
	path, err := exec.LookPath(options.Binpath)
	if err != nil {
		return nil, err
	}

	d := new(Dumper)
	d.Binpath = path
	if len(options.Socket) != 0 {
		d.Addr = options.Socket
	} else {
		d.Addr = fmt.Sprintf("%s:%s", options.Host, options.Port)
	}
	d.User = options.User
	d.Password = options.Password
	d.Tables = make([]string, 0, 16)
	d.Charset = ""
	d.ExtraOptions = []string{}

	d.ErrOut = os.Stderr

	return d, nil
}

func (d *Dumper) SetCharset(charset string) {
	d.Charset = charset
}

func (d *Dumper) SetWhere(where string) {
	d.Where = where
}

func (d *Dumper) SetExtraOptions(options []string) {
	d.ExtraOptions = options
}

func (d *Dumper) SetErrOut(o io.Writer) {
	d.ErrOut = o
}

func (d *Dumper) SetMaxAllowedPacket(i int) {
	d.maxAllowedPacket = i
}

func (d *Dumper) SetDestinationDatabase(name string) {
	d.Destination = name
}

func (d *Dumper) AddTables(db string, tables ...string) {
	if d.Database != db {
		d.Database = db
		d.Tables = d.Tables[0:0]
	}

	d.Tables = append(d.Tables, tables...)
}

func (d *Dumper) Reset() {
	d.Tables = d.Tables[0:0]
	d.Database = ""
	d.Destination = ""
	d.Where = ""
}

func (d *Dumper) DestinationDatabase() string {
	if len(d.Destination) > 0 {
		return d.Destination
	} else {
		return d.Database
	}
}

func (d *Dumper) Dump(w io.Writer) error {
	args := make([]string, 0, 16)

	if strings.Contains(d.Addr, "/") {
		args = append(args, fmt.Sprintf("--socket=%s", d.Addr))
	} else {
		seps := strings.SplitN(d.Addr, ":", 2)
		args = append(args, fmt.Sprintf("--host=%s", seps[0]))
		if len(seps) > 1 {
			args = append(args, fmt.Sprintf("--port=%s", seps[1]))
		}
	}

	args = append(args, fmt.Sprintf("--user=%s", d.User))
	passwordArg := fmt.Sprintf("--password=%s", d.Password)
	args = append(args, passwordArg)
	passwordArgIndex := len(args) - 1

	if d.maxAllowedPacket > 0 {
		args = append(args, fmt.Sprintf("--max-allowed-packet=%dM", d.maxAllowedPacket))
	}

	args = append(args, "--single-transaction")
	args = append(args, "--compact")
	args = append(args, "--skip-lock-tables")
	args = append(args, "--skip-opt")
	args = append(args, "--quick")
	args = append(args, "--skip-extended-insert")
	args = append(args, "--tz-utc")
	args = append(args, "--hex-blob")
	args = append(args, "--add-drop-table")

	if len(d.Charset) != 0 {
		args = append(args, fmt.Sprintf("--default-character-set=%s", d.Charset))
	}

	if len(d.Where) != 0 {
		args = append(args, fmt.Sprintf("--where=%s", d.Where))
	}

	if len(d.ExtraOptions) != 0 {
		args = append(args, d.ExtraOptions...)
	}

	args = append(args, d.Database)
	args = append(args, d.Tables...)

	_, err := w.Write([]byte(fmt.Sprintf("USE `%s`;\n", d.DestinationDatabase())))
	if err != nil {
		return fmt.Errorf(`could not write USE command: %w`, err)
	}

	args[passwordArgIndex] = "--password=******"
	log.Printf("<- %s.%s %s\n", d.Database, d.Tables[0], d.Where)
	args[passwordArgIndex] = passwordArg
	cmd := exec.Command(d.Binpath, args...)

	cmd.Stderr = d.ErrOut
	cmd.Stdout = w

	return cmd.Run()
}
