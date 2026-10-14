package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

const helpText = `hl - highlight text on the console

Usage:
    command | hl [options] [pattern ...]

Reads lines from standard input and writes them to standard output with each
pattern's matches highlighted. Patterns are matched in the order given.

Display modifiers and color options apply only to the pattern that immediately
follows them, then reset to their defaults for the next pattern. With no
modifiers, a pattern is highlighted as black text on a bright yellow background.

Options:
    -b, --bold         Render the following pattern in bold.
    -r, --reverse      Use the foreground-only form of the color (colored text on
                       the terminal's default background) instead of the default
                       black-on-color highlight.
    -i                 Match the following pattern case-insensitively. Patterns
                       are case-sensitive by default.
    --pre REGEX        Require REGEX to appear immediately before the following
                       pattern. The prefix is matched but not highlighted.
    --post REGEX       Require REGEX to appear immediately after the following
                       pattern. The suffix is matched but not highlighted.
    --escaped          Interpret backslash escapes in the following pattern:
                           \n  newline          \r  carriage return
                           \t  tab              \|  literal |
                           \'  literal '        \"  literal "
                           \\  literal \
    --color SPEC       Use a raw Term::ANSIColor-style SPEC for the following
                       pattern, e.g. "bold green on_blue". Overrides --bold.
    -d, --debug        Print rule/debug information to standard error.
    -h, --help         Show this help and exit.
    --colortest        Print a chart of every color and exit.

Colors:
    --COLOR            Highlight the following pattern using COLOR. A "b" prefix
                       selects the bright variant.
                           yellow   byellow   white    bwhite
                           black    bblack    blue     bblue
                           magenta  bmagenta  green    bgreen
                           cyan     bcyan     red      bred

    --color accepts the following raw attribute names:
                       clear reset bold dark faint italic underline underscore
                       blink reverse concealed strikethrough
                       black red green yellow blue magenta cyan white
                       bright_black ... bright_white
                       on_black ... on_white
                       on_bright_black ... on_bright_white
                       ansi0 ... ansi255
                       on_ansi0 ... on_ansi255

Environment:
    NO_COLOR               If set, disables colored output.
    ANSI_COLORS_DISABLED   If set, disables colored output.

Examples:
    tail -f app.log | hl --red ERROR --byellow WARN
    grep -i error app.log | hl -b --color 'bold white on_red' critical
    ps aux | hl -i --green nginx --cyan node
    hl --pre 'user=' --red root --post ':' < /etc/passwd
    hl --colortest
`

func help() {
	fmt.Print(helpText)
}

func (h *HL) debugRules(rules []rule) {
	if !h.debug {
		return
	}
	for i, r := range rules {
		fmt.Fprintf(os.Stderr, "hl debug: rule %d: pattern=%q bold=%t respectCase=%t escaped=%t pre=%q post=%q color=%q rawColor=%q hasRawColor=%t\n",
			i, r.pattern, r.bold, r.respectCase, r.escaped, r.pre, r.post, r.color, r.rawColor, r.hasRawColor)
	}
}

func main() {
	initConsole()

	h := newHL()
	rules, f, err := h.parseCmdArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "hl: %v\n", err)
		os.Exit(2)
	}

	if f.help {
		help()
		return
	}

	if f.colorTest {
		out, err := h.colorTest()
		if err != nil {
			fmt.Fprintf(os.Stderr, "hl: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.WriteString(out)
		return
	}

	compiled, err := h.prepare(rules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hl: %v\n", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	for {
		line, readErr := reader.ReadString('\n')
		if len(line) > 0 {
			out, err := h.highlight(compiled, line)
			if err != nil {
				writer.Flush()
				fmt.Fprintf(os.Stderr, "hl: %v\n", err)
				os.Exit(1)
			}
			if _, err := writer.WriteString(out); err != nil {
				return
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			writer.Flush()
			fmt.Fprintf(os.Stderr, "hl: %v\n", readErr)
			os.Exit(1)
		}
	}
}
