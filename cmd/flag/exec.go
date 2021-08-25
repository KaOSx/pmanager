package flag

import (
	"fmt"
	"pmanager/database"
	"pmanager/util/shell"
	"time"
)

func Exec() {
	var flags []database.Flag
	for {
		args := shell.Prompt("> ")
		if len(args) == 0 {
			fmt.Println("Type help for usage")
		} else {
			switch args[0] {
			case "help":
				fmt.Print(help)
			case "quit":
				return
			case "list":
				flags = listFlags()
				for i, f := range flags {
					date := f.CreatedAt.Format(time.RFC1123)
					fmt.Printf("\033[1;36m%d\033[m → \033[1;32m%s\033m \033[1;31m\033[m\n", i+1, f.FullName())
					fmt.Printf("\033[1mDate:   \033[m %s\n", date)
					fmt.Printf("\033[1mEmail:  \033[m %s\n", f.Email)
					fmt.Printf("\033[1mComment:\033[m %s\n", f.Comment)
				}
			case "delete":
				ids := getIds(args[1:], flags)
				c := deleteFlags(ids)
				fmt.Printf("%d flag(s) deleted\n", c)
				flags = nil
			default:
				fmt.Printf("Command “%s” unknown. Type help for usage\n", args[0])
			}
		}
	}
}
