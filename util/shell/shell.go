package shell

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func readline(prompt string) string {
	b := bufio.NewReader(os.Stdin)

	fmt.Print(prompt)
	s, _ := b.ReadString('\n')

	if l := len(s); l > 0 {
		s = s[:l-1]
	}

	return s
}

func prompt(question string, defaultValue any) string {
	return fmt.Sprintf("%s \033[33;1m[%v]\033[m ", question, defaultValue)
}

func Prompt(prompt string) []string {
	out := strings.Split(readline(prompt), " ")
	args := make([]string, 0, len(out))

	for _, e := range out {
		if e != "" {
			args = append(args, e)
		}
	}

	return args
}

func GetString(question string, defaultValue string) string {
	response := readline(prompt(question, defaultValue))

	if response == "" {
		return defaultValue
	}

	return response
}

func GetInt(question string, defaultValue int) int {
	response := readline(prompt(question, defaultValue))

	if i, err := strconv.Atoi(response); err == nil {
		return i
	}

	return defaultValue
}

func GetBool(question string, defaultValue bool) bool {
	df := "y/N"
	if defaultValue {
		df = "Y/n"
	}

	response := readline(prompt(question, df))
	if len(response) > 0 {
		switch response[0] {
		case 'y', 'Y':
			return true
		case 'n', 'N':
			return false
		}
	}

	return defaultValue
}
