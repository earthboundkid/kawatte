package replaceall

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/fs"
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
		app.errorLevel.Println(err)
	}
	return err
}

type appEnv struct {
	patFile    string
	dir        string
	incFile    []string
	exFile     []string
	incDir     []string
	exDir      []string
	dryRun     bool
	infoLevel  *log.Logger
	warnLevel  *log.Logger
	errorLevel *log.Logger
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

	app.warnLevel = log.New(os.Stderr, AppName+" [WARNING] ", log.LstdFlags|log.Lmsgprefix)
	app.errorLevel = log.New(os.Stderr, AppName+" [ERROR] ", log.LstdFlags|log.Lmsgprefix)
	app.infoLevel = log.New(io.Discard, AppName+" [INFO] ", log.LstdFlags|log.Lmsgprefix)
	flagx.BoolFunc(fl, "verbose", "log debug output", func() error {
		app.infoLevel.SetOutput(os.Stderr)
		return nil
	})
	fl.Usage = func() {
		fmt.Fprintf(fl.Output(), `kawatte - %s

Kawatte recursively walks the file tree and finds and replaces the patterns
found in a substitution file. The substitution file is a CSV file of
old,new substitutions.

Example:

-- subs.csv --
a,b
b,c
c,a
-- in.txt --
abcdef

kawatte -pat subs.csv -match '*.txt'

-- in.txt --
bcadef

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

	if len(records) == 0 {
		app.warnLevel.Print("found no substitutions")
	} else {
		app.infoLevel.Printf("found %d substitutions", len(records))
	}

	replacements := slices.Grow[[]string](nil, len(records)*2)
	for _, record := range records {
		replacements = append(replacements, record[0], record[1])
	}

	return strings.NewReplacer(replacements...), nil
}

func (app *appEnv) walkDir() []string {
	var paths []string
	_ = filepath.WalkDir(app.dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			app.warnLevel.Printf("walking directories: %v", err)
			return nil
		}
		if entry.IsDir() {
			if path == "." {
				return nil
			}
			for _, glob := range app.exDir {
				if matched, _ := filepath.Match(glob, entry.Name()); matched {
					app.infoLevel.Printf("exclude dir %q", path)
					return filepath.SkipDir
				}
			}

			for _, glob := range app.incDir {
				if matched, _ := filepath.Match(glob, entry.Name()); matched {
					app.infoLevel.Printf("match for dir %q", path)
					return nil
				}
			}
			app.infoLevel.Printf("no match for dir %q", path)
			return filepath.SkipDir
		}
		for _, glob := range app.exFile {
			if matched, _ := filepath.Match(glob, entry.Name()); matched {
				app.infoLevel.Printf("exclude for %q", path)
				return nil
			}
		}

		for _, glob := range app.incFile {
			if matched, _ := filepath.Match(glob, entry.Name()); matched {
				app.infoLevel.Printf("match for %q", path)
				paths = append(paths, path)
				return nil
			}
		}
		app.infoLevel.Printf("no match for %q", path)
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

	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return fmt.Errorf("processFile(%q): writing: %w", filePath, err)
	}

	return nil
}
