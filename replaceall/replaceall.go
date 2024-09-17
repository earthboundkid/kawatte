package replaceall

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/carlmjohnson/flagx"
	"github.com/carlmjohnson/flagx/lazyio"
	"github.com/carlmjohnson/versioninfo"
)

const AppName = "Kawatte"

func CLI(args []string) error {
	var app appEnv
	err := app.ParseArgs(args)
	if err != nil {
		return err
	}
	if err = app.Exec(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	return err
}

func (app *appEnv) ParseArgs(args []string) error {
	fl := flag.NewFlagSet(AppName, flag.ContinueOnError)
	src := lazyio.FileOrURL(lazyio.StdIO, nil)
	app.src = src
	fl.Var(src, "src", "source file or URL")
	app.Logger = log.New(io.Discard, AppName+" ", log.LstdFlags)
	flagx.BoolFunc(fl, "verbose", "log debug output", func() error {
		app.Logger.SetOutput(os.Stderr)
		return nil
	})
	fl.Usage = func() {
		fmt.Fprintf(fl.Output(), `kawatte - %s

Recursive find and replace

Usage:

	kawatte [options]

Options:
`, versioninfo.Version)
		fl.PrintDefaults()
	}
	if err := fl.Parse(args); err != nil {
		return err
	}
	if err := flagx.ParseEnv(fl, AppName); err != nil {
		return err
	}
	return nil
}

type appEnv struct {
	src io.ReadCloser
	*log.Logger
}

func (app *appEnv) Exec() (err error) {
	app.Println("starting")
	defer func() { app.Println("done") }()

	n, err := io.Copy(os.Stdout, app.src)
	defer func() {
		e2 := app.src.Close()
		if err == nil {
			err = e2
		}
	}()
	app.Printf("copied %d bytes\n", n)

	return err
}
