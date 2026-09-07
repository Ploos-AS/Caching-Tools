package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type puzzleRequest struct {
	Text     string `json:"text"`
	Shift    int    `json:"shift,omitempty"`
	Alphabet string `json:"alphabet,omitempty"`
	Mapping  string `json:"mapping,omitempty"`
	FromBase int    `json:"from_base,omitempty"`
	ToBase   int    `json:"to_base,omitempty"`
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

func rot47(text string) string {
	var b strings.Builder
	for _, r := range text {
		if r >= 33 && r <= 126 { b.WriteRune(33 + (r-33+47)%94) } else { b.WriteRune(r) }
	}
	return b.String()
}

func digitChecksum(text string) (int, int) {
	sum := 0
	for _, r := range text {
		if unicode.IsDigit(r) && r <= '9' { sum += int(r - '0') }
	}
	root := sum
	for root >= 10 { n := 0; for root > 0 { n += root % 10; root /= 10 }; root = n }
	return sum, root
}

func substitute(text, alphabet, mapping string) (string, error) {
	alphabet = strings.TrimSpace(alphabet); mapping = strings.TrimSpace(mapping)
	if alphabet == "" { alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ" }
	if mapping == "" { return "", errors.New("mapping is required") }
	ar, mr := []rune(alphabet), []rune(mapping)
	if len(ar) != len(mr) { return "", errors.New("alphabet and mapping must have equal length") }
	lookup := map[rune]rune{}
	for i, r := range ar { lookup[unicode.ToUpper(r)] = mr[i] }
	var b strings.Builder
	for _, r := range text { upper := unicode.ToUpper(r); if v, ok := lookup[upper]; ok { if unicode.IsLower(r) { v = unicode.ToLower(v) }; b.WriteRune(v) } else { b.WriteRune(r) } }
	return b.String(), nil
}

var morseEncode = map[rune]string{
	'A': ".-", 'B': "-...", 'C': "-.-.", 'D': "-..", 'E': ".", 'F': "..-.", 'G': "--.", 'H': "....", 'I': "..", 'J': ".---", 'K': "-.-", 'L': ".-..", 'M': "--", 'N': "-.", 'O': "---", 'P': ".--.", 'Q': "--.-", 'R': ".-.", 'S': "...", 'T': "-", 'U': "..-", 'V': "...-", 'W': ".--", 'X': "-..-", 'Y': "-.--", 'Z': "--..",
	'0': "-----", '1': ".----", '2': "..---", '3': "...--", '4': "....-", '5': ".....", '6': "-....", '7': "--...", '8': "---..", '9': "----.",
}

func morse(text string, decode bool) (string, error) {
	if !decode {
		words := strings.Fields(strings.ToUpper(text)); out := make([]string, 0, len(words))
		for _, word := range words { chars := make([]string, 0, len(word)); for _, r := range word { code, ok := morseEncode[r]; if !ok { return "", fmt.Errorf("unsupported Morse character %q", r) }; chars = append(chars, code) }; out = append(out, strings.Join(chars, " ")) }
		return strings.Join(out, " / "), nil
	}
	decodeMap := map[string]rune{}; for r, code := range morseEncode { decodeMap[code] = r }
	words := strings.Split(strings.TrimSpace(text), "/"); decoded := make([]string, 0, len(words))
	for _, word := range words { var b strings.Builder; for _, code := range strings.Fields(word) { r, ok := decodeMap[code]; if !ok { return "", fmt.Errorf("unknown Morse code %q", code) }; b.WriteRune(r) }; decoded = append(decoded, b.String()) }
	return strings.Join(decoded, " "), nil
}

func bacon(text string, decode bool) (string, error) {
	if !decode {
		out := make([]string, 0)
		for _, r := range strings.ToUpper(text) { if r < 'A' || r > 'Z' { continue }; n := int(r-'A'); var g [5]byte; for i:=4;i>=0;i-- { if n&1 == 1 { g[i]='B' } else { g[i]='A' }; n >>= 1 }; out = append(out, string(g[:])) }
		return strings.Join(out, " "), nil
	}
	clean := strings.NewReplacer(" ", "", "\n", "", "\t", "").Replace(strings.ToUpper(text)); if len(clean)%5 != 0 { return "", errors.New("Bacon input length must be a multiple of 5") }
	var b strings.Builder
	for i:=0;i<len(clean);i+=5 { n:=0; for _, r := range clean[i:i+5] { n <<= 1; if r=='B' { n++ } else if r!='A' { return "", errors.New("Bacon input must contain only A and B") } }; if n > 25 { return "", errors.New("Bacon group is outside A-Z") }; b.WriteByte(byte('A'+n)) }
	return b.String(), nil
}

func convertBase(text string, fromBase, toBase int) (string, error) {
	if fromBase < 2 || fromBase > 36 || toBase < 2 || toBase > 36 { return "", errors.New("bases must be between 2 and 36") }
	parts := strings.Fields(strings.TrimSpace(text)); if len(parts)==0 { return "", errors.New("value is required") }
	out := make([]string, 0, len(parts)); for _, p := range parts { n, err := strconv.ParseInt(p, fromBase, 64); if err != nil { return "", fmt.Errorf("invalid base-%d value %q", fromBase, p) }; out = append(out, strings.ToUpper(strconv.FormatInt(n, toBase))) }
	return strings.Join(out, " "), nil
}

func phoneKeypad(text string) puzzleResponse {
	values := make([]int, 0); sum := 0
	for _, r := range strings.ToUpper(text) { v:=0; switch { case strings.ContainsRune("ABC",r): v=2; case strings.ContainsRune("DEF",r): v=3; case strings.ContainsRune("GHI",r): v=4; case strings.ContainsRune("JKL",r): v=5; case strings.ContainsRune("MNO",r): v=6; case strings.ContainsRune("PQRS",r): v=7; case strings.ContainsRune("TUV",r): v=8; case strings.ContainsRune("WXYZ",r): v=9 }; if v>0 { values=append(values,v); sum+=v } }
	return puzzleResponse{Values:values, Sum:sum}
}
