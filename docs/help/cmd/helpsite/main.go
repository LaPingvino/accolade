// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

// Command helpsite writes Accolade's help as web pages, for the browser
// version (scripts/build-web.sh): helpsite -version v1.2.3 dist/web/help
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/LaPingvino/accolade/docs/help"
)

func main() {
	version := flag.String("version", "dev", "the version of Accolade the pages are for")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: helpsite [-version v] folder")
		os.Exit(2)
	}
	if err := help.WriteSite(flag.Arg(0), *version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
