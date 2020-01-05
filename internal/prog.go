package vre

import (
	"regexp"
	"strings"
)

type Input struct {
	cmd     string
	pattern string
	replace *string
	flag    string
}

var parser = regexp.MustCompile(`([^\\/]|\\.)*`)

func Parse(s string) *Input {
	res := parser.FindAllString(s, -1)
	if len(res) != 3 && len(res) != 4 {
		return nil
	}
	if res[1] == "" {
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

type Prog struct {
	re      *regexp.Regexp
	replace *string
	n       int
}

func NewProg(s string) *Prog {
	i := Parse(s)
	if i == nil {
		return nil
	}

	ret := Prog{}

	// replace escaped \/ in pattern with just /
	re, e := regexp.Compile(strings.ReplaceAll(i.pattern, `\/`, `/`))
	if e != nil {
		return nil
	}

	ret.re = re
	ret.replace = i.replace
	ret.n = 1

	return &ret
}

func (p *Prog) Find(s []byte) [][]int {
	return p.re.FindAllIndex(s, p.n)
}

// Replace returns (in order) the indices of the matches in the original
// string, the indices of the replacements in the new string, and the new string
func (p *Prog) Replace(s []byte) ([][]int, [][]int, []byte) {
	if p.replace == nil {
		panic("Shouldn't be here")
	}

	res := []byte{}
	submatches := p.re.FindAllSubmatchIndex(s, p.n)
	nbounds := make([][]int, 0)
	prev := 0

	for _, submatch := range submatches {
		res = append(res, s[prev:submatch[0]]...)
		old := len(res)
		res = p.re.Expand(res, []byte(*p.replace), s, submatch)
		nbounds = append(nbounds, []int{old, len(res)})

		prev = submatch[1]
	}

	res = append(res, s[prev:len(s)]...)

	return submatches, nbounds, res
}
