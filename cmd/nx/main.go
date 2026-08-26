// Command nx is a read-only query CLI for Sonatype Nexus Repository 3.
//
// main is wiring only: it constructs the dependency container and hands it to
// internal/cli. All business logic lives under internal/.
package main

import (
	"os"

	"github.com/addozhang/nexus-cli/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
