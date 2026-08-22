package commands

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

type Command struct {
	interfaces.Command
	Perform func(*utils.Writer, *parser.State, *any) (commandEdits map[string]utils.TextEdits, err error)
}

var Commands []Command
var CommandNames []string

var CommandMap = map[string]Command{}

type Action struct {
	Perform func(*utils.Writer, parser.State, parser.File, utils.Range) (actionEdits []utils.TextEdit, allowed bool, err error)
	Title   string
}

func registerCommand(command Command) {
	Commands = append(Commands, command)
}

func InitCommands() {
	registerCommand(Command{Command: interfaces.Command{Command: "ts_inspector/addImport", Title: "Add Import"}, Perform: AddImport})
	registerCommand(Command{Command: interfaces.Command{Command: "ts_inspector/goToDeclaringModule", Title: "Go to declaring module"}, Perform: GoToDeclaringModule})
	registerCommand(Command{Command: interfaces.Command{Command: "ts_inspector/printProviders", Title: "Print providers"}, Perform: PrintProviders})
	registerCommand(Command{Command: interfaces.Command{Command: "ts_inspector/saveDotCfg", Title: "Save dot graph for CFG"}, Perform: SaveDotForCfg})
	registerCommand(Command{Command: interfaces.Command{Command: "ts_inspector/viewTcb", Title: "View the TCB for the template"}, Perform: ViewTcb})

	for _, command := range Commands {
		CommandNames = append(CommandNames, command.Command.Command)
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
