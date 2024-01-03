// Copyright 2020 John Cramb. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package cedict

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pkg/errors"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (

	// URL of the latest CC-CEDICT data in .tar.gz archive format.
	URL = "https://www.mdbg.net/chinese/export/cedict/cedict_1_0_ts_utf-8_mdbg.txt.gz"

	// LineEnding used by Save(), defaults to "\r\n" to match original content.
	LineEnding = "\r\n"

	// MaxResults determines the most entries returned for any Dict method.
	MaxResults = 50

	// MaxLD controls the max levenshtein distance allowed for matches.
	MaxLD = 10
)

var (
	instance *Dict
	loadOnce sync.Once
)

/*
	todo: look into "github.com/yanyiwu/gojieba"
*/

// Dict represents an instance of the CC-CEDICT entries.
// By default, the latest version will be downloaded on creation.
type Dict struct {
	e      []*Entry
	md     Metadata
	ready  chan bool
	header []string
	mutex  sync.Mutex
	err    error
}

// Entry represents a single entry in the CC-CEDICT dictionary.
type Entry struct {
	Traditional string
	Simplified  string
	Pinyin      string
	Meanings    []string
}

// Metadata represents information embedded in the CC-CEDICT header.
type Metadata struct {
	Version    int
	Subversion int
	Format     string
	Charset    string
	Entries    int
	Publisher  string
	License    string
	Timestamp  time.Time
}

// Parse creates a Dict instance from an io.Reader
// It expects text input in the format, https://cc-cedict.org/wiki/format:syntax
func Parse(r io.Reader) (*Dict, error) {
	d := newDict()
	scanner := bufio.NewScanner(r)

	// scan lines from text input
	for scanner.Scan() {
		line := scanner.Text()

		// is this a comment line?
		if strings.HasPrefix(line, "#") {
			d.header = append(d.header, line)

			// does the line include metadata?
			if strings.HasPrefix(line, "#!") {
				i := strings.Index(line, "=")
				v := line[i+1:]
				k := line[3:i]

				// parse metadata value
				switch k {
				case "version":
					n, err := strconv.Atoi(v)
					if err != nil {
						return nil, errors.Wrap(err, "version: expected number")
					}
					d.md.Version = n

				case "subversion":
					n, err := strconv.Atoi(v)
					if err != nil {
						return nil, errors.Wrap(err, "subversion: expected number")
					}
					d.md.Subversion = n

				case "format":
					d.md.Format = v

				case "charset":
					d.md.Charset = v

				case "entries":
					n, err := strconv.Atoi(v)
					if err != nil {
						return nil, errors.Wrap(err, "entries: expected number")
					}
					d.md.Entries = n

				case "publisher":
					d.md.Publisher = v

				case "license":
					d.md.License = v

				case "date":
					t, err := time.Parse(time.RFC3339, v)
					if err != nil {
						return nil, errors.Wrap(err, "date: expected RFC3339 format")
					}
					d.md.Timestamp = t
				}
			}

			// skip commented lines
			continue
		}

		// add entry to dict
		e := &Entry{}
		if err := e.Unmarshal(line); err != nil {
			return nil, errors.Wrap(err, "unmarshal: "+line)
		}
		d.e = append(d.e, e)
	}

	// validate header entry count
	if len(d.e) != d.md.Entries {
		return nil, fmt.Errorf("loaded entries (%d) != header entries (%d)",
			len(d.e), d.md.Entries)
	}

	// unblock dict methods
	d.setReady()

	return d, nil
}

