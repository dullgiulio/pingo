// Copyright 2015 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package pingo

import (
	"fmt"
	"math/rand"
	"strings"
)

type meta string

func (h meta) output(key, val string) {
	fmt.Printf("%s: %s: %s\n", string(h), key, val)
}

func (h meta) parse(line string) (key, val string) {
	if line == "" {
		return
	}

	if len(line) < len(string(h)) {
		return
	}

	if line[0:len(string(h))] != string(h) {
		return
	}

	line = line[len(string(h))+2:]
	end := strings.IndexByte(line, ':')
	if end < 0 {
		return
	}

	return line[0:end], line[end+2:]
}

var _letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-")

func randstr(n int) string {
	b := make([]rune, n)
	l := len(_letters)

	for i := range b {
		b[i] = _letters[rand.Intn(l)]
	}

	return string(b)
}
