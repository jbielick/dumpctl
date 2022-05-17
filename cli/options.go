package cli

type Options struct {
	ConfigFile string `short:"c" long:"config" description:"Path to config file"`
	Host       string `short:"h" long:"host" description:"hostname of server" default:"127.0.0.1"`
	Port       string `short:"P" long:"port" description:"port of server"`
	User       string `short:"u" long:"user" description:"user for login"`
	Password   string `short:"p" long:"password" description:"password for login"`
	Binpath    string `long:"binpath" description:"Path to mysqldump" default:"mysqldump"`
	Verbose    []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	ExtraArgs  []string
}