// Download returns a Dict using the latest CC-CEDICT archive from MDBG.
// This file is regularly updated but relatively small at approx 4MB.
func Download() (io.ReadCloser, error) {

	resp, err := http.Get(URL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return gz, nil
}

// Load returns a Dict loaded from a CC-CEDICT formatted file.
// This is provided for completeness, but I encourage you to
// use default behaviour of downloading the latest dict each time.
func Load(filename string) (*Dict, error) {

	f, err := os.Open(filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer f.Close()

	var r io.Reader = f
	if filepath.Ext(filename) == ".gz" {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		defer gz.Close()
		r = gz
	}

	dict, err := Parse(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dict, nil
}

// New returns a Dict immediately but downloads the latest
// CC-CEDICT data in the background. Dict methods can be
// safely called, but will block until parsing is complete.
func New() *Dict {
	loadOnce.Do(func() {
		instance = newDict()
		go instance.lazyLoad()
	})
	return instance
}

// newDict creates a new Dict struct.
func newDict() *Dict {
	return &Dict{
		ready: make(chan bool),
	}
}

// Err blocks until the Dict is finished parsing and then
// returns any errors encountered during loading/download.
func (d *Dict) Err() error {
	d.lazyLoad()
	return d.err
}

// DefaultFilename returns the CC-CEDICT filename format.
// constructed using the Dict's parsed metadata.
func (d *Dict) DefaultFilename() string {
	d.lazyLoad()
	return fmt.Sprintf("cedict_%d_%d_%s_%s_%s.txt.gz",
		d.md.Version, d.md.Subversion, strings.ToLower(d.md.Format),
		strings.ToLower(d.md.Charset), strings.ToLower(d.md.Publisher))
}

// Save writes the Dict to a file using the format at
// https://cc-cedict.org/wiki/format:syntax, and should
// be identical to the unpacked CC-CEDICT file download.
// Saved as gzip archive if filename ends in '.gz'.
func (d *Dict) Save(filename string) error {
	d.lazyLoad()

	// create file, overwrite if needed
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	// wrap in gzip writer, if requested
	var w io.Writer = f
	if filepath.Ext(filename) == ".gz" {
		gz := gzip.NewWriter(f)
		defer gz.Close()
		w = gz
	}

	// write commented lines
	for i, line := range d.header {
		if i != len(d.header)-1 {
			line += LineEnding
		}
		if _, err := w.Write([]byte(line)); err != nil {
			return errors.WithStack(err)
		}
	}

	// write dict entries
	for _, e := range d.e {
		line := LineEnding + e.Marshal()
		if _, err := w.Write([]byte(line)); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Metadata returns the Dict's metadata parsed from header comments.
func (d *Dict) Metadata() Metadata {
	d.lazyLoad()
	return d.md
}

// GetByHanzi returns the Dict entry for the hanzi, if found.
// Supports input using traditional or simplified characters.
func (d *Dict) GetByHanzi(s string) *Entry {
	d.lazyLoad()
	s = strings.TrimSpace(s)
	for _, e := range d.e {
		if e.Traditional == s || e.Simplified == s {
			return e
		}
	}
	return nil
}

// GetAllByHanzi returns all the Dict entries that match the hanzi.
// Supports input using traditional or simplified characters.
func (d *Dict) GetAllByHanzi(s string) []*Entry {
	d.lazyLoad()
	s = strings.TrimSpace(s)
	var results []*Entry
	for _, e := range d.e {
		if e.Traditional == s || e.Simplified == s {
			results = append(results, e)
		}
	}
	return results
}

// GetByPinyin returns hanzi matching the given pinyin string.
// Supports pinyin in plaintext or with tones/tone numbers.
// With plaintext, all tone variations are considered matching.
func (d *Dict) GetByPinyin(s string) []*Entry {
	d.lazyLoad()

	// convert tones to tone numbers
	s = PinyinToneNums(s)
	isPlaintext := strings.IndexAny(s, toneNums) < 0

	// normalise pinyin to lowercase, no spaces
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")

	var results []*Entry
	for _, e := range d.e {

		// normalise entry pinyin to lowercase, no spaces
		p := strings.ToLower(e.Pinyin)
		p = strings.ReplaceAll(p, " ", "")

		// if input is plaintext, remove tone numbers from entry
		if isPlaintext {
			p = StripDigits(p)
		}

		// add matching pinyin entries
		if p == s {
			results = append(results, e)
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Pinyin < results[j].Pinyin
	})

	return results
}

// GetByMeaning returns entries containing the specified meaning.
// Matching is not case-sensitive and can be exact/non-exact.
func (d *Dict) GetByMeaning(s string) []*Entry {
	d.lazyLoad()

	// normalise input to lowercase
	s = strings.ToLower(s)

	var results []*Entry
	lev := make(map[*Entry]int)
nextEntry:
	for _, e := range d.e {
		for _, m := range e.Meanings {

			// normalise entry to lowercase
			m = strings.ToLower(m)

			// check if meaning matches
			if strings.Contains(s, m) {
				ld := levenshtein(s, m)

				// discard matches too far from input
				if ld <= MaxLD {
					lev[e] = ld
					results = append(results, e)
					continue nextEntry
				}
			}
		}
	}

	// sort by levenshtein distance
	sort.SliceStable(results, func(i, j int) bool {
		return lev[results[i]] < lev[results[j]]
	})

	// limit results returned
	if len(results) > MaxResults {
		results = results[:MaxResults]
	}

	return results
}

// HanziToPinyin converts hanzi to their pinyin representation.
// It implements greedy matching for longest character combos.
func (d *Dict) HanziToPinyin(s string) string {
	d.lazyLoad()

	// handle early exit
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}

	// hanzi to latin symbols
	s = ConvertSymbols(s)

	// iterate through possible word combos
	p := ""
	runes := []rune(s)
	for i := 0; i < len(runes); {

		// skip non-hanzi characters
		if !unicode.In(runes[i], unicode.Han) {
			for ; i < len(runes) && !unicode.In(runes[i], unicode.Han); i++ {
				p += string(runes[i])
			}
			p += " "
			continue
		}

		// try to match longest hanzi combo to entry
		found := false
		for j := len(runes); j > i; j-- {
			han := string(runes[i:j])
			e := d.GetByHanzi(han)
			if e != nil {
				i = j
				found = true
				p += e.Pinyin + " "
				break
			}
		}

		// we didn't find it, just add it as-is
		if !found {
			p += string(runes[i])
			i++
		}
	}

	// todo: check how this interacts with uppercase tones?
	return strings.ToUpper(p[:1]) + strings.ToLower(strings.TrimSpace(p[1:]))
}

// lazyLoad is used as a blocking barrier to ensure methods
// are only executed after Dict is populated. If needed, it
// will trigger the download and parsing of the CC-CEDICT.
func (d *Dict) lazyLoad() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if !d.isReady() {

		// download latest CC-CEDICT
		r, err := Download()
		if err != nil {
			d.err = errors.WithStack(err)
			return
		}

		// parse metadata + entries
		dict, err := Parse(r)
		if err != nil {
			d.err = errors.WithStack(err)
			return
		}

		// populate dict
		d.e = dict.e
		d.md = dict.md
		d.header = dict.header

		// unblock methods
		d.setReady()
	}
}

// isReady returns true if Dict is populated
func (d *Dict) isReady() bool {
	select {
	case <-d.ready:
		return true
	default:
		return false
	}
}

// setReady unblocks lazyLoad calls
func (d *Dict) setReady() {
	if !d.isReady() {
		close(d.ready)
	}
}

// Marshal returns the entry, formatted according to
// https://cc-cedict.org/wiki/format:syntax
func (e *Entry) Marshal() string {
	return fmt.Sprintf("%s %s [%s] %s",
		e.Traditional, e.Simplified, e.Pinyin,
		"/"+strings.Join(e.Meanings, "/")+"/",
	)
}

// Unmarshal populates the entry, from input text formatted
// according to https://cc-cedict.org/wiki/format:syntax
func (e *Entry) Unmarshal(s string) error {

	// parse pinyin and meanings
	fields := strings.Split(s, "/")
	off := strings.Index(fields[0], "[")
	end := strings.Index(fields[0], "]")
	if off < 0 || end < 0 {
		return errors.New("expected '[pinyin]' format")
	}
	chars := fields[0][:off]
	pinyin := fields[0][off+1 : end]

	// 龍豆 龙豆 [long2 dou4] /dragon bean/long bean/
	var trad, sim string
	n, err := fmt.Sscanf(chars, "%s %s ", &trad, &sim)
	if err != nil {
		return errors.WithStack(err)
	} else if n != 2 {
		return errors.New("expected two hanzi fields i.e. '龍豆 龙豆 '")
	}

	// set entry data
	e.Traditional = trad
	e.Simplified = sim
	e.Pinyin = pinyin
	e.Meanings = fields[1 : len(fields)-1]

	return nil
}

// IsHanzi returns true if the string contains only han characters.
// http://www.unicode.org/reports/tr38/tr38-27.html HAN Unification
func IsHanzi(s string) bool {
	for _, r := range []rune(s) {
		_, isSymbol := symbols[r]
		if !unicode.Is(unicode.Han, r) && !isSymbol {
			return false
		}
	}
	return true
}

// StripDigits returns the string with all unicode digits removed.
func StripDigits(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.Predicate(unicode.IsDigit)), norm.NFC)
	s, _, _ = transform.String(t, s)
	return s
}

// StripTones returns the string with all (mark, nonspacing) removed.
func StripTones(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ = transform.String(t, s)
	return s
}

// ConvertSymbols replaces common hanzi symbols with latin symbols.
func ConvertSymbols(s string) string {
	result := ""
	for _, r := range []rune(s) {
		if sym, ok := symbols[r]; ok {
			result += sym
		} else {
			result += string(r)
		}
	}
	return result
}

// PinyinPlaintext returns pinyin string without tones or tone numbers.
func PinyinPlaintext(s string) string {
	return StripTones(StripDigits(s))
}

// PinyinToneNums returns pinyin string converting tones to tone numbers.
func PinyinToneNums(s string) string {
	result := ""
	for _, w := range strings.Split(s, " ") {
		tone := ""
		for _, r := range w {
			m := mapToneToNum[r]
			if m != "" {
				result += m[:len(m)-1]
				tone = strings.TrimSpace(m[len(m)-1:])
			} else {
				result += string(r)
			}
		}
		result += tone + " "
	}
	return strings.TrimSpace(result)
}

// PinyinTones returns pinyin string converting tone numbers to tones.
// It supports both CC-CEDICT format, with tones at the end of syllables
// i.e. Zhong1 wen2, as well as inline format with tones after their
// respective character i.e. Zho1ng we2n.
func PinyinTones(s string) string {

	// convert u: into single rune ü
	s = strings.ReplaceAll(s, "u:", "ü")

	result := ""
	for _, w := range strings.Split(s, " ") {

		// find rune to apply tone to
		i := guessToneIndex(w)

		if i < 0 {
			result += w + " "
			continue
		}

		// todo: does this need to be done
		numIndex := strings.IndexAny(w, toneNums)
		if numIndex < 0 {
			result += w + " "
			continue
		}

		tone, _ := strconv.Atoi(string(w[numIndex]))
		tone--
		if tone < 0 || tone >= len(mapNumToTone) {
			result += w + " "
			continue
		}

		w = w[:numIndex] + w[numIndex+1:]
		runes := []rune(w)
		k := runes[i]
		result += string(runes[:i])
		result += string(mapNumToTone[k][tone])
		result += string(runes[i+1:]) + " "
	}
	return strings.TrimSpace(result)
}

// FixSymbolSpaces removes spaces added by HanziToPinyin
// conversion and makes the string look more natural.
func FixSymbolSpaces(s string) string {
	s = strings.ReplaceAll(s, " ?", "?")
	s = strings.ReplaceAll(s, " .", ".")
	s = strings.ReplaceAll(s, " !", "!")
	s = strings.ReplaceAll(s, " :", ":")
	s = strings.ReplaceAll(s, " ;", ";")
	s = strings.ReplaceAll(s, " ,", ",")
	s = strings.ReplaceAll(s, "[ ", "[")
	s = strings.ReplaceAll(s, " ]", "]")
	s = strings.ReplaceAll(s, "( ", "(")
	s = strings.ReplaceAll(s, " )", ")")
	return s
}

// guessToneIndex returns an index of the highest priority
// vowel in the string, as a guess for which gets the tone.
func guessToneIndex(s string) int {
	for _, r := range vowels {
		byteIndex := strings.IndexRune(s, r)
		if byteIndex >= 0 {
			runeIndex := len([]rune(s[0:byteIndex]))
			return runeIndex
		}
	}
	return -1
}

// levenshtein calculates the Levenshtein distance (LD), which is a measure
// of the similarity between two strings. The distance is the number of
// deletions, insertions, or substitutions required to transform s1 into s2.
//
// Adapted from https://github.com/agnivade/levenshtein
// The MIT License (MIT) Copyright (c) 2015 Agniva De Sarker
func levenshtein(src, dst string) int {
	if src == dst {
		return 0
	}
	s1 := []rune(src)
	s2 := []rune(dst)
	if len(src) > len(dst) {
		s1, s2 = s2, s1
	}
	l1 := len(s1)
	l2 := len(s2)
	ld := make([]int, l1+1)
	for i := 0; i < len(ld); i++ {
		ld[i] = i
	}
	_ = ld[l1]
	for i := 1; i <= l2; i++ {
		prev := i
		curr := 0
		for j := 1; j <= l1; j++ {
			if s2[i-1] == s1[j-1] {
				curr = ld[j-1]
			} else {
				curr = min(ld[j-1]+1, prev+1, ld[j]+1)
			}
			ld[j-1] = prev
			prev = curr
		}
		ld[l1] = prev
	}
	return ld[l1]
}

// min returns the minimum of three int inputs
func min(x, y, z int) int {
	if x < y {
		if x < z {
			return x
		}
	} else if y < z {
		return y
	}
	return z
}

var vowels = "AaEeiOouür"

var toneNums = "12345"

var mapNumToTone = map[rune][]rune{
	'A': []rune("ĀÁǍÀA"),
	'a': []rune("āáǎàa"),
	'E': []rune("ĒÉĚÈE"),
	'e': []rune("ēéěèe"),
	'i': []rune("īíǐìi"),
	'O': []rune("ŌÓǑÒO"),
	'o': []rune("ōóǒòo"),
	'u': []rune("ūúǔùu"),
	'ü': []rune("ǖǘǚǜü"),
	'r': []rune("rrrrr"),
}

var mapToneToNum = map[rune]string{
	'Ā': "A1",
	'Á': "A2",
	'Ǎ': "A3",
	'À': "A4",
	'ā': "a1",
	'á': "a2",
	'ǎ': "a3",
	'à': "a4",
	'Ē': "E1",
	'É': "E2",
	'Ě': "E3",
	'È': "E4",
	'ē': "e1",
	'é': "e2",
	'ě': "e3",
	'è': "e4",
	'ī': "i1",
	'í': "i2",
	'ǐ': "i3",
	'ì': "i4",
	'Ō': "O1",
	'Ó': "O2",
	'Ǒ': "O3",
	'Ò': "O4",
	'ō': "o1",
	'ó': "o2",
	'ǒ': "o3",
	'ò': "o4",
	'ū': "u1",
	'ú': "u2",
	'ǔ': "u3",
	'ù': "u4",
	'ü': "u: ",
	'ǖ': "u:1",
	'ǘ': "u:2",
	'ǚ': "u:3",
	'ǜ': "u:4",
}

var symbols = map[rune]string{
	'？': "?",
	'！': "!",
	'：': ":",
	'。': ".",
	'・': ".",
	'，': ",",
	'；': ";",
	'（': "(",
	'）': ")",
	'【': "[",
	'】': "]",
}
