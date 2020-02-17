// Copyright 2020 John Cramb. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jcramb/cedict"
)

func main() {
	d := cedict.New()
	s := strings.Join(os.Args[1:], " ")

	if cedict.IsHanzi(s) {
		fmt.Printf("[input] hanzi\n")

		// convert to pinyin
		fmt.Printf("%s\n", cedict.PinyinTones(d.HanziToPinyin(s)))

	} else {
		fmt.Printf("[input] english \n")

		// search by meaning
		elements := d.GetByMeaning(s)
		for _, e := range elements {
			fmt.Printf("%s\n", e.Marshal())
		}
	}
}
