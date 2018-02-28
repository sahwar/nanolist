package config

import (
	"database/sql"
	"io"
	"log"
	"net/smtp"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/ini.v1"
)

type Config struct {
	CommandAddress string `ini:"command_address"`
	Log            string `ini:"log"`
	Database       string `ini:"database"`
	SMTPHostname   string `ini:"smtp_hostname"`
	SMTPPort       string `ini:"smtp_port"`
	SMTPUsername   string `ini:"smtp_username"`
	SMTPPassword   string `ini:"smtp_password"`
	Lists          map[string]*List
}

type List struct {
	Name            string `ini:"name"`
	Description     string `ini:"description"`
	Id              string
	Address         string   `ini:"address"`
	Hidden          bool     `ini:"hidden"`
	SubscribersOnly bool     `ini:"subscribers_only"`
	Posters         []string `ini:"posters,omitempty"`
	Bcc             []string `ini:"bcc,omitempty"`
}

// Check for a valid configuration
func (config *Config) Check() error {
	if db, err := config.OpenDB(); err != nil {
		return err
	} else {
		defer db.Close()
	}

	if log, err := config.OpenLog(); err != nil {
		return err
	} else {
		defer log.Close()
	}

	if client, err := smtp.Dial(
		config.SMTPHostname + ":" + config.SMTPPort,
	); err != nil {
		return err
	} else {
		defer client.Close()
		auth := smtp.PlainAuth("",
			config.SMTPUsername,
			config.SMTPPassword,
			config.SMTPHostname)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	return nil
}

// Load gConfig from the on-disk config file
func Load(path *string) (*Config, error) {
	var (
		err  error
		file *ini.File
		cfg  *Config
	)

	if path != nil {
		file, err = ini.Load(path)
	} else {
		file, err = ini.LooseLoad(
			"./nanolist.ini",
			"/usr/local/etc/nanolist.ini",
			"/etc/nanolist.ini",
		)
	}

	if err != nil {
		return nil, err
	}

	cfg = &Config{}

	if err = file.Section("").MapTo(cfg); err != nil {
		return nil, err
	}

	cfg.Lists = make(map[string]*List)

	for _, section := range file.ChildSections("list") {
		list := &List{}
		err = section.MapTo(list)
		if err != nil {
			return nil, err
		}
		list.Id = strings.TrimPrefix(section.Name(), "list.")
		cfg.Lists[list.Address] = list
	}

	return cfg, nil
}

func (cfg *Config) OpenLog() (io.Closer, error) {
	file, err := os.OpenFile(
		cfg.Log, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	out := io.MultiWriter(file, os.Stderr)
	log.SetOutput(out)
	return file, nil
}

func (cfg *Config) OpenDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", cfg.Database)

	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS "subscriptions" (
		"list" TEXT,
		"user" TEXT
	);
	`)

	return db, err
}
