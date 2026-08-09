package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type Color string

const (
	ColorReset  Color = "\033[0m"
	ColorBold   Color = "\033[1m"
	ColorRed    Color = "\033[31m"
	ColorGreen  Color = "\033[32m"
	ColorYellow Color = "\033[33m"
	ColorBlue   Color = "\033[34m"
	ColorPurple Color = "\033[35m"
	ColorCyan   Color = "\033[36m"
	ColorWhite  Color = "\033[37m"
)

var useColor = true
var sharedReader = bufio.NewReader(os.Stdin)

func init() {
	fi, err := os.Stdout.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		useColor = false
	}
}

func colorize(c Color, s string) string {
	if !useColor {
		return s
	}
	return string(c) + s + string(ColorReset)
}

func colorizef(c Color, format string, a ...interface{}) string {
	return colorize(c, fmt.Sprintf(format, a...))
}

func PrintBanner() {
	banner := `
██████╗ ███████╗███╗   ██╗ ██████╗ ███████╗
██╔══██╗██╔════╝████╗  ██║██╔═══██╗██╔════╝
██████╔╝█████╗  ██╔██╗ ██║██║   ██║███████╗
██╔══██╗██╔══╝  ██║╚██╗██║██║   ██║╚════██║
██║  ██║███████╗██║ ╚████║╚██████╔╝███████║
╚═╝  ╚═╝╚══════╝╚═╝  ╚═══╝ ╚═════╝ ╚══════╝
`
	fmt.Println(colorize(ColorCyan, banner))
	title := colorizef(ColorBold, "         Interactive Installation Wizard v1.0")
	subtitle := colorizef(ColorYellow, "         RenOS - The Modern AI Platform")
	fmt.Println(title)
	fmt.Println(subtitle)
	fmt.Println()
}

func PrintProgress(current, total int) {
	width := 40
	filled := current * width / total
	bar := "["
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "="
		} else if i == filled && current < total {
			bar += ">"
		} else {
			bar += "-"
		}
	}
	bar += "]"

	pct := float64(current) * 100 / float64(total)
	progress := colorizef(ColorCyan, "Step %d/%d", current, total)
	barStr := colorize(ColorGreen, bar)
	fmt.Printf("  %s  %s  %.0f%%\n", progress, barStr, pct)
	fmt.Println()
}

func PrintSection(title string) {
	line := strings.Repeat("=", 60)
	fmt.Println(colorize(ColorBlue, line))
	fmt.Println(colorizef(ColorBold+ColorBlue, "  %s", title))
	fmt.Println(colorize(ColorBlue, line))
	fmt.Println()
}

func PrintTable(headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}

	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	for i := range colWidths {
		colWidths[i] += 2
	}

	red := func(v string) string { return colorize(ColorRed, v) }
	if !useColor {
		red = func(v string) string { return v }
	}

	var sb strings.Builder
	for i, h := range headers {
		fmt.Fprintf(&sb, "%-*s", colWidths[i], h)
	}
	fmt.Println(red(sb.String()))

	var sep strings.Builder
	for _, w := range colWidths {
		sep.WriteString(strings.Repeat("-", w))
	}
	fmt.Println(sep.String())

	for _, row := range rows {
		var rb strings.Builder
		for i, cell := range row {
			if i < len(colWidths) {
				fmt.Fprintf(&rb, "%-*s", colWidths[i], cell)
			}
		}
		fmt.Println(rb.String())
	}
	fmt.Println()
}

