package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const homeSigil = "~"
const pathSeparator = string(os.PathSeparator)

var rootFiles = []string{".git", ".hg", ".svn", "pom.xml", "package.json", ".editorconfig"}
var shortenRegex = regexp.MustCompile(`^([\w-]).*`)

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func sourceRoot(dir string) bool {
	for _, file := range rootFiles {
		if fileExists(filepath.Join(dir, file)) {
			return true
		}
	}
	return false
}

func shorten(pathSegment string) string {
	return shortenRegex.ReplaceAllString(pathSegment, "$1")
}

func inDir(path, parent string) bool {
	return strings.HasPrefix(path, parent+"/")
}

func abbrev(path, homeDir string) string {
	if path == homeDir {
		return homeSigil
	}

	inHome := inDir(path, homeDir)
	start := 0
	if inHome {
		start = len(strings.Split(homeDir, pathSeparator))
	}

	segments := strings.Split(path, string(os.PathSeparator))
	endSegment := len(segments) - 1

	var pathSegments []string

	add := func(segment string) {
		pathSegments = append(pathSegments, segment)
	}

	if inHome {
		add(homeSigil)
	}

	for i := start; i < endSegment; i++ {
		dir := strings.Join(segments[:i+1], pathSeparator)

		if sourceRoot(dir) {
			add(segments[i])
		} else {
			add(shorten(segments[i]))
		}
	}
	add(segments[len(segments)-1])

	return strings.Join(pathSegments, pathSeparator)
}

func main() {
	for _, path := range os.Args[1:] {
		fmt.Println(abbrev(path, os.Getenv("HOME")))
	}
}
