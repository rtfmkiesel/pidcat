package options

import (
	"fmt"
	"os"

	"github.com/projectdiscovery/goflags"
	"github.com/rtfmkiesel/pidcat/internal/adb"
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
	flagset := goflags.NewFlagSet()
	flagset.SetConfigFilePath(os.DevNull) // Disables the invasive config file creation of goflags
	flagset.SetDescription("Makes 'adb logcat' colored and adds the feature of filtering by app or tag\nA Golang port of github.com/JakeWharton/pidcat")
	flagset.CreateGroup("Package Options", "Package Options",
		flagset.BoolVarP(&allPackages, "all", "a", false, "display messages from all packages"),
		flagset.BoolVar(&currentPackage, "current", false, "filter by the app currently in the foreground"),
		flagset.StringSliceVarP(&opt.Logcat.Packages, "package", "p", goflags.StringSlice{}, "application package name(s)", goflags.CommaSeparatedStringSliceOptions),
		flagset.BoolVar(&listPackages, "list-packages", false, "list all third party package names"),
		flagset.BoolVar(&listAllPackages, "list-all-packages", false, "list all package names"),
	)

	var (
		binpath  string
		serial   string
		device   bool
		emulator bool
	)
	flagset.CreateGroup("ADB Options", "ADB Options",
		flagset.StringVarP(&serial, "serial", "s", "", "device serial number (adb -s)"),
		flagset.BoolVarP(&device, "device", "d", false, "use the first device (adb -d)"),
		flagset.BoolVarP(&emulator, "emulator", "e", false, "use the first emulator (adb -e)"),
		flagset.StringVar(&binpath, "adb-path", "", "path to the ADB binary"),
	)

	var (
		clearOutput bool
		logFile     string
	)
	flagset.CreateGroup("Logcat Options", "Logcat Options",
		flagset.EnumVarP(&opt.Logcat.MinLevel, "min-level", "l", adb.LevelVerbose, "minimum log level to be displayed", adb.AllowedLevels),
		flagset.BoolVarP(&clearOutput, "clear", "c", false, "clear the log before running"),
		flagset.StringSliceVarP(&opt.Logcat.Tags, "match-tag", "mt", goflags.StringSlice{}, "filter by specific tag(s)", goflags.CommaSeparatedStringSliceOptions),
		flagset.StringSliceVarP(&opt.Logcat.IgnoreTags, "filter-tag", "ft", goflags.StringSlice{}, "ignore specific tag(s)", goflags.CommaSeparatedStringSliceOptions),
		flagset.StringVarP(&logFile, "log-file", "lf", "", "write logcat output to file (level:tag:message)"),
	)

	if err = flagset.Parse(); err != nil {
		return nil, err
	}

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

	return opt, nil
}
