package main

import (
	"errors"
	"strings"
	"unicode"
)

type puzzleRequest struct {
	Text     string `json:"text"`
	Shift    int    `json:"shift,omitempty"`
	Alphabet string `json:"alphabet,omitempty"`
	Mapping  string `json:"mapping,omitempty"`
}

type puzzleResponse struct {
	Result   string `json:"result,omitempty"`
	Values   []int  `json:"values,omitempty"`
	Sum      int    `json:"sum,omitempty"`
	DigitSum int    `json:"digit_sum,omitempty"`
	Root     int    `json:"digital_root,omitempty"`
}

func a1z26(text string) puzzleResponse {
	values := make([]int, 0)
	sum := 0
	for _, r := range strings.ToUpper(text) {
		if r >= 'A' && r <= 'Z' {
			v := int(r-'A') + 1
			values = append(values, v)
			sum += v
		}
	}
	return puzzleResponse{Values: values, Sum: sum}
}

func caesar(text string, shift int) string {
	shift = ((shift % 26) + 26) % 26
	var b strings.Builder
	for _, r := range text {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune('A' + rune((int(r-'A')+shift)%26))
		case r >= 'a' && r <= 'z':
			b.WriteRune('a' + rune((int(r-'a')+shift)%26))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func digitChecksum(text string) (int, int) {
	sum := 0
	for _, r := range text {
		if unicode.IsDigit(r) && r <= '9' {
			sum += int(r - '0')
		}
	}
	root := sum
	for root >= 10 {
		n := 0
		for root > 0 {
			n += root % 10
			root /= 10
		}
		root = n
	}
	return sum, root
}

func substitute(text, alphabet, mapping string) (string, error) {
	alphabet = strings.TrimSpace(alphabet)
	mapping = strings.TrimSpace(mapping)
	if alphabet == "" {
		alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	if mapping == "" {
		return "", errors.New("mapping is required")
	}
	ar := []rune(alphabet)
	mr := []rune(mapping)
	if len(ar) != len(mr) {
		return "", errors.New("alphabet and mapping must have equal length")
	}
	lookup := map[rune]rune{}
	for i, r := range ar {
		lookup[unicode.ToUpper(r)] = mr[i]
	}
	var b strings.Builder
	for _, r := range text {
		upper := unicode.ToUpper(r)
		if v, ok := lookup[upper]; ok {
			if unicode.IsLower(r) {
				v = unicode.ToLower(v)
			}
			b.WriteRune(v)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String(), nil
}
