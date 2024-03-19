# `cedict` Chinese-English Dictionary Go Package - 漢英詞典Go軟件包 - 汉英词典Go软件包

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/Ecostack/cedict)
[![Go Report Card](https://goreportcard.com/badge/github.com/Ecostack/cedict?style=flat-square)](https://goreportcard.com/report/github.com/Ecostack/cedict)
[![Coveralls](https://img.shields.io/coveralls/github/Ecostack/cedict/master?style=flat-square)](https://coveralls.io/github/Ecostack/cedict)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE) 

# Overview 

Golang library for the community maintained Chinese-English dictionary (CC-CEDICT), published by MDBG. 

> [https://www.mdbg.net/chinese/dictionary?page=cedict](https://www.mdbg.net/chinese/dictionary?page=cedict)

The basic format of a CEDICT entry is:

    Traditional Simplified [pin1 yin1] /American English equivalent 1/equivalent 2/
    漢字 汉字 [han4 zi4] /Chinese character/CL:個|个/


# Install
First grab the latest version of the package,

    go get -u github.com/Ecostack/cedict

Next, include it in your application:

```go
import "github.com/Ecostack/cedict"
```

# Getting Started

## Hanzi to Pinyin

```go
package main

import (
	"fmt"
	"log"
	"github.com/Ecostack/cedict" // replace with your module path
)

func main() {
	dict := cedict.New()

	pinyin := dict.HanziToPinyin("龙豆")
	fmt.Println("Pinyin for 龙豆 is:", pinyin)
}
```

## Finding Entry by Hanzi

```go
package main

import (
	"fmt"
	"log"
	"github.com/Ecostack/cedict" // replace with your module path
)

func main() {
	dict := cedict.New()
	
	entry := dict.GetByHanzi("龙")
	if entry != nil {
		fmt.Printf("Found entry for 龙: %+v\n", entry)
	} else {
		fmt.Println("Entry for 龙 not found.")
	}
}
```

## Finding Entries by Hanzi

```go
package main

import (
	"fmt"
	"log"
	"github.com/Ecostack/cedict" // replace with your module path
)

func main() {
	dict := cedict.New()

	entries := dict.GetAllByHanzi("龙")
	for _, entry := range entries {
        fmt.Printf("Found entry: %+v\n", entry)
    }
}
```

## Searching by Pinyin

```go
package main

import (
	"fmt"
	"log"
	"github.com/Ecostack/cedict" // replace with your module path
)

func main() {
	dict := cedict.New()

	entries := dict.GetByPinyin("long2")
	for _, entry := range entries {
		fmt.Printf("Found entry: %+v\n", entry)
	}
}
```

# Contributing

1. Fork the repo
2. Clone your fork (`git clone https://github.com/<username>/cedict && cd cedict`)
3. Create your own branch (`git checkout -b my-patch`)
4. Make changes and add them (`git add .`)
5. Commit your changes (`git commit -m 'Fixed #123'`)
6. Push to the branch (`git push origin my-patch`)
7. Create new pull request

## License

Copyright 2020 John Cramb. All rights reserved.
Copyright 2024 Sebastian Scheibe. All rights reserved.

Licensed under the MIT License. See [LICENSE](https://github.com/Ecostack/cedict/blob/master/LICENSE) in the project root for license information.

