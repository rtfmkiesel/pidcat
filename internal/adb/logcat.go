package adb

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/nathan-fiscaletti/consolesize-go"
)

type LogcatOptions struct {
	Packages   []string // The packages to filter for
	MinLevel   string   // The minimum log level to show
	Tags       []string // The tags to filter for
	IgnoreTags []string // The tags to ignore
	FhLogFile  *os.File // The file handle to write the logcat output to
}

// The struct to represent a logcat line
type LogcatEntry struct {
	Level string
	Tag   string
	PID   string
	MSG   string
}

const (
	// The available log levels
	LevelVerbose = iota // 0
	LevelDebug          // 1
	LevelInfo           // 2
	LevelWarning        // 3
	LevelError          // 4
	LevelFatal          // 5

	MaxLenTag = 20 // The maximum length of a tag for the terminal UI
)

var (
	// The regex to parse a logcat line
	reLine = regexp.MustCompile(`(\S){1}/([^\(]*)\([\s]*([\d]*)\):[\s]*([^\n]*)`)

	// The colors for the log levels
	colorLevelV = color.New(color.BgBlack, color.FgWhite)
	colorLevelD = color.New(color.BgBlue, color.FgBlack)
	colorLevelI = color.New(color.BgGreen, color.FgBlack)
	colorLevelW = color.New(color.BgYellow, color.FgBlack)
	colorLevelE = color.New(color.BgRed, color.FgBlack)
	colorLevelF = color.New(color.BgRed, color.FgBlack)

	// A slice of colors for the tags that gets rotated when a new tag is encountered
	colorTags = []*color.Color{
		color.New(color.BgBlack, color.FgRed),
		color.New(color.BgBlack, color.FgGreen),
		color.New(color.BgBlack, color.FgYellow),
		color.New(color.BgBlack, color.FgBlue),
		color.New(color.BgBlack, color.FgMagenta),
		color.New(color.BgBlack, color.FgCyan),
	}

	// A map to store encountered tags and their color
	tagColorMap = make(map[string]*color.Color)

	LevelMap = map[string]int{
		"V": LevelVerbose,
		"D": LevelDebug,
		"I": LevelInfo,
		"W": LevelWarning,
		"E": LevelError,
		"F": LevelFatal,
	}
)

// Clears the logcat output via 'logcat -c'
func (client *Client) ClearLogcatOutput() (err error) {
	if _, err := client.Run(5, "logcat", "-c"); err != nil {
		return err
	}

	return nil
}

// Parses a logcat line into a LogcatEntry struct
func ParseLogcatLine(line string) (entry LogcatEntry, err error) {
	matches := reLine.FindStringSubmatch(line)
	if len(matches) < 5 {
		return entry, fmt.Errorf("could not parse logcat line")
	}

	entry = LogcatEntry{
		Level: strings.TrimSpace(matches[1]),
		Tag:   strings.TrimSpace(matches[2]),
		PID:   strings.TrimSpace(matches[3]),
		MSG:   strings.TrimSpace(matches[4]),
	}

	return entry, err
}

// Prints a logcat line with colors
func (entry LogcatEntry) Print() {
	coloredName := formatTag(entry.Tag)

	// Color the level based on the log level
	var coloredLevel string
	switch entry.Level {
	case "V":
		coloredLevel = colorLevelV.Sprintf(" %s ", entry.Level)
	case "D":
		coloredLevel = colorLevelD.Sprintf(" %s ", entry.Level)
	case "I":
		coloredLevel = colorLevelI.Sprintf(" %s ", entry.Level)
	case "W":
		coloredLevel = colorLevelW.Sprintf(" %s ", entry.Level)
	case "E":
		coloredLevel = colorLevelE.Sprintf(" %s ", entry.Level)
	case "F":
		coloredLevel = colorLevelF.Sprintf(" %s ", entry.Level)
	}

	coloredMsg := colorLevelV.Sprint(formatMsg(entry.MSG)) // Use colorLevelV for the black background

	fmt.Fprintln(color.Output, coloredName+coloredLevel+coloredMsg)
}

// Writes a logcat line to a file
func (entry LogcatEntry) ToFile(fh *os.File) (err error) {
	if fh == nil {
		return nil
	}

	_, err = fh.WriteString(fmt.Sprintf("%s:%s:%s\n", entry.Level, entry.Tag, entry.MSG))
	if err != nil {
		return err
	}

	return nil
}

// Formats the tag to be colored and have a fixed length
func formatTag(tag string) string {
	// Add a space if the tag is empty or does not end with a space
	if len(tag) == 0 || tag[len(tag)-1] != ' ' {
		tag = tag + " "
	}

	// Trim the tag if it's too long
	if len(tag) > MaxLenTag {
		return " " + tag[:MaxLenTag-5] + "... "
	}

	str := fmt.Sprintf("%*s", MaxLenTag, tag)

	// Color the tag
	if _, exists := tagColorMap[tag]; !exists {
		// Unknown tag, use a new color
		tagColorMap[tag] = colorTags[0]
		// Rotate the colors
		colorTags = append(colorTags[1:], colorTags[0])
	}

	return tagColorMap[tag].Sprint(str)
}

// Formats the message to have a fixed length
func formatMsg(msg string) string {
	// Get the console width (3rd party because the stdlib does not provide a working solution for Windows)
	width, _ := consolesize.GetConsoleSize()

	// Calculate the maximum width for the message
	maxWidthMsg := width - MaxLenTag - 3 // 3 = 2 spaces and 1 char for level

	// Add a space if the message is empty or does not start with a space
	if len(msg) == 0 || msg[0] != ' ' {
		msg = " " + msg
	}

	// Trim the message if it's too long
	if len(msg) > maxWidthMsg {
		return msg[:maxWidthMsg-4] + "... "
	}

	// Add spaces to fill the rest of the line
	for len(msg) < maxWidthMsg {
		msg += " "
	}

	return msg
}

// Checks if the level is higher or the same as the wanted one
func IsLevelInScope(entryLevel string, wantedLevel string) bool {
	return LevelMap[entryLevel] >= LevelMap[wantedLevel]
}
