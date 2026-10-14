package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type colorSpec struct {
	hl string
	lh string
}

func defaultColorSpecs() map[string]colorSpec {
	specs := map[string]colorSpec{
		"yellow":   {"black on_yellow", "yellow"},
		"byellow":  {"black on_bright_yellow", "bright_yellow"},
		"white":    {"black on_white", "white"},
		"bwhite":   {"black on_bright_white", "bright_white"},
		"black":    {"white on_black", "black"},
		"bblack":   {"white on_bright_black", "bright_black"},
		"blue":     {"black on_blue", "blue"},
		"bblue":    {"black on_bright_blue", "bright_blue"},
		"magenta":  {"black on_magenta", "magenta"},
		"bmagenta": {"black on_bright_magenta", "bright_magenta"},
		"green":    {"black on_green", "green"},
		"bgreen":   {"black on_bright_green", "bright_green"},
		"cyan":     {"black on_cyan", "cyan"},
		"bcyan":    {"black on_bright_cyan", "bright_cyan"},
		"red":      {"black on_red", "red"},
		"bred":     {"black on_bright_red", "bright_red"},
	}
	specs["_default"] = specs["byellow"]
	return specs
}

type rule struct {
	pattern     string
	bold        bool
	escaped     bool
	respectCase bool
	pre         string
	post        string
	color       string
	rawColor    string
	hasRawColor bool
}

type flags struct {
	help      bool
	colorTest bool
}

type HL struct {
	debug      bool
	colorSpecs map[string]colorSpec
}

func newHL() *HL {
	return &HL{colorSpecs: defaultColorSpecs()}
}

var ansiNames = map[string]string{
	"clear": "0", "reset": "0",
	"bold": "1", "dark": "2", "faint": "2",
	"italic": "3", "underline": "4", "underscore": "4",
	"blink": "5", "reverse": "7", "concealed": "8", "strikethrough": "9",
	"black": "30", "red": "31", "green": "32", "yellow": "33",
	"blue": "34", "magenta": "35", "cyan": "36", "white": "37",
	"bright_black": "90", "bright_red": "91", "bright_green": "92",
	"bright_yellow": "93", "bright_blue": "94", "bright_magenta": "95",
	"bright_cyan": "96", "bright_white": "97",
	"on_black": "40", "on_red": "41", "on_green": "42", "on_yellow": "43",
	"on_blue": "44", "on_magenta": "45", "on_cyan": "46", "on_white": "47",
	"on_bright_black": "100", "on_bright_red": "101", "on_bright_green": "102",
	"on_bright_yellow": "103", "on_bright_blue": "104", "on_bright_magenta": "105",
	"on_bright_cyan": "106", "on_bright_white": "107",
}

func ansiCode(token string) (string, bool) {
	if code, ok := ansiNames[token]; ok {
		return code, true
	}
	if strings.HasPrefix(token, "on_ansi") {
		if n, err := strconv.Atoi(token[len("on_ansi"):]); err == nil && n >= 0 && n <= 255 {
			return "48;5;" + strconv.Itoa(n), true
		}
	}
	if strings.HasPrefix(token, "ansi") {
		if n, err := strconv.Atoi(token[len("ansi"):]); err == nil && n >= 0 && n <= 255 {
			return "38;5;" + strconv.Itoa(n), true
		}
	}
	return "", false
}

func parseColorSpec(spec string) (string, error) {
	fields := strings.Fields(strings.ToLower(spec))
	if len(fields) == 0 {
		return "", nil
	}
	codes := make([]string, 0, len(fields))
	for _, field := range fields {
		code, ok := ansiCode(field)
		if !ok {
			return "", fmt.Errorf("Invalid attribute name %s", field)
		}
		codes = append(codes, code)
	}
	return strings.Join(codes, ";"), nil
}

func colorsDisabled() bool {
	return os.Getenv("ANSI_COLORS_DISABLED") != "" || os.Getenv("NO_COLOR") != ""
}

func (h *HL) wrap(spec string) (string, string, error) {
	if colorsDisabled() {
		return "", "", nil
	}
	codes, err := parseColorSpec(spec)
	if err != nil {
		return "", "", err
	}
	if codes == "" {
		return "", "", nil
	}
	return "\x1b[" + codes + "m", "\x1b[0m", nil
}

