package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MatejGadjo/NewsAggregator/internal/app"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("version=%s built=%s\n", version, buildTime)
		os.Exit(0)
	}

	app.Initialize().Run()
}
