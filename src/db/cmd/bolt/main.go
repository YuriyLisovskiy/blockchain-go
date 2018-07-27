// Copyright (c) 2018 Yuriy Lisovskiy
// Distributed under the BSD 3-Clause software license, see the accompanying
// file LICENSE or https://opensource.org/licenses/BSD-3-Clause.

package db

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"
)

func main() {
	log.SetFlags(0)
	NewApp().Run(os.Args)
}

// NewApp creates an Application instance.
func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = "bolt"
	app.Usage = "BoltDB toolkit"
	app.Version = "0.1.0"
	app.Commands = []cli.Command{
		{
			Name:  "get",
			Usage: "Retrieve a value for given key in a bucket",
			Action: func(c *cli.Context) {
				path, name, key := c.Args().Get(0), c.Args().Get(1), c.Args().Get(2)
				Get(path, name, key)
			},
		},
		{
			Name:  "set",
			Usage: "Sets a value for given key in a bucket",
			Action: func(c *cli.Context) {
				path, name, key, value := c.Args().Get(0), c.Args().Get(1), c.Args().Get(2), c.Args().Get(3)
				Set(path, name, key, value)
			},
		},
		{
			Name:  "keys",
			Usage: "Retrieve a list of all keys in a bucket",
			Action: func(c *cli.Context) {
				path, name := c.Args().Get(0), c.Args().Get(1)
				Keys(path, name)
			},
		},
		{
			Name:  "buckets",
			Usage: "Retrieves a list of all buckets",
			Action: func(c *cli.Context) {
				path := c.Args().Get(0)
				Buckets(path)
			},
		},
		{
			Name:  "pages",
			Usage: "Dumps page information for a database",
			Action: func(c *cli.Context) {
				path := c.Args().Get(0)
				Pages(path)
			},
		},
		{
			Name:  "check",
			Usage: "Performs a consistency check on the database",
			Action: func(c *cli.Context) {
				path := c.Args().Get(0)
				Check(path)
			},
		},
	}
	return app
}

var logger = log.New(os.Stderr, "", 0)
var logBuffer *bytes.Buffer

func print(v ...interface{}) {
	if testMode {
		logger.Print(v...)
	} else {
		fmt.Print(v...)
	}
}

func printf(format string, v ...interface{}) {
	if testMode {
		logger.Printf(format, v...)
	} else {
		fmt.Printf(format, v...)
	}
}

func println(v ...interface{}) {
	if testMode {
		logger.Println(v...)
	} else {
		fmt.Println(v...)
	}
}

func fatal(v ...interface{}) {
	logger.Print(v...)
	if !testMode {
		os.Exit(1)
	}
}

func fatalf(format string, v ...interface{}) {
	logger.Printf(format, v...)
	if !testMode {
		os.Exit(1)
	}
}

func fatalln(v ...interface{}) {
	logger.Println(v...)
	if !testMode {
		os.Exit(1)
	}
}

// LogBuffer returns the contents of the log.
// This only works while the CLI is in test mode.
func LogBuffer() string {
	if logBuffer != nil {
		return logBuffer.String()
	}
	return ""
}

var testMode bool

// SetTestMode sets whether the CLI is running in test mode and resets the logger.
func SetTestMode(value bool) {
	testMode = value
	if testMode {
		logBuffer = bytes.NewBuffer(nil)
		logger = log.New(logBuffer, "", 0)
	} else {
		logger = log.New(os.Stderr, "", 0)
	}
}
