package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/rtfmkiesel/pidcat/internal/adb"
	"github.com/rtfmkiesel/pidcat/internal/options"
)

func main() {
	opt, err := options.Parse()
	if err != nil {
		fmt.Fprintf(color.Error, "FATAL: %s\n", err)
		os.Exit(1)
	}

	// To keep track of wanted PIDs
	pids := []string{}

	// Create a context to later kill the logcat process
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, opt.ADBClient.BaseCmdLogcat[0], opt.ADBClient.BaseCmdLogcat[1:]...)

	// Capture the output of the logcat command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(color.Error, "FATAL: %s\n", err)
		return
	}

	// Start logcat
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(color.Error, "FATAL: %s\n", err)
		os.Exit(1)
	}

	// Create a go function that every two seconds checks for the PIDs of the wanted packages
	stopChanPidWatchDog := make(chan bool)
	wgPidWatchDog := new(sync.WaitGroup)
	wgPidWatchDog.Add(1)
	go func() {
		defer wgPidWatchDog.Done()

		for {
			select {
			case <-stopChanPidWatchDog:
				// We got a stop signal, return
				return
			default:
				for _, slug := range opt.Logcat.Packages {
					pid, err := opt.ADBClient.GetPID(slug)
					if err != nil {
						fmt.Fprintf(color.Error, "FATAL: %s\n", err)
						os.Exit(1)
					}

					// Add the pid to the slice if it's not already there
					if !slices.Contains(pids, pid) {
						pids = append(pids, pid)
					}
				}
			}

			time.Sleep(time.Second * 2)
		}
	}()

	// Channel were the logcat lines are sent to
	chanLogcatLines := make(chan string)
	wgOutputWriter := new(sync.WaitGroup)

	// Start a go function that reads the logcat lines and prints them to the terminal after filtering and formatting
	wgOutputWriter.Add(1)
	go func() {
		defer wgOutputWriter.Done()
		for line := range chanLogcatLines {
			entry, err := adb.ParseLogcatLine(line)
			if err != nil {
				continue // Ignore parse errors
			}

			// Check if the PID of the entry is not in the wanted PIDs
			if len(pids) > 0 && !slices.Contains(pids, entry.PID) {
				continue
			}

			// Check if the level is in scope to be processed
			if !adb.IsLevelInScope(entry.Level, opt.Logcat.MinLevel) {
				continue
			}

			// Check if the tag is to be ignored
			if slices.Contains(opt.Logcat.IgnoreTags, entry.Tag) {
				continue
			}

			// Check if the tag is not wanted if there are tags to be matched
			if len(opt.Logcat.Tags) > 0 && !slices.Contains(opt.Logcat.Tags, entry.Tag) {
				continue
			}

			// Print the logcat line
			entry.Print()
		}
	}()

	// Start a go function that reads the logcat lines and sends them to the channel
	wgLogcatReader := new(sync.WaitGroup)
	wgLogcatReader.Add(1)
	go func() {
		defer wgLogcatReader.Done()

		// Read the logcat lines
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			chanLogcatLines <- scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(color.Error, "FATAL: %s\n", err)
			os.Exit(1)
		}

		close(chanLogcatLines)
	}()

	// Wait for the user to press CTRL+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	// Wait for the logcat process to finish
	cmd.Wait()

	wgLogcatReader.Wait()
	wgOutputWriter.Wait()
}
