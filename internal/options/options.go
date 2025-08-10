package options

import (
	"fmt"
	"os"
	"strings"

	"github.com/rtfmkiesel/pidcat/internal/adb"
	flag "github.com/spf13/pflag"
)

type Options struct {
	ADBClient *adb.Client
	Logcat    *adb.LogcatOptions
	FhLogFile *os.File
}

// Parses the cli options
func Parse() (opt *Options, err error) {
	opt = &Options{
		Logcat: &adb.LogcatOptions{},
	}

	var (
		allPackages     bool
		currentPackage  bool
		listPackages    bool
		listAllPackages bool
	)

	flag.BoolVarP(&allPackages, "all", "a", false, "display messages from all packages")
	flag.BoolVar(&currentPackage, "current", false, "filter by the current app only")
	flag.StringSliceVarP(&opt.Logcat.Packages, "package", "p", nil, "application package name(s)")
	flag.BoolVar(&listPackages, "list-packages", false, "list all third party package names")
	flag.BoolVar(&listAllPackages, "list-all-packages", false, "list all package names")

	var (
		binpath  string
		serial   string
		device   bool
		emulator bool
	)

	flag.StringVarP(&serial, "serial", "s", "", "device serial number (adb -s)")
	flag.BoolVarP(&device, "device", "d", false, "use the first device (adb -d)")
	flag.BoolVarP(&emulator, "emulator", "e", false, "use the first emulator (adb -e)")
	flag.StringVar(&binpath, "adb-path", "", "path to the ADB binary")

	var (
		minLevel    string
		clearOutput bool
		logFile     string
	)

	flag.StringVarP(&minLevel, "min-level", "l", "V", "minimum log level to be displayed (V,D,I,W,E,F)")
	flag.BoolVarP(&clearOutput, "clear", "c", false, "clear the log before running")
	flag.StringSliceVarP(&opt.Logcat.Tags, "match-tag", "m", nil, "filter by specific tag(s)")
	flag.StringSliceVarP(&opt.Logcat.IgnoreTags, "filter-tag", "f", nil, "ignore specific tag(s)")
	flag.StringVarP(&logFile, "log-file", "L", "", "write logcat output to file (level:tag:message)")

	flag.Parse()

	// Select the connection option
	connectionStr := []string{}
	if serial != "" {
		connectionStr = append(connectionStr, []string{"-s", serial}...)
	} else if device {
		connectionStr = append(connectionStr, "-d")
	} else if emulator {
		connectionStr = append(connectionStr, "-e")
	} else {
		return nil, fmt.Errorf("mission adb option, chooose '-s/--serial', '-d/--device' or '-e/--emulator'")
	}

	opt.ADBClient, err = adb.NewClient(binpath, connectionStr)
	if err != nil {
		return nil, err
	}

	if listPackages {
		packages, err := opt.ADBClient.ListThirdPartyPackages()
		if err != nil {
			return nil, err
		}

		for _, p := range packages {
			fmt.Println(p)
		}

		os.Exit(0)
	}

	if listAllPackages {
		packages, err := opt.ADBClient.ListAllPackages()
		if err != nil {
			return nil, err
		}

		for _, p := range packages {
			fmt.Println(p)
		}

		os.Exit(0)
	}

	if allPackages {
		// Users wants all packages, do not filter

	} else if currentPackage {
		// Get the current package
		foregroundApp, err := opt.ADBClient.GetCurrentApp()
		if err != nil {
			return nil, err
		}
		opt.Logcat.Packages = append(opt.Logcat.Packages, foregroundApp)

	} else if len(opt.Logcat.Packages) == 0 {
		return nil, fmt.Errorf("no packge names supplied")
	}

	opt.ADBClient.BaseCmdLogcat = append(opt.ADBClient.BaseCmd, "logcat", "-v", "brief")

	if clearOutput {
		if err := opt.ADBClient.ClearLogcatOutput(); err != nil {
			return nil, err
		}
	}

	if logFile != "" {
		opt.FhLogFile, err = os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
	}

	minLevel = strings.ToUpper(minLevel)
	if _, ok := adb.LevelMap[minLevel]; !ok {
		return nil, fmt.Errorf("invalid level '%s'", minLevel)
	}
	opt.Logcat.MinLevel = minLevel

	return opt, nil
}
