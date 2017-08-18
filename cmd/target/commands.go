package target
import (
	"github.com/mitchellh/cli"
)

var Commands map[string]cli.CommandFactory
func init(){
	Commands= map[string]cli.CommandFactory{
		"ls": func()(cli.Command,error){
			return LsCommand{},nil
		},
	}
}
