package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type Command struct {
	Command string   // the command
	Args    []string // any arguments
}

func newCommand(in []string) *Command {
	var command string
	var args []string
	for i, exp := range in {
		if i == 0 {
			command = exp
		} else {
			args = append(args, exp)
		}
	}
	return &Command{
		command,
		args,
	}
}

type CommandHandler interface {
	String() string // print the command
}

func (c Command) String() string {
	return fmt.Sprintf("Command: %s\nArgs: %s\n", c.Command, fmt.Sprint(c.Args))
}

func (c Command) Exit() int {
	if len(c.Args) > 0 {
		if c.Args[0] == "0" {
			os.Exit(0)
		} else if c.Args[0] == "1" {
			os.Exit(1)
		}
	}
	fmt.Fprint(os.Stdout, "Command format error, usage: exit <0|1>\n")
	return 1
}

func (c Command) Echo() int {
	text := Stringify(c.Args)
	fmt.Fprintf(os.Stdout, "%s\n", text)
	return 0
}

func (c Command) Pwd(d string) int {
	fmt.Fprint(os.Stdout, d+"\n")
	return 0
}

func (c Command) Cd(currentDir *string) int {

	path := c.Args[0]

	//We need to identify ./ and ../ (.. can be used many times)

	reDot2, errDot2 := regexp.Compile(`\.\.`)
	if errDot2 != nil {
		fmt.Fprint(os.Stdout, errDot2)
		return 1
	}
	dot2Matches := reDot2.FindAllString(path, -1)
	dot2MatchesLength := len(dot2Matches)
	if dot2MatchesLength > 0 {
		reBackDir, errBackDir := regexp.Compile(`/\w+$`)
		if errBackDir != nil {
			fmt.Fprint(os.Stdout, errBackDir)
			return 1
		}
		lCurrentDir := *currentDir
		for i := 0; i < dot2MatchesLength; i++ {
			lCurrentDir = reBackDir.ReplaceAllString(lCurrentDir, "")
		}
		path = lCurrentDir
	} else {
		reDot, errDot := regexp.Compile(`^\.`)
		if errDot != nil {
			fmt.Fprint(os.Stdout, errDot)
			return 1
		}
		dotMatch := reDot.MatchString(path)
		if dotMatch {
			path = reDot.ReplaceAllString(path, *currentDir)
		}
	}

	reHome, errHome := regexp.Compile(`^~`)
	if errHome != nil {
		fmt.Fprint(os.Stdout, errHome)
		return 1
	} else if reHome.MatchString(path) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprint(os.Stdout, err)
			return 1
		}
		path = reHome.ReplaceAllString(path, home)
	}

	//check path exists
	if _, err := os.Stat(path); err == nil {
		*currentDir = path
		return 0
	}
	fmt.Fprintf(os.Stdout, "cd: %s: No such file or directory\n", path)
	return 1
}

func (c Command) Type(paths []string) int {
	if len(c.Args) > 0 {
		switch arg := c.Args[0]; arg {
		case "type", "echo", "exit":
			fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", arg)
		default:
			//search in path
			for i := 0; i < len(paths); i++ {
				path := fmt.Sprintf("%s/%s", paths[i], c.Args[0])
				if _, err := os.Stat(path); err == nil {
					fmt.Fprintf(os.Stdout, "%s is %s\n", c.Args[0], path)
					return 0
				}
			}
			fmt.Fprintf(os.Stdout, "%s not found\n", arg)
		}
		return 0
	}
	fmt.Fprint(os.Stdout, "Command format error, usage: type <command>\n")
	return 1
}

func (c Command) Exec(paths []string) int {
	cmd := exec.Command(c.Command, strings.Join(c.Args, " "))
	cmd.Env = append(cmd.Env, paths...)
	cmd.Stdout = os.Stdout
	// fmt.Fprintf(os.Stdout, "%s\n", strings.Join(cmd.Env, "\n"))
	err := cmd.Run()
	if err == nil {
		return 0
	} else {
		fmt.Fprintf(os.Stdout, "%s: command not found\n", c.Command)
		return 1
	}
}

func Stringify(in []string) string {
	text := ""
	for _, arg := range in {
		text += arg + " "
	}
	text = strings.TrimSpace(text)
	return text
}

func main() {

	//retrieves the PATH arg passed in the shell script or the system path env var.
	path := strings.Split(os.Getenv("PATH"), ":")
	currentDir := "/app"

	for {
		fmt.Fprint(os.Stdout, "$ ")

		//Wait for user input.
		input, err := readInput()

		if err != nil {
			fmt.Fprint(os.Stdout, err)
		} else {
			parsedInput := parseInput(input)
			command := newCommand(parsedInput)
			// fmt.Print(command)
			exitCode := handleInput(command, path, &currentDir)

			if exitCode == 0 { // exit 0 command received - executed successfully
				// os.Exit(0)
			} else if exitCode == 1 { // exit 1 command received - an error occured
				// os.Exit(1)
			} else if exitCode == 2 { // bad command received
				fmt.Fprintf(os.Stdout, "%s: command not found\n", parsedInput[0])
			}
		}
	}
}

func readInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	return input, err
}

func handleInput(c *Command, paths []string, currentDir *string) int {
	for {
		switch c.Command {
		case "exit":
			return c.Exit()
		case "echo":
			return c.Echo()
		case "type":
			return c.Type(paths)
		case "pwd":
			return c.Pwd(*currentDir)
		case "cd":
			return c.Cd(currentDir)
		default:
			return c.Exec(paths)
		}
	}
}

func parseInput(in string) []string {
	exp := strings.Split(strings.TrimSpace(in), " ")
	return exp
}
