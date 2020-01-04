package vre

import (
	"regexp"
)

type Input struct {
	cmd     string
	pattern string
	replace *string
	flag    string
	v       int
}

// var parser = regexp.MustCompile(`(.*?[^\\]/)|/`)
var parser = regexp.MustCompile(`([^\\/]|\\.)*`)

func Parse(s string) *Input {
	res := parser.FindAllString(s, -1)
	if len(res) != 3 && len(res) != 4 {
		return nil
	}

	ret := Input{
		cmd:     res[0],
		pattern: res[1],
	}

	if len(res) == 4 {
		o := res[2]
		ret.replace = &o
		ret.flag = res[3]
	} else {
		ret.flag = res[2]
	}

	return &ret
}
