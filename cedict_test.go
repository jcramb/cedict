// Copyright 2020 John Cramb. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package cedict

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	testDir = "./testdata"
)

func TestLoadSave(t *testing.T) {

	// cleanup test data
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)

	// create dict
	d := New()
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}

	// display metadata, validate loaded entries
	md := d.Metadata()
	t.Logf("CEDICT: %s\n", md.Timestamp.Format(time.RFC3339))
	t.Logf("Loaded %d entries.\n", md.Entries)

	// save dict to file
	filename := filepath.Join(testDir, d.DefaultFilename())
	if err := d.Save(filename); err != nil {
		t.Fatal(err)
	}

	// load dict from file
	dict, err := Load(filename)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// confirm same metadata
	if d.Metadata() != dict.Metadata() {
		t.Fatalf("metadata mismatch")
	}

	// sanity check metadata
	if md.Publisher != "MDBG" {
		t.Fatalf("publisher != MDBG")
	}
}

func TestHanzi(t *testing.T) {
	d := New()

	if IsHanzi("我的大王！") != true {
		t.Errorf("我的大王！ is hanzi")
	}

	e := d.GetByHanzi("中文")
	if e != nil {
		t.Logf("ByHanzi:   %s\n", e.Marshal())
	} else if e.Meanings[0] != "Chinese language" {
		t.Fail()
	}
}

func TestPinyin(t *testing.T) {
	d := New()

	check := func(n int, in, trad, pinyin string) {
		elements := d.GetByPinyin(in)
		if n > 0 && len(elements) != n {
			for _, e := range elements {
				t.Logf("ByPinyin:  %s\n", e.Marshal())
			}
			t.Errorf("got %d (want %d) - '%s'", len(elements), n, in)
			return
		}
		if n == 0 {
			return
		}
		if len(elements) < 1 {
			t.Errorf("got 0 (want > 0) - '%s'", in)
			return
		}
		for _, e := range elements {
			t.Logf("ByPinyin:  %s\n", e.Marshal())
		}
		if trad != "" && elements[0].Traditional != trad {
			t.Errorf("'%s' - got '%s' (want '%s')", in, elements[0].Traditional, trad)
		}
		if pinyin != "" && elements[0].Pinyin != pinyin {
			t.Errorf("'%s' - got '%s' (want '%s')", in, elements[0].Pinyin, pinyin)
		}
	}

	check(1, "zhongwen", "中文", "Zhong1 wen2")
	check(1, "ZHONG WEN", "中文", "Zhong1 wen2")
	check(1, "mei guo ren", "美國人", "Mei3 guo2 ren2")
	check(1, "mei3 guo2 ren2", "美國人", "Mei3 guo2 ren2")
	check(0, "mei1 guo2 ren2", "", "")
	check(0, "mei3 guo ren2", "", "")
	check(0, "zhong6", "", "")
	check(0, "zhong0", "", "")

	numToTone := map[string]string{
		"AaEeiOouü1":     "ĀaEeiOouü",
		"zaEeiOouü2":     "záEeiOouü",
		"zzEeiOouü3":     "zzĚeiOouü",
		"zzzeiOouü4":     "zzzèiOouü",
		"zzzziOouü1":     "zzzzīOouü",
		"zzzzzOouü2":     "zzzzzÓouü",
		"zzzzzzouü3":     "zzzzzzǒuü",
		"zzzzzzzuü4":     "zzzzzzzùü",
		"zzzzzzzzü1":     "zzzzzzzzǖ",
		"Zhong1 wen2":    "Zhōng wén",
		"zhong1 Wen2":    "zhōng Wén",
		"Zho1ng we2n":    "Zhōng wén",
		"Ni3 hao2 ma5":   "Nǐ háo ma",
		"Mei3 guo2 ren2": "Měi guó rén",
		"Me3i guo2 re2n": "Měi guó rén",
	}

	toneToNum := map[string]string{
		"üz zǖz zü":   "u:z zu:z1 zu:",
		"Zhōng wén":   "Zhong1 wen2",
		"zhōng Wén":   "zhong1 Wen2",
		"Nǐ háo ma":   "Ni3 hao2 ma", // 5?
		"Měi guó rén": "Mei3 guo2 ren2",
	}

	for withNum, withTones := range numToTone {
		s := PinyinTones(withNum)
		if s != withTones {
			t.Errorf("got '%s' (want '%s')", s, withTones)
		}
	}
	for withTones, withNum := range toneToNum {
		s := PinyinToneNums(withTones)
		if s != withNum {
			t.Errorf("got '%s' (want '%s')", s, withNum)
		}
	}
}

func TestMeaning(t *testing.T) {
	d := New()
	elements := d.GetByMeaning("Chinese Language")
	for _, e := range elements[:1] {
		t.Logf("ByMeaning: %s\n", e.Marshal())
	}
	if len(elements) < 1 || elements[0].Pinyin != "Zhong1 wen2" {
		t.Fail()
	}
}

