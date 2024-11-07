# pidcat
Makes `adb logcat` colored and adds the feature of being able to filter by app or tag. A Golang port~ of [github.com/JakeWharton/pidcat](https://github.com/JakeWharton/pidcat).

![Demo Image](demo.png)

## Usage
```
Usage:
  pidcat [flags]

Flags:
PACKAGE OPTIONS:
   -a, -all               display messages from all packages
   -current               filter by the app currently in the foreground
   -p, -package string[]  application package name(s)
   -list-packages         list all third party package names
   -list-all-packages     list all package names

ADB OPTIONS:
   -s, -serial string  device serial number (adb -s)
   -d, -device         use the first device (adb -d)
   -e, -emulator       use the first emulator (adb -e)
   -adb-path string    path to the ADB binary (default "adb")

LOGCAT OPTIONS:
   -l, -min-level value       minimum log level to be displayed (default verbose)
   -c, -clear                 clear the log before running
   -mt, -match-tag string[]   filter by specific tag(s)
   -ft, -filter-tag string[]  ignore specific tag(s)
   -lf, -log-file string      write logcat output to file (level:tag:message)
```

## Installation
### Binaries
Download the prebuilt binaries [here](https://github.com/rtfmkiesel/pidcat/releases).

### Using Go
If you have Go installed, run `go install github.com/rtfmkiesel/pidcat/cmd/pidcat@latest`.

### Build from source
```
git clone https://github.com/rtfmkiesel/pidcat
cd pidcat
go build -o pidcat -ldflags="-s -w" ./cmd/pidcat/pidcat.go
```

## How does this work?
In the background, `adb shell ps` is used to get the PIDs of the wanted packages. In the `adb logcat` output, the lines/entries have a PID assigned to them. That's enough to apply a filter.