func (h *HL) parseCmdArgs(args []string) ([]rule, flags, error) {
	var rules []rule
	var f flags
	bold := false
	reverse := false
	respectCase := true
	escaped := false
	colorKey := "_default"
	pre := ""
	post := ""
	rawColor := ""
	hasRawColor := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		specArg := false

		switch {
		case arg == "--bold" || arg == "-b":
			bold = true
			specArg = true
		case arg == "--reverse" || arg == "-r":
			reverse = true
			specArg = true
		case arg == "-i":
			respectCase = false
			specArg = true
		case arg == "-h" || arg == "--help":
			f.help = true
		case arg == "--colortest":
			f.colorTest = true
		case arg == "--pre":
			if i+1 < len(args) {
				i++
				pre = args[i]
			}
			specArg = true
		case arg == "--post":
			if i+1 < len(args) {
				i++
				post = args[i]
			}
			specArg = true
		case arg == "--escaped":
			escaped = true
			specArg = true
		case arg == "--color":
			if i+1 < len(args) {
				i++
				rawColor = args[i]
				hasRawColor = true
			}
			specArg = true
		case arg == "-d" || arg == "--debug":
			h.debug = true
		default:
			if isColorArg(arg) {
				key := arg[2:]
				if _, ok := h.colorSpecs[key]; ok {
					colorKey = key
					specArg = true
				}
			}
		}

		if specArg {
			continue
		}

		r := rule{
			pattern:     arg,
			bold:        bold,
			escaped:     escaped,
			respectCase: respectCase,
			pre:         pre,
			post:        post,
		}
		if reverse {
			r.color = h.colorSpecs[colorKey].lh
		} else {
			r.color = h.colorSpecs[colorKey].hl
		}
		if hasRawColor {
			r.rawColor = rawColor
			r.hasRawColor = true
		}
		rules = append(rules, r)

		colorKey = "_default"
		bold = false
		reverse = false
		respectCase = true
		escaped = false
		rawColor = ""
		hasRawColor = false
		pre = ""
		post = ""
	}

	h.debugRules(rules)
	return rules, f, nil
}

func isColorArg(arg string) bool {
	if len(arg) < 3 || arg[0] != '-' || arg[1] != '-' {
		return false
	}
	for i := 2; i < len(arg); i++ {
		if arg[i] < 'a' || arg[i] > 'z' {
			return false
		}
	}
	return true
}

type compiledRule struct {
	r           rule
	re          *regexp.Regexp
	reUnescaped *regexp.Regexp
}

func (h *HL) compile(r rule, pattern string) (*regexp.Regexp, error) {
	expr := r.pre + "(" + pattern + ")" + r.post
	if !r.respectCase {
		expr = "(?i)" + expr
	}
	return regexp.Compile(expr)
}

func (h *HL) prepare(rules []rule) ([]compiledRule, error) {
	compiled := make([]compiledRule, 0, len(rules))
	for _, r := range rules {
		re, err := h.compile(r, r.pattern)
		if err != nil {
			return nil, err
		}
		cr := compiledRule{r: r, re: re}
		if r.escaped {
			cr.reUnescaped, err = h.compile(r, unescape(r.pattern))
			if err != nil {
				return nil, err
			}
		}
		compiled = append(compiled, cr)
	}
	return compiled, nil
}

var unescapeRe = regexp.MustCompile(`\\([|rnt'"\\])`)

func unescape(s string) string {
	return unescapeRe.ReplaceAllStringFunc(s, func(m string) string {
		switch m[1] {
		case 'r':
			return "\r"
		case 'n':
			return "\n"
		case 't':
			return "\t"
		default:
			return string(m[1])
		}
	})
}

func escapeReplacement(s string) string {
	return strings.ReplaceAll(s, "$", "$$")
}

func (h *HL) colorConcat(r rule) string {
	if r.hasRawColor {
		return r.rawColor
	}
	if r.bold {
		return "bold " + r.color
	}
	return r.color
}

func (h *HL) highlight(compiled []compiledRule, text string) (string, error) {
	for _, cr := range compiled {
		re := cr.re
		if cr.reUnescaped != nil && re.MatchString(text) {
			re = cr.reUnescaped
		}
		if !re.MatchString(text) {
			continue
		}
		open, close, err := h.wrap(h.colorConcat(cr.r))
		if err != nil {
			return "", err
		}
		preRepl := escapeReplacement(unescape(cr.r.pre))
		postRepl := escapeReplacement(unescape(cr.r.post))
		repl := preRepl + open + "${1}" + close + postRepl
		text = re.ReplaceAllString(text, repl)
	}
	return text, nil
}

func (h *HL) colorTest() (string, error) {
	maxWidth := 0
	for name := range h.colorSpecs {
		if len(name) > maxWidth {
			maxWidth = len(name)
		}
	}
	names := make([]string, 0, len(h.colorSpecs))
	for name := range h.colorSpecs {
		names = append(names, name)
	}
	sort.SliceStable(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})

	var sb strings.Builder
	for _, name := range names {
		spec := h.colorSpecs[name]
		pad := strings.Repeat(" ", maxWidth-len(name)) + "   "
		variants := []rule{
			{pattern: name, respectCase: true, color: spec.hl},
			{pattern: name, respectCase: true, color: spec.hl, bold: true},
			{pattern: name, respectCase: true, color: spec.lh},
			{pattern: name, respectCase: true, color: spec.lh, bold: true},
		}
		for _, v := range variants {
			compiled, err := h.prepare([]rule{v})
			if err != nil {
				return "", err
			}
			out, err := h.highlight(compiled, name)
			if err != nil {
				return "", err
			}
			sb.WriteString(out)
			sb.WriteString(pad)
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
