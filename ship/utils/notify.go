package utils

import (
	"bufio"
	"errors"
	"io/fs"
	"log"
	"os"
	"strings"
)

type NotifyRule struct {
	Container string
	Pattern   string
}

func LoadNotifyRules(path string) []NotifyRule {
	f, err := os.Open(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("notify rules %q open err: %s", path, err.Error())
		}
		return nil
	}
	defer f.Close()

	var rules []NotifyRule
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var container, pattern string
		if i := strings.Index(line, "|"); i >= 0 {
			container = strings.TrimSpace(line[:i])
			pattern = strings.TrimSpace(line[i+1:])
		} else {
			pattern = line
		}
		if pattern == "" {
			continue
		}
		rules = append(rules, NotifyRule{
			Container: container,
			Pattern:   strings.ToLower(pattern),
		})
	}
	if err := sc.Err(); err != nil {
		log.Printf("notify rules %q read err: %s", path, err.Error())
	}
	return rules
}
