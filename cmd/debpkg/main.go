package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cjey/debpkg"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("debpkg: ")
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

// run returns instead of exiting, so that the deferred Close removes the intermediate files
func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("debpkg", flag.ExitOnError)
	configFile := flags.String("c", "debpkg.yml", "YAML configuration file")
	outputFile := flags.String("o", "", "Debian output file (default <name>_<version>_<architecture>.deb)")
	version := flags.String("v", os.Getenv("DEBPKG_VERSION"),
		"Package version number (or via DEBPKG_VERSION environment variable)")
	flags.Parse(args)
	// Otherwise "debpkg foo.yml" would silently build debpkg.yml
	if flags.NArg() > 0 {
		return fmt.Errorf("unexpected argument %s, the specfile is given by -c", flags.Arg(0))
	}

	deb := debpkg.New()
	defer deb.Close()
	if err := deb.Config(*configFile); err != nil {
		return err
	}
	if *version != "" {
		deb.SetVersion(*version)
	}
	if err := deb.Write(*outputFile); err != nil {
		return fmt.Errorf("write package: %w", err)
	}
	out := *outputFile
	if out == "" {
		out = deb.GetFilename()
	}
	// With -o /dev/stdout the message would land in the package
	if isFile(stdout, out) {
		return nil
	}
	fmt.Fprintln(stdout, "debpkg: written:", out)
	return nil
}

// isFile reports whether w is the file at path
func isFile(w io.Writer, path string) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	a, err := f.Stat()
	if err != nil {
		return false
	}
	b, err := os.Stat(path)
	return err == nil && os.SameFile(a, b)
}
