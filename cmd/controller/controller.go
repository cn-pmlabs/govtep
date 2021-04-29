package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	driver "github.com/cn-pmlabs/govtep/driver/uninos"
	govtep "github.com/cn-pmlabs/govtep/go_vtep"
	odbc "github.com/cn-pmlabs/govtep/lib/ovsdb_client"
	"github.com/cn-pmlabs/govtep/tai"
)

var (
	version string = "0.0.0"
	help    bool   = false
)

func usage() {
	fmt.Fprintf(os.Stderr, `controller %s
Usage: controller [-h] [-v vtepdbAddr] [-s ovnsbAddr] [-n ovnnbAddr] [-f switchConfFile]

Options:
`, version)
	flag.PrintDefaults()
}

func init() {
	flag.StringVar(&odbc.VtepdbAddr, "v", odbc.VtepdbAddr, "vtep database address")
	flag.StringVar(&odbc.OvnsbAddr, "s", odbc.OvnsbAddr, "ovnsb database address")
	flag.StringVar(&odbc.OvnnbAddr, "n", odbc.OvnnbAddr, "ovnnb database address")
	flag.StringVar(&odbc.ConfigdbAddr, "c", odbc.ConfigdbAddr, "unos config database address")
	flag.StringVar(&govtep.SwitchConfFile, "f", govtep.SwitchConfFile, "Switch (group) configure file")
	flag.BoolVar(&help, "h", false, "display this help message")
	flag.Usage = usage
}

func main() {
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}

	// Start VTEPDB connection and update Notifier
	govtep.NewVtepDbClient()

	// TAI driver init
	driver.Init()
	// Start TAI vtepDB connection and update Notifier
	tai.NewTaiDbClient()

	// Can't ensure ovn db connection until ovn db target configured in vtepdb.Global
	govtep.OvnCentralConnect()

	// mainloop, do nothing for now
	for {
		time.Sleep(10 * time.Second)
	}
}
