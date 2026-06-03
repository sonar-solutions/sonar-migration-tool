package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/sonar-solutions/sonar-migration-tool/cmd"
	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		var ec *common.ExitCodeError
		if errors.As(err, &ec) {
			os.Exit(ec.Code)
		}
		os.Exit(1)
	}
}
