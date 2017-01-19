package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mgutz/ansi"
)

const homeSigil = "~"
const pathSeparator = string(os.PathSeparator)

var projectFiles = flag.String("project-files", ".git,.hg,.svn,pom.xml,package.json,.editorconfig", "Files/directories that indicate a repository root when present")
var envRoots = flag.String("env-roots", "GOPATH", "Environment variables defining special directory paths that should be abbreviated as $ENV. $HOME is automatically included")
var color = flag.String("color", "project=blue+b,root=245,separator=245", "Color attributes in the form project=ATTR,root=ATTR, where attributes are as documented in https://github.com/mgutz/ansi")
var escapeColor = flag.Bool("zsh-escape-color", false, "If true, colors will be escaped with %{ %} for use in zsh prompts")

type colorizer struct {
	EnvRoot   func(string) string
	Separator func(string) string
	Project   func(string) string
	None      func(string) string
}

type env struct {
	name  string
	value string
}

type envrange []env

// EnvPrefix returns an environment prefix that applies to path, and how many
// leading path segments that prefix replaces.
func (e envrange) EnvPrefix(path string) (prefix string, replacedSegments int) {
	for _, envRep := range e {
		if path == envRep.value || strings.HasPrefix(path, envRep.value+pathSeparator) {
			return envRep.name, strings.Count(envRep.value, pathSeparator) + 1
		}
	}
	return "", 0
}

type pathShortener struct {
	colorizer

	rootFiles []string
	envs      envrange
}

func stripTrailingSlash(dir string) string {
	if len(dir) <= 1 {
		return dir
	}
	return strings.TrimSuffix(dir, pathSeparator)
}

func getEnvs(envNames []string) (envs envrange) {
	for _, envName := range envNames {
		envVar := strings.TrimSpace(envName)
		if envVar == "" {
			continue
		}

		if envValue := os.Getenv(envName); envValue != "" {
			envs = append(envs, env{name: "$" + envName, value: stripTrailingSlash(envValue)})
		}
	}
	return append(envs, env{name: homeSigil, value: stripTrailingSlash(os.Getenv("HOME"))})
}

func (p pathShortener) sourceRoot(dir string) bool {
	for _, file := range p.rootFiles {
		if fileExists(filepath.Join(dir, file)) {
			return true
		}
	}
	return false
}

func (p pathShortener) shorten(pathSegment string) string {
	return shortenRegex.ReplaceAllString(pathSegment, "$1")
}

// Shorten shortens a path applying rootEnvs and rootFiles
func (p pathShortener) Shorten(path string) string {
	if path == "" {
		return ""
	}

	prefix, start := p.envs.EnvPrefix(path)

	var shortenedSegments []string
	add := func(segment string) {
		shortenedSegments = append(shortenedSegments, segment)
	}

	if prefix != "" {
		add(p.colorizer.EnvRoot(prefix))
	}

	segments := strings.Split(path, pathSeparator)
	endSegment := len(segments) - 1

	shortenedSegment := func(i int) string {
		dir := strings.Join(segments[:i+1], pathSeparator)
		if p.sourceRoot(dir) {
			return p.colorizer.Project(segments[i])
		}
		return p.shorten(segments[i])
	}

	for i := start; i < endSegment; i++ {
		add(shortenedSegment(i))
	}

	if endSegment >= start {
		col := p.colorizer.None
		if p.sourceRoot(path) {
			col = p.colorizer.Project
		}
		add(col(segments[endSegment]))
	}

	return strings.Join(shortenedSegments, p.colorizer.Separator(pathSeparator))
}

var shortenRegex = regexp.MustCompile(`^([\w-]).*`)

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func split(s string) []string {
	return splitSep(s, ",")
}

func splitSep(s, sep string) []string {
	inputSegments := strings.Split(s, sep)
	trimmedStrings := make([]string, 0, len(inputSegments))
	for _, part := range inputSegments {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			trimmedStrings = append(trimmedStrings, trimmed)
		}
	}
	return trimmedStrings
}

func createColorizer(colorDef string, escapeColors bool) colorizer {
	splitIdentifierAttribute := func(colorSpec string) (string, string) {
		parts := splitSep(colorSpec, "=")
		if len(parts) != 2 {
			return "", ""
		}
		return parts[0], parts[1]
	}

	noop := func(s string) string { return s }

	escape := noop
	if escapeColors {
		escape = func(s string) string {
			return "%{" + s + "%}"
		}
	}

	makeColorizer := func(attr, reset string) func(string) string {
		return func(s string) string {
			return escape(attr) + s + escape(reset)
		}
	}

	c := colorizer{
		EnvRoot:   noop,
		Project:   noop,
		Separator: noop,
		None:      noop,
	}

	colorConfigSettings := map[string]*func(string) string{
		"root":      &c.EnvRoot,
		"project":   &c.Project,
		"separator": &c.Separator,
	}

	for _, colorSpec := range split(colorDef) {
		identifier, attrDef := splitIdentifierAttribute(colorSpec)
		if setting := colorConfigSettings[identifier]; setting != nil {
			*setting = makeColorizer(ansi.ColorCode(attrDef), ansi.Reset)
		}
	}
	return c
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[options] dir-name1 [dir-name2 ...]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	shorten := pathShortener{
		rootFiles: split(*projectFiles),
		envs:      getEnvs(split(*envRoots)),
		colorizer: createColorizer(*color, *escapeColor),
	}.Shorten

	for _, path := range flag.Args() {
		fmt.Println(shorten(path))
	}
}
