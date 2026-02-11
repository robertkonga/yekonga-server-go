package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// ANSI escape codes for colors
const (
	Red     = "\033[91m"
	Green   = "\033[92m"
	Yellow  = "\033[93m"
	Blue    = "\033[94m"
	Magenta = "\033[95m"
	Cyan    = "\033[96m"
	White   = "\033[97m"
	Reset   = "\033[0m" // Reset color
)

func Log(args ...any) {
	printColor(Magenta, args)
}

func Error(args ...any) {
	printColor(Red, args)
}

func Success(args ...any) {
	printColor(Green, args)
}

func Warn(args ...any) {
	printColor(Yellow, args)
}

func Info(args ...any) {
	printColor(Blue, args)
}

func Logo() {
	banner := `
   __     __  _                                  
   \ \   / / | |                                 
    \ \_/ /__| | _ ___  _ __   __ _  __ _        
     \   / _ \ |/ / _ \| '_ \ /  ' |/  ' |       
      | |  __/   < (_) | | | |  () |  () |       
      |_|\___|_|\_\___/|_| |_|\__  |\__,_|       
                              \____|   `

	Warn(banner, "no-line-break")
	Info("SERVER", "no-line-break")
	Warn(".", "no-line-break")
	Info("GO \n")
}

func printColor(color string, args []any) {
	count := len(args)
	hasFormat := false
	formatString := ""
	otherArgs := []any{}
	noLineBreak := slices.Contains(args, "no-line-break")
	skipPos := slices.Index(args, "no-line-break")

	if count > 1 && !noLineBreak {
		if s, ok := args[0].(string); ok && strings.Contains(s, "%") {
			hasFormat = true
		}
	}

	for argNum, arg := range args {
		if noLineBreak && argNum == skipPos {
			continue
		}

		if argNum == 0 {
			if hasFormat {
				if v, ok := arg.(string); ok {
					formatString = v
				}
			} else {
				fmt.Print(color, getString(arg), Reset)
				if count > 1 && !noLineBreak {
					fmt.Print(": ")
				}
			}
		} else {
			if hasFormat {
				if err, ok := arg.(error); ok {
					otherArgs = append(otherArgs, err.Error())
				} else {
					otherArgs = append(otherArgs, arg)
				}
			} else {
				fmt.Print(getString(arg) + " ")
			}
		}
	}

	if hasFormat {
		fmt.Print(color, getString(string(fmt.Appendf([]byte{}, formatString, otherArgs...))), Reset)
	}

	if !noLineBreak {
		fmt.Print("\n")
	}
}

func getString(data interface{}) string {
	var str string

	if v, ok := data.(string); ok {
		str = v
	} else {
		str = toJson(data)
	}

	return str
}

func toJson(data interface{}) string {
	jsonData, _ := json.MarshalIndent(data, "", " ")

	return string(jsonData)
}

func loadFile(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return string(bytes)
}

// func (p *pp) doPrintln(a []any) {
// 	for argNum, arg := range a {
// 		if argNum > 0 {
// 			p.buf.writeByte(' ')
// 		}
// 		p.printArg(arg, 'v')
// 	}
// 	p.buf.writeByte('\n')
// }