func TestMetadata(t *testing.T) {
	s := `# CC-CEDICT
# Community maintained free Chinese-English dictionary.
#
# Published by MDBG
#
# License:
# Creative Commons Attribution-ShareAlike 4.0 International License
# https://creativecommons.org/licenses/by-sa/4.0/
#
# Referenced works:
# CEDICT - Copyright (C) 1997, 1998 Paul Andrew Denisowski
#
# CC-CEDICT can be downloaded from:
# https://www.mdbg.net/chinese/dictionary?page=cc-cedict
#
# Additions and corrections can be sent through:
# https://cc-cedict.org/editor/editor.php
#
# For more information about CC-CEDICT see:
# https://cc-cedict.org/wiki/
#
#! version=123
#! subversion=456
#! format=ts
#! charset=UTF-8
#! entries=4
#! publisher=MDBG
#! license=https://creativecommons.org/licenses/by-sa/4.0/
#! date=2020-02-14T06:15:46Z
#! time=1581660946
% % [pa1] /percent (Tw)/
21三體綜合症 21三体综合症 [er4 shi2 yi1 san1 ti3 zong1 he2 zheng4] /trisomy/Down's syndrome/
3C 3C [san1 C] /abbr. for computers, communications, and consumer electronics/China Compulsory Certificate (CCC)/
3P 3P [san1 P] /(slang) threesome/`
	r := strings.NewReader(s)
	d, err := Parse(r)
	if err != nil {
		t.Fatal(err)
	}
	md := d.Metadata()
	if md.Version != 123 {
		t.Errorf("version != 123")
	}
	if md.Subversion != 456 {
		t.Errorf("subversion != 456")
	}
	if md.Format != "ts" {
		t.Errorf("format != ts")
	}
	if md.Charset != "UTF-8" {
		t.Errorf("charset != UTF-8")
	}
	if md.Entries != 4 {
		t.Errorf("entries != 4")
	}
	if md.Publisher != "MDBG" {
		t.Errorf("publisher != MDBG")
	}
	if md.License != "https://creativecommons.org/licenses/by-sa/4.0/" {
		t.Errorf("license != https://creativecommons.org/licenses/by-sa/4.0/")
	}
	if md.Timestamp.Format(time.RFC3339) != "2020-02-14T06:15:46Z" {
		t.Errorf("date != 2020-02-14T06:15:46Z")
	}
	if md.Timestamp.Unix() != 1581660946 {
		t.Errorf("time != 1581660946")
	}
}

func TestLoadInvalid(t *testing.T) {
	tests := map[string]string{
		"#! version=\n":                 "expected number",
		"#! subversion=abc\n":           "expected number",
		"#! entries=a1 \n":              "expected number",
		"#! date=2020-0214T06:15:46Z\n": "expected RFC3339",
		"#! entries=1\n":                "loaded entries (0)",
		"% % [pa1 /percent (Tw)/\n":     "unmarshal: ",
	}
	for s, wantErr := range tests {
		r := strings.NewReader(s)
		d, err := Parse(r)
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			var md Metadata
			if d != nil {
				md = d.Metadata()
			}
			t.Errorf("got '%v', want '%s'\n%v", err, wantErr, md)
		}
	}
}

func TestEntry(t *testing.T) {

	equal := func(s string, e *Entry) error {
		o := &Entry{}
		if err := o.Unmarshal(s); err != nil {
			return err
		}
		if o.Traditional != e.Traditional {
			return fmt.Errorf("'%s' != '%s'", o.Traditional, e.Traditional)
		}
		if o.Simplified != e.Simplified {
			return fmt.Errorf("'%s' != '%s'", o.Simplified, e.Simplified)
		}
		if o.Pinyin != e.Pinyin {
			return fmt.Errorf("'%s' != '%s'", o.Pinyin, e.Pinyin)
		}
		for i := range o.Meanings {
			if o.Meanings[i] != e.Meanings[i] {
				return fmt.Errorf("[%d] '%s' != '%s'", i, o.Meanings[i], e.Meanings[i])
			}
		}
		return nil
	}

	if err := equal("中 中 [Zhong1] /China/Chinese/surname Zhong/", &Entry{
		Traditional: "中",
		Simplified:  "中",
		Pinyin:      "Zhong1",
		Meanings: []string{
			"China", "Chinese", "surname Zhong",
		},
	}); err != nil {
		t.Error(err)
	}

	if err := equal("中國人 中国人 [Zhong1 guo2 ren2] /Chinese person/", &Entry{
		Traditional: "中國人",
		Simplified:  "中国人",
		Pinyin:      "Zhong1 guo2 ren2",
		Meanings: []string{
			"Chinese person",
		},
	}); err != nil {
		t.Error(err)
	}

	if err := equal("美國人 美国人 [Mei3 guo2 ren2] /American/American person/American people/CL:個|个[ge4]/", &Entry{
		Traditional: "美國人",
		Simplified:  "美国人",
		Pinyin:      "Mei3 guo2 ren2",
		Meanings: []string{
			"American", "American person", "American people", "CL:個|个[ge4]",
		},
	}); err != nil {
		t.Error(err)
	}
}

