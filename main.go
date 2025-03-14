package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type fileOptions struct {
	path     string
	str      string
	fileType string
	replace  string
}
type config struct {
	options         fileOptions
	withVerbose     bool
	withDryRun      bool
	withInteractive bool
	withRegex       bool
}

func main() {
	cfg := parseFlags()
	if cfg.options.path == "" || cfg.options.str == "" {
		flag.Usage()
		os.Exit(1)
	}

	var pattern *regexp.Regexp
	var err error
	if cfg.withRegex {
		pattern, err = regexp.Compile(cfg.options.str)
		if err != nil {
			fmt.Println("compile pattern:", err)
			os.Exit(1)
		}
	}
	pairs, err := walker(cfg, pattern)
	if err != nil {
		fmt.Println("walk dir:", err)
		os.Exit(2)
	}
	if cfg.withDryRun {
		fmt.Printf("Found %d file(s) to rename!\n", len(pairs))
		if cfg.withVerbose {
			for k, v := range pairs {
				fmt.Printf("%s -> %s\n", k, v)
			}
		}
		return
	}
	if cfg.withInteractive {
		fmt.Printf("Found %d file(s). Proceed?(y/n) ", len(pairs))
		if !canProceed() {
			fmt.Println("Aborted.")
			return
		}
	}

	start := time.Now()
	n, err := rename(pairs)
	if err != nil {
		fmt.Println("Renaming:", err)
		fmt.Printf("%d file(s) were renamed.\n", n)
		os.Exit(2)
	}
	if cfg.withVerbose {
		fmt.Printf("Renamed %d file(s) in %s.\n", n, time.Since(start))
	}
}

func walker(config config, pattern *regexp.Regexp,
) (map[string]string, error) {
	pairs := make(map[string]string)
	err := filepath.WalkDir(
		config.options.path,
		func(path string, file fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case file.IsDir():
				return nil
			}
			oldName := file.Name()
			fileExt := searchFileExtention(file.Name())
			if config.options.fileType != "" && fileExt != "" {
				if fileExt != config.options.fileType {
					return nil
				}
			}
			targetStr := searchString(pattern, config.options.str, oldName)
			if config.withRegex && targetStr == "" {
				return nil
			}

			newName := strings.ReplaceAll(oldName, targetStr, config.options.replace)
			if newName == oldName || newName == "" {
				return nil
			}

			if config.options.replace != "" {
				newName = resolveConflict(filepath.Dir(path), newName, pairs)
			}
			newPath := filepath.Join(filepath.Dir(path), newName)
			if path == newPath {
				return nil
			}
			pairs[path] = newPath
			return nil
		})
	return pairs, err
}

func rename(pairs map[string]string) (uint, error) {
	var renamed uint
	for oldName, newName := range pairs {
		if err := os.Rename(oldName, newName); err != nil {
			return renamed, fmt.Errorf(
				"%q to %q: %w", oldName, newName, err,
			)
		}
		renamed++
	}
	return renamed, nil
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.options.path, "p", "", "path to dir")
	flag.StringVar(&cfg.options.str, "s", "", "string to find")
	flag.StringVar(&cfg.options.fileType, "t", "", "filter file type to modify")
	flag.StringVar(&cfg.options.replace, "replace", "", "replace str instead of remove it")
	flag.BoolVar(&cfg.withVerbose, "v", false, "verbose")
	flag.BoolVar(&cfg.withDryRun, "d", false, "dry run")
	flag.BoolVar(&cfg.withInteractive, "i", false, "interactive")
	flag.BoolVar(&cfg.withRegex, "r", false, "enable regex")
	flag.Parse()
	return cfg
}

func searchString(pattern *regexp.Regexp, str, fileName string) string {
	if pattern == nil {
		return str
	}
	return pattern.FindString(fileName)
}

func searchFileExtention(fileName string) string {
	return filepath.Ext(fileName)
}

func canProceed() bool {
	r := bufio.NewReader(os.Stdin)
	s, err := r.ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func resolveConflict(dir, newName string, pairs map[string]string) string {
	candidate := newName
	count := 1
	for {
		conflict := false
		for _, v := range pairs {
			if filepath.Base(v) == candidate {
				conflict = true
				break
			}
		}
		if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
			conflict = true
		}
		if !conflict {
			break
		}
		ext := filepath.Ext(newName)
		nameOnly := strings.TrimSuffix(newName, ext)
		candidate = fmt.Sprintf("%s_%d%s", nameOnly, count, ext)
		count++
	}
	return candidate
}
