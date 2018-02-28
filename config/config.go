package config

import (
	"crypto/tls"
	"database/sql"
	"io"
	"log"
	"net/smtp"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

type Config struct {
	CommandAddress string `ini:"command_address"`
	Log            string `ini:"log"`
	Database       string `ini:"database"`
	SMTPHostname   string `ini:"smtp_hostname"`
	SMTPPort       string `ini:"smtp_port"`
	SMTPTLS        bool   `ini:"smtp_starttls"`
	SMTPTLSVerify  bool   `ini:"smtp_tlsverify"`
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

	if client, err := config.OpenSMTP(); err != nil {
		return err
	} else {
		defer client.Close()
	}

	return nil
}

// Load config from an on-disk config file
func Load(path *string) (*Config, error) {
	var (
		err    error
		file   *ini.File
		config *Config
	)

	config = &Config{
		SMTPTLSVerify: true,
	}

	if path != nil && *path != "" {
		file, err = ini.Load(path)
	} else {
		file, err = ini.LooseLoad(
			"./nanolist.ini",
			"/usr/local/etc/nanolist.ini",
			"/etc/nanolist.ini",
		)
	}

	if err != nil {
		return nil, errors.Wrap(err, "Could not open config file for reading")
	}

	config = &Config{}

	if err = file.Section("").MapTo(config); err != nil {
		return nil, err
	}

	config.Lists = make(map[string]*List)

	for _, section := range file.ChildSections("list") {
		list := &List{}
		err = section.MapTo(list)
		if err != nil {
			return nil, err
		}
		list.Id = strings.TrimPrefix(section.Name(), "list.")
		config.Lists[list.Address] = list
	}

	return config, nil
}

func (config *Config) OpenLog() (io.Closer, error) {
	file, err := os.OpenFile(
		config.Log, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "could not open log")
	}
	out := io.MultiWriter(file, os.Stderr)
	log.SetOutput(out)
	return file, nil
}

func (config *Config) OpenDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", config.Database)

	if err != nil {
		return nil, errors.Wrap(err, "could not open database")
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS "subscriptions" (
		"list" TEXT,
		"user" TEXT
	);
	`)

	return db, err
}

func (config *Config) OpenSMTP() (*smtp.Client, error) {
	client, err := smtp.Dial(config.SMTPHostname + ":" + config.SMTPPort)
	if err != nil {
		return nil, errors.Wrap(err, "could not dial SMTP server")
	}
	if config.SMTPTLS {
		var tlsConfig *tls.Config
		if !config.SMTPTLSVerify {
			tlsConfig = &tls.Config{InsecureSkipVerify: true}
		}
		client.StartTLS(tlsConfig)
	}
	auth := smtp.PlainAuth("",
		config.SMTPUsername,
		config.SMTPPassword,
		config.SMTPHostname)
	if err := client.Auth(auth); err != nil {
		return nil, errors.Wrap(err,
			"could not authenticate to SMTP server")
	}
	return client, nil
}
