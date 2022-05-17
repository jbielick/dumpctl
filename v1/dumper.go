package v1

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Unlick mysqldump, Dumper is designed for parsing and syning data easily.
type Dumper struct {
	ExecutionPath    string
	Addr             string
	User             string
	Password         string
	Tables           []string
	Database         string
	Where            string
	Charset          string
	IgnoreTables     []string
	ExtraOptions     []string
	ErrOut           io.Writer
	maxAllowedPacket int
}

func NewDumper(executionPath string, addr string, user string, password string) (*Dumper, error) {
	if len(executionPath) == 0 {
		return nil, nil
	}

	path, err := exec.LookPath(executionPath)
	if err != nil {
		return nil, err
	}

	d := new(Dumper)
	d.ExecutionPath = path
	d.Addr = addr
	d.User = user
	d.Password = password
	d.Tables = make([]string, 0, 16)
	d.Charset = "utf8mb4"
	d.IgnoreTables = []string{}
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

func (d *Dumper) AddTables(db string, tables ...string) {
	if d.Database != db {
		d.Database = db
		d.Tables = d.Tables[0:0]
	}

	d.Tables = append(d.Tables, tables...)
}

func (d *Dumper) AddIgnoreTables(db string, tables ...string) {
	d.IgnoreTables = append(d.IgnoreTables, tables...)
}

func (d *Dumper) Reset() {
	d.Tables = d.Tables[0:0]
	d.Database = ""
	d.IgnoreTables = []string{}
	d.Where = ""
}

func (d *Dumper) Dump(w io.Writer) error {
	args := make([]string, 0, 16)

	// Common args
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
	// args = append(args, "--complete-insert")
	args = append(args, "--compact")
	args = append(args, "--skip-lock-tables")
	args = append(args, "--skip-add-drop-table")
	args = append(args, "--no-create-info")
	args = append(args, "--skip-opt")
	args = append(args, "--quick")
	// args = append(args, "--skip-extended-insert")
	args = append(args, "--tz-utc")
	args = append(args, "--hex-blob")

	for db, tables := range d.IgnoreTables {
		for _, table := range tables {
			args = append(args, fmt.Sprintf("--ignore-table=%s.%s", db, table))
		}
	}

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

	_, err := w.Write([]byte(fmt.Sprintf("USE `%s`;\n", d.Database)))
	if err != nil {
		return fmt.Errorf(`could not write USE command: %w`, err)
	}

	args[passwordArgIndex] = "--password=******"
	// log.Infof("exec mysqldump with %v", args)
	args[passwordArgIndex] = passwordArg
	cmd := exec.Command(d.ExecutionPath, args...)

	cmd.Stderr = d.ErrOut
	cmd.Stdout = w

	return cmd.Run()
}

// DumpAndParse: Dump MySQL and parse immediately
// func (d *Dumper) DumpAndParse(h dump.ParseHandler) error {
// 	r, w := io.Pipe()

// 	done := make(chan error, 1)
// 	go func() {
// 		err := dump.Parse(r, h, false)
// 		_ = r.CloseWithError(err)
// 		done <- err
// 	}()

// 	err := d.Dump(w)
// 	_ = w.CloseWithError(err)

// 	err = <-done

// 	return errors.Trace(err)
// }
