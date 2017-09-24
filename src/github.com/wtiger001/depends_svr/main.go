package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/howeyc/gopass"
	"github.com/wtiger001/depends_svr/db"
)

func main() {
	cfg := getConfig()

	db.ExtractData(cfg)

	fmt.Printf("Complete\n")
}

// Load the configuration from file, from the command line, etc.
func getConfig() *db.JiraConfig {
	var cfgFile string
	var cfg *db.JiraConfig

	// Loads froma configuration file. Any other variables will override
	flag.StringVar(&cfgFile, "cfg", "", "Configuration File")
	if cfgFile != "" && !exists(cfgFile) {
		cfgFile = "config.json"
	}

	// Load the configuration
	cfg = new(db.JiraConfig)
	if exists(cfgFile) {
		err := cfg.Load(cfgFile)
		if err != nil {
			os.Exit(1)
		}
	}
	cfg.ApplyDefaults()

	// Load any command line overrides
	flag.StringVar(&cfg.User, "user", cfg.User, "JIRA User Name")
	flag.StringVar(&cfg.Password, "password", cfg.Password, "JIRA Password")
	flag.StringVar(&cfg.JiraURL, "url", cfg.JiraURL, "JIRA URL")
	flag.BoolVar(&cfg.Debug, "debug", cfg.Debug, "Enable Debuging mode")
	flag.StringVar(&cfg.OutputFile, "out", cfg.OutputFile, "Output File")
	flag.Parse()

	// Validate and ask for missing fields from the command line
	if !cfg.Valid() {
		readFromTerminal(cfg)
	}

	if cfg.Debug {
		cfg.Print()
	}

	return cfg
}

// Reads inputs from
func readFromTerminal(cfg *db.JiraConfig) {
	reader := bufio.NewReader(os.Stdin)
	if cfg.User == "" {
		fmt.Print("Enter JIRA Username: ")
		user, _ := reader.ReadString('\n')
		cfg.User = user
	}
	if cfg.Password == "" {
		fmt.Print("Enter JIRA Password: ")
		pass, _ := gopass.GetPasswdMasked()
		cfg.Password = string(pass)
	}
	if cfg.JiraURL == "" {
		fmt.Print("Enter JIRA URL: ")
		url, _ := reader.ReadString('\n')
		cfg.JiraURL = url
	}

}

func exists(file string) bool {
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		return true
	}
	return false
}
