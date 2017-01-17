package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const homeSigil = "~"
const pathSeparator = string(os.PathSeparator)

var rootFiles = flag.String("root-files", ".git,.hg,.svn,pom.xml,package.json,.editorconfig", "Files/directories that indicate a repository root when present")
var magicEnv = flag.String("magic-root", "GOPATH", "Environment variables defining special directory paths that should be abbreviated as $ENV")

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
			return envRep.name, len(strings.Split(envRep.value, pathSeparator))
		}
	}
	return "", 0
}

type pathShortener struct {
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
	return append(envs, env{name: homeSigil, value: os.Getenv("HOME")})
}

func newPathShortener(rootFiles, envNames []string) pathShortener {
	return pathShortener{
		rootFiles: rootFiles,
		envs:      getEnvs(envNames),
	}
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
		add(prefix)
	}

	segments := strings.Split(path, pathSeparator)
	endSegment := len(segments) - 1

	shortenedSegment := func(i int) string {
		dir := strings.Join(segments[:i+1], pathSeparator)
		if p.sourceRoot(dir) {
			return segments[i]
		}
		return p.shorten(segments[i])
	}

	for i := start; i < endSegment; i++ {
		add(shortenedSegment(i))
	}

	if endSegment >= start {
		add(segments[endSegment])
	}

	return strings.Join(shortenedSegments, pathSeparator)
}

var shortenRegex = regexp.MustCompile(`^([\w-]).*`)

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func split(s string) []string {
	inputSegments := strings.Split(s, ",")
	trimmedStrings := make([]string, 0, len(inputSegments))
	for _, part := range inputSegments {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			trimmedStrings = append(trimmedStrings, trimmed)
		}
	}
	return trimmedStrings
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[options] dir-name1 [dir-name2 ...]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	shortener := newPathShortener(split(*rootFiles), split(*magicEnv))
	for _, path := range flag.Args() {
		fmt.Println(shortener.Shorten(path))
	}
}