func TestHanziToPinyin(t *testing.T) {
	tests := map[string]string{
		"":   "",
		"  ": "",
		"人民银行旁边一行人abc字母【路牌】，平行宇宙发行股票。": "Rén mín yín háng páng biān yī xíng rén abc zì mǔ [lù pái], píng xíng yǔ zhòu fā xíng gǔ piào.",
		"我的大王！": "Wǒ de dà wáng!",
		//"你好。我饿了。":       "Nǐ hǎo. Wǒ èle.",
		"地址：重庆市江北区重工业？": "Dì zhǐ: chóng qìng shì jiāng běi qū zhòng gōng yè?",
		//"abc123": "abc123",
		//"*123*人": "*123* Rén",
		//"` 1234 ~!@# $% ^&* ()_+": "` 1234 ~!@# $% ^&* ()_+",
		//"打印A-4纸。": "Dǎyìn A-4 zhǐ.",
		//"测试zZz你好123": "Cèshì zZz nǐ hǎo 123",
	}
	d := New()
	for s, want := range tests {
		got := FixSymbolSpaces(PinyinTones(d.HanziToPinyin(s)))
		if got != want {
			t.Errorf("\ngot:  '%s'\nwant: '%s'\n", got, want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		src, dst string
		want     int
	}{
		{"", "中文老師", 4},
		{"中文老師", "", 4},
		{"中文老師", "中文老師", 0},
		{"中文", "中國", 1},
		{"中國人", "美國人", 1},
		{"小籠包", "人", 3},
		{"I like learning chinese.", "Do you like learning chinese?", 7},
		{"Wǒ xǐhuān xuéxí zhōngwén.", "Nǐ xǐhuān xué zhōngwén ma?", 8},
		{"我喜欢学习中文。", "你喜欢学中文吗？", 4},
		{"我喜歡學習中文。", "你喜歡學中文嗎？", 4},
	}
	for i, test := range tests {
		if n := levenshtein(test.src, test.dst); n != test.want {
			t.Errorf("Test[%d]: levenshtein(%q,%q) got %v, want %v",
				i, test.src, test.dst, n, test.want)
		}
	}
}

func ExampleDict_getByPinyin() {
	d := New()
	elements := d.GetByPinyin("mei guo ren")
	for _, e := range elements {
		fmt.Printf("%s - %s\n", e.Traditional, FixSymbolSpaces(PinyinTones(e.Pinyin)))
	}
	// Output:
	// 美國人 - Měi guó rén
}

func ExampleDict_hanziToPinyin() {
	d := New()
	hans := "你喜歡學中文嗎？"
	fmt.Printf("%s (plaintext) '%s'\n", hans, PinyinPlaintext(d.HanziToPinyin(hans)))
	fmt.Printf("%s (tonenums)  '%s'\n", hans, d.HanziToPinyin(hans))
	fmt.Printf("%s (tones)     '%s'\n", hans, FixSymbolSpaces(PinyinTones(d.HanziToPinyin(hans))))
	// Output:
	// 你喜歡學中文嗎？ (plaintext) 'Ni xi huan xue zhong wen ma ?'
	// 你喜歡學中文嗎？ (tonenums)  'Ni3 xi3 huan5 xue2 zhong1 wen2 ma3 ?'
	// 你喜歡學中文嗎？ (tones)     'Nǐ xǐ huan xué zhōng wén mǎ?'
}

func BenchmarkHanziToPinyin(b *testing.B) {
	d := New()
	tests := []struct {
		label string
		s     string
	}{
		{"HanziToPinyin", "中國人"},
	}
	for _, test := range tests {
		b.Run(test.label, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				d.HanziToPinyin(test.s)
			}
		})
	}
}

func BenchmarkLevenshtein(b *testing.B) {
	tests := []struct {
		label    string
		src, dst string
	}{
		{"English", "I like learning chinese.", "Do you like learning chinese?"},
		{"Pinyin", "Wǒ xǐhuān xuéxí zhōngwén.", "Nǐ xǐhuān xué zhōngwén ma?"},
		{"Simplified", "我喜欢学习中文。", "你喜欢学中文吗？"},
		{"Traditional", "我喜歡學習中文。", "你喜歡學中文嗎？"},
	}
	for _, test := range tests {
		b.Run(test.label, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				levenshtein(test.src, test.dst)
			}
		})
	}
}
