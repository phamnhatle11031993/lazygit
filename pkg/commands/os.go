package commands

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
	gitconfig "github.com/tcnksm/go-gitconfig"
)

var (
	// ErrNoOpenCommand : When we don't know which command to use to open a file
	ErrNoOpenCommand = errors.New("Unsure what command to use to open this file")
	// ErrNoEditorDefined : When we can't find an editor to edit a file
	ErrNoEditorDefined = errors.New("No editor defined in $VISUAL, $EDITOR, or git config")
)

// Platform stores the os state
type Platform struct {
	os           string
	shell        string
	shellArg     string
	escapedQuote string
}

// OSCommand holds all the os commands
type OSCommand struct {
	Log      *logrus.Logger
	Platform *Platform
}

// NewOSCommand os command runner
func NewOSCommand(log *logrus.Logger) (*OSCommand, error) {
	osCommand := &OSCommand{
		Log:      log,
		Platform: getPlatform(),
	}
	return osCommand, nil
}

// RunCommand wrapper around commands
func (c *OSCommand) RunCommand(command string) (string, error) {
	c.Log.WithField("command", command).Info("RunCommand")
	splitCmd := strings.Split(command, " ")
	cmdOut, err := exec.Command(splitCmd[0], splitCmd[1:]...).CombinedOutput()
	return sanitisedCommandOutput(cmdOut, err)
}

// RunDirectCommand wrapper around direct commands
func (c *OSCommand) RunDirectCommand(command string) (string, error) {
	c.Log.WithField("command", command).Info("RunDirectCommand")

	cmdOut, err := exec.
		Command(c.Platform.shell, c.Platform.shellArg, command).
		CombinedOutput()
	return sanitisedCommandOutput(cmdOut, err)
}

func sanitisedCommandOutput(output []byte, err error) (string, error) {
	outputString := string(output)
	if outputString == "" && err != nil {
		return err.Error(), err
	}
	return outputString, err
}

func getPlatform() *Platform {
	switch runtime.GOOS {
	case "windows":
		return &Platform{
			os:           "windows",
			shell:        "cmd",
			shellArg:     "/c",
			escapedQuote: "\\\"",
		}
	default:
		return &Platform{
			os:           runtime.GOOS,
			shell:        "bash",
			shellArg:     "-c",
			escapedQuote: "\"",
		}
	}
}

// GetOpenCommand get open command
func (c *OSCommand) GetOpenCommand() (string, string, error) {
	//NextStep open equivalents: xdg-open (linux), cygstart (cygwin), open (OSX)
	trailMap := map[string]string{
		"xdg-open": " &>/dev/null &",
		"cygstart": "",
		"open":     "",
	}
	for name, trail := range trailMap {
		if out, _ := c.RunCommand("which " + name); out != "exit status 1" {
			return name, trail, nil
		}
	}
	return "", "", ErrNoOpenCommand
}

// VsCodeOpenFile opens the file in code, with the -r flag to open in the
// current window
// each of these open files needs to have the same function signature because
// they're being passed as arguments into another function,
// but only editFile actually returns a *exec.Cmd
func (c *OSCommand) VsCodeOpenFile(filename string) (*exec.Cmd, error) {
	_, err := c.RunCommand("code -r " + filename)
	return nil, err
}

// SublimeOpenFile opens the filein sublime
// may be deprecated in the future
func (c *OSCommand) SublimeOpenFile(filename string) (*exec.Cmd, error) {
	_, err := c.RunCommand("subl " + filename)
	return nil, err
}

// OpenFile opens a file with the given
func (c *OSCommand) OpenFile(filename string) (*exec.Cmd, error) {
	cmdName, cmdTrail, err := c.GetOpenCommand()
	if err != nil {
		return nil, err
	}
	_, err = c.RunCommand(cmdName + " " + filename + cmdTrail)
	return nil, err
}

// EditFile opens a file in a subprocess using whatever editor is available,
// falling back to core.editor, VISUAL, EDITOR, then vi
func (c *OSCommand) EditFile(filename string) (*exec.Cmd, error) {
	editor, _ := gitconfig.Global("core.editor")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		if _, err := c.RunCommand("which vi"); err == nil {
			editor = "vi"
		}
	}
	if editor == "" {
		return nil, ErrNoEditorDefined
	}
	return c.PrepareSubProcess(editor, filename)
}

// PrepareSubProcess iniPrepareSubProcessrocess then tells the Gui to switch to it
func (c *OSCommand) PrepareSubProcess(cmdName string, commandArgs ...string) (*exec.Cmd, error) {
	subprocess := exec.Command(cmdName, commandArgs...)
	return subprocess, nil
}
