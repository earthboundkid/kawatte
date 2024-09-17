package replaceall

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/carlmjohnson/flagx"
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

type appEnv struct {
	patFile string
	dir     string
	incFile []string
	exFile  []string
	incDir  []string
	exDir   []string
	dryRun  bool
	*log.Logger
}

func (app *appEnv) ParseArgs(args []string) error {
	fl := flag.NewFlagSet(AppName, flag.ContinueOnError)

	fl.StringVar(&app.patFile, "pat", "", "path to the CSV `file` containing substitution patterns")
	fl.StringVar(&app.dir, "dir", ".", "path to the starting `directory`")
	fl.BoolVar(&app.dryRun, "dry-run", false, "just print the names of files that would be modified")

	fl.Func("match", "`glob` matching files to include (default *)", func(glob string) error {
		app.incFile = append(app.incFile, glob)
		return nil
	})
	fl.Func("exclude", "`glob` matching files to exclude (default .*)", func(glob string) error {
		app.incFile = append(app.exFile, glob)
		return nil
	})
	fl.Func("match-dir", "`glob` matching directories to include (default *)", func(glob string) error {
		app.incFile = append(app.incFile, glob)
		return nil
	})
	fl.Func("exclude-dir", "`glob` matching directories to exclude (default .*)", func(glob string) error {
		app.incFile = append(app.exFile, glob)
		return nil
	})

	app.Logger = log.New(io.Discard, AppName+" ", log.LstdFlags)
	flagx.BoolFunc(fl, "verbose", "log debug output", func() error {
		app.Logger.SetOutput(os.Stderr)
		return nil
	})
	fl.Usage = func() {
		fmt.Fprintf(fl.Output(), `kawatte - %s

Kawatte recursively walks the file tree and finds and replaces the patterns found in the substitution file.

Usage:

	kawatte [options]

Options:
`, versioninfo.Version)
		fl.PrintDefaults()
	}
	versioninfo.AddFlag(fl)
	if err := fl.Parse(args); err != nil {
		return err
	}
	if err := flagx.ParseEnv(fl, AppName); err != nil {
		return err
	}
	if err := flagx.MustHave(fl, "pat"); err != nil {
		return err
	}

	if len(app.incFile) == 0 {
		app.incFile = []string{"*"}
	}
	if len(app.exFile) == 0 {
		app.exFile = []string{".*"}
	}
	if len(app.incDir) == 0 {
		app.incDir = []string{"*"}
	}
	if len(app.exDir) == 0 {
		app.exDir = []string{".*"}
	}
	return nil
}

func (app *appEnv) Exec() (err error) {
	replacer, err := app.loadSubstitutions()
	if err != nil {
		return err
	}

	paths := app.walkDir()
	for _, path := range paths {
		if err := app.processFile(path, replacer); err != nil {
			return err
		}
	}
	return nil
}

func (app *appEnv) loadSubstitutions() (*strings.Replacer, error) {
	file, err := os.Open(app.patFile)
	if err != nil {
		return nil, fmt.Errorf("opening substitution patterns file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 2
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading substitution patterns file %q: %w", app.patFile, err)
	}

	app.Printf("found %d substitutions", len(records))

	replacements := slices.Grow[[]string](nil, len(records)*2)
	for _, record := range records {
		replacements = append(replacements, record[0], record[1])
	}

	return strings.NewReplacer(replacements...), nil
}

func (app *appEnv) walkDir() []string {
	var paths []string
	_ = filepath.Walk(app.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("warning: error walking directories: %v\n", err)
			return nil
		}
		if info.IsDir() {
			if path == "." {
				return nil
			}
			for _, glob := range app.exDir {
				if matched, _ := filepath.Match(glob, info.Name()); matched {
					app.Printf("exclude dir %q", path)
					return filepath.SkipDir
				}
			}

			for _, glob := range app.incDir {
				if matched, _ := filepath.Match(glob, info.Name()); matched {
					app.Printf("include dir %q", path)
					return nil
				}
			}
			app.Printf("no match for dir %q", path)
			return filepath.SkipDir
		}
		for _, glob := range app.exFile {
			if matched, _ := filepath.Match(glob, info.Name()); matched {
				app.Printf("exclude file %q", path)
				return nil
			}
		}

		for _, glob := range app.incFile {
			if matched, _ := filepath.Match(glob, info.Name()); matched {
				app.Printf("include file %q", path)
				paths = append(paths, path)
				return nil
			}
		}
		app.Printf("no match for file %q", path)
		return nil
	})
	return paths
}

func (app *appEnv) processFile(filePath string, replacer *strings.Replacer) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("processFile(%q): reading: %w", filePath, err)
	}

	oldContent := string(b)
	newContent := replacer.Replace(oldContent)

	if app.dryRun {
		if oldContent != newContent {
			fmt.Printf("* %q\n", filePath)
		}
		return nil
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("processFile(%q): stating: %w", filePath, err)
	}

	err = os.WriteFile(filePath, []byte(newContent), info.Mode())
	if err != nil {
		return fmt.Errorf("processFile(%q): writing: %w", filePath, err)
	}

	return nil
}
