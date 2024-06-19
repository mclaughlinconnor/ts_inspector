package commands

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

type Command struct {
	interfaces.Command
	Perform func(parser.State, *[]any) (commandEdits map[string]utils.TextEdits, err error)
}

var Commands []Command

var CommandMap map[string]Command = map[string]Command{}

type Action struct {
	Perform func(parser.State, parser.File, utils.Range) (actionEdits []utils.TextEdit, allowed bool, err error)
	Title   string
}

func registerCommand(command Command) {
	Commands = append(Commands, command)
}

func InitCommands() {
	registerCommand(
		Command{
			Command: interfaces.Command{
				Command: "ts_inspector/addImport",
				Title:   "Add Import",
			},
			Perform: AddImport,
		},
	)

	for _, command := range Commands {
		CommandMap[command.Command.Command] = command
	}
}

func GetLspCommands() []string {
	commands := []string{}
	for _, command := range Commands {
		commands = append(commands, command.Command.Command)
	}

	return commands
}
