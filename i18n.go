// Package i18n contains internationalization and location
// modules.
//
// MIT License
//
// Copyright (c) 2016 Angel Del Castillo
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package i18n

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	langs   = make(map[string]string)
	defLang string

	// FuncMap contain all template funcs for integration with html templates.
	FuncMap = template.FuncMap{
		"i18n":  Println,
		"i18nf": Printf,
	}
	mut sync.RWMutex

	errFormatNotValid = errors.New("i18n: language file must contain KEY=VALUE")
)

// Load reads files in directory (skipping subdirs) if file contains language data (KEY=VALUE)
//
// defaultLanguage is used if lang+key is not set.
// separator if empty is (=), only first ocurrence in every line is taken.
// comment symbol if empty is (#).
func Load(dir, defaultLanguage, separator, comment string) error {
	defLang = defaultLanguage
	if separator == "" {
		separator = "="
	}
	if comment == "" {
		comment = "#"
	}
	err := filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
		// skip directories
		if info.IsDir() {
			return nil
		}

		// read language file
		// must be format key=value
		// file name is interpret it as language.
		// it can be Language+Region like es-MX
		lines, err := readLines(name, comment)
		if err != nil {
			return err
		}

		for i := range lines {
			line := lines[i]
			// skip empty lines
			if len(line) < 1 {
				continue
			}
			key, value, err := processLine(line, separator)
			if err != nil {
				// we don't return error here because .DS_Store file is created automatically
				//
				// if buggy we need a rule to skip files later.
				continue
			}
			langs[bullet(info.Name(), key)] = value
		}
		return nil
	})
	return err
}

func readLines(path, commentSymbol string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Text()
		if len(line) < 1 {
			continue
		}
		// skip comments
		if line[:1] == commentSymbol {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scan.Err()
}

// processLine returns key and value if sucessful
//
// If found more than 2 separators (=) takes only the first one.
func processLine(s, separator string) (string, string, error) {
	x := strings.Split(s, separator)
	if len(x) < 2 {
		return "", "", errFormatNotValid
	}
	return x[0], s[len(x[0])+1:], nil
}

// ReutilizeFuncMap takes a Template.FuncMap and adds methods of i18n returning it.
func ReutilizeFuncMap(fnmap template.FuncMap) template.FuncMap {
	for k, val := range FuncMap {
		fnmap[k] = val
	}
	return fnmap
}

// Printf func
func Printf(lang, key string, args ...interface{}) string {
	mut.RLock()
	defer mut.RUnlock()
	slug := bullet(lang, key)
	k, ok := langs[slug]
	if !ok {
		log.Printf("Printf : lang [%s] key [%s] not found", lang, key)
		// try default language (first 2 digits)
		// at this point lang length must be equal or greater than 2, so it's
		// secure accesing it.
		kl, ok := langs[bullet(lang[:2], key)]
		if ok {
			return fmt.Sprintf(kl, args...)
		}

		// try default language
		kdef, ok := langs[bullet(defLang, key)]
		if !ok {
			return key
		}
		return fmt.Sprintf(kdef, args...)
	}
	return fmt.Sprintf(k, args...)
}

// Println func
func Println(lang, key string) string {
	mut.RLock()
	defer mut.RUnlock()
	slug := bullet(lang, key)
	k, ok := langs[slug]
	if !ok {
		// log.Printf("Println : lang [%s] key [%s] not found", lang, key)
		// try default language (first 2 digits)
		// at this point lang length must be equal or greater than 2, so it's
		// secure accesing it.
		kl, ok := langs[bullet(lang[:2], key)]
		if ok {
			return kl
		}

		// try default language
		kdef, ok := langs[bullet(defLang, key)]
		if !ok {
			return key
		}
		return kdef
	}
	return k
}

// bullet we need a format key for map of languages
func bullet(lang, key string) string {
	return cleanLang(lang) + ":" + key
}

func cleanLang(s string) string {
	if len(s) <= 5 {
		return strings.ToLower(s)
	}
	return strings.ToLower(s[:5])
}