func Confirm(prompt string, defaultYes bool) bool {
	suffix := " [Y/n]"
	if !defaultYes {
		suffix = " [y/N]"
	}
	fmt.Printf("%s%s: ", prompt, colorize(ColorCyan, suffix))

	input, err := readInput()
	if err != nil {
		return defaultYes
	}
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

func Prompt(label, defaultVal string, validateFn func(string) error) string {
	for {
		defaultStr := ""
		if defaultVal != "" {
			defaultStr = colorizef(ColorYellow, " [%s]", defaultVal)
		}
		fmt.Printf("%s%s: ", label, defaultStr)

		input, err := readInput()
		if err != nil {
			return defaultVal
		}
		input = strings.TrimSpace(input)

		if input == "" && defaultVal != "" {
			return defaultVal
		}
		if input == "" {
			fmt.Println(colorize(ColorRed, "  This field is required."))
			continue
		}
		if validateFn != nil {
			if err := validateFn(input); err != nil {
				fmt.Printf("%s %v\n", colorize(ColorRed, "!"), err)
				continue
			}
		}
		return input
	}
}

func PromptSecret(label string) string {
	for {
		fmt.Printf("%s: ", label)

		input, err := readSecret()
		if err != nil {
			fmt.Println()
			return input
		}

		if input == "" {
			fmt.Println(colorize(ColorRed, "  This field is required."))
			continue
		}
		return input
	}
}

func PromptSelect(label string, options []string, defaultVal string) string {
	for {
		fmt.Println(label + ":")
		for i, opt := range options {
			mark := " "
			if opt == defaultVal {
				mark = colorize(ColorYellow, "*")
			}
			fmt.Printf("  %s %d. %s\n", mark, i+1, opt)
		}
		defaultHint := ""
		if defaultVal != "" {
			defaultHint = colorizef(ColorYellow, " [%s]", defaultVal)
		}
		fmt.Printf("Enter number (1-%d)%s: ", len(options), defaultHint)

		input, err := readInput()
		if err != nil {
			if defaultVal != "" {
				return defaultVal
			}
			continue
		}
		input = strings.TrimSpace(input)

		if input == "" && defaultVal != "" {
			return defaultVal
		}

		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > len(options) {
			fmt.Printf("%s Please enter a number between 1 and %d.\n", colorize(ColorRed, "!"), len(options))
			continue
		}
		return options[n-1]
	}
}

func PromptMultiSelect(label string, options []string) []string {
	selected := make([]bool, len(options))

	for {
		fmt.Println(label + " (toggle numbers, or press Enter when done):")
		for i, opt := range options {
			check := " "
			if selected[i] {
				check = colorize(ColorGreen, "x")
			}
			fmt.Printf("  [%s] %d. %s\n", check, i+1, opt)
		}
		fmt.Print("> ")

		input, err := readInput()
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)

		if input == "" {
			break
		}

		parts := strings.Split(input, ",")
		valid := false
		for _, part := range parts {
			part = strings.TrimSpace(part)
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 || n > len(options) {
				fmt.Printf("%s Invalid number: %s (valid: 1-%d)\n", colorize(ColorRed, "!"), part, len(options))
				continue
			}
			selected[n-1] = !selected[n-1]
			valid = true
		}

		if !valid {
			continue
		}

		fmt.Print("\033[A")  // cursor up
		for range options {
			fmt.Print("\033[A") // cursor up for each option line
		}
		fmt.Print("\033[A") // cursor up for the header line
		for i := 0; i < len(options)+2; i++ {
			fmt.Print("\033[K") // clear line
			if i < len(options)+1 {
				fmt.Print("\033[A") // cursor up again
			}
		}
	}

	var result []string
	for i, opt := range options {
		if selected[i] {
			result = append(result, opt)
		}
	}
	return result
}

func readInput() (string, error) {
	input, err := sharedReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(input, "\r\n"), nil
}

func readSecret() (string, error) {
	fd := int(os.Stdin.Fd())
	var oldState syscall.Termios
	if _, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL, uintptr(fd), syscall.TCGETS,
		uintptr(unsafe.Pointer(&oldState)), 0, 0, 0,
	); err != 0 {
		return readInput()
	}

	newState := oldState
	newState.Lflag &^= syscall.ECHO
	newState.Lflag &^= syscall.ICANON
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0

	if _, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL, uintptr(fd), syscall.TCSETS,
		uintptr(unsafe.Pointer(&newState)), 0, 0, 0,
	); err != 0 {
		return readInput()
	}

	defer func() {
		syscall.Syscall6(
			syscall.SYS_IOCTL, uintptr(fd), syscall.TCSETS,
			uintptr(unsafe.Pointer(&oldState)), 0, 0, 0,
		)
		fmt.Println()
	}()

	var buf []byte
	for {
		b, err := sharedReader.ReadByte()
		if err != nil {
			return string(buf), err
		}
		switch b {
		case '\n', '\r':
			return string(buf), nil
		case 127, 8:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
		case 3:
			os.Exit(1)
		default:
			buf = append(buf, b)
		}
	}
}
