package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type finalCoordinateRequest struct {
	Variables map[string]float64 `json:"variables"`
	Latitude  string             `json:"latitude_formula"`
	Longitude string             `json:"longitude_formula"`
}

type finalCoordinateResponse struct {
	LatitudeExpanded  string        `json:"latitude_expanded"`
	LongitudeExpanded string        `json:"longitude_expanded"`
	Point             pointResponse `json:"point"`
}

func solveFinalCoordinate(req finalCoordinateRequest) (finalCoordinateResponse, error) {
	vars, err := normalizeFormulaVariables(req.Variables)
	if err != nil {
		return finalCoordinateResponse{}, err
	}
	latText, err := expandCoordinateFormula(req.Latitude, vars)
	if err != nil {
		return finalCoordinateResponse{}, fmt.Errorf("latitude formula: %w", err)
	}
	lonText, err := expandCoordinateFormula(req.Longitude, vars)
	if err != nil {
		return finalCoordinateResponse{}, fmt.Errorf("longitude formula: %w", err)
	}
	lat, err := parseCoordinate(latText, true)
	if err != nil {
		return finalCoordinateResponse{}, fmt.Errorf("latitude result: %w", err)
	}
	lon, err := parseCoordinate(lonText, false)
	if err != nil {
		return finalCoordinateResponse{}, fmt.Errorf("longitude result: %w", err)
	}
	return finalCoordinateResponse{LatitudeExpanded: latText, LongitudeExpanded: lonText, Point: point(lat, lon)}, nil
}

func normalizeFormulaVariables(input map[string]float64) (map[string]float64, error) {
	out := make(map[string]float64, len(input))
	for key, value := range input {
		name := strings.ToUpper(strings.TrimSpace(key))
		if len(name) != 1 || name[0] < 'A' || name[0] > 'Z' {
			return nil, fmt.Errorf("variable %q must be a single letter A-Z", key)
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("variable %s must be finite", name)
		}
		out[name] = value
	}
	return out, nil
}

func expandCoordinateFormula(template string, vars map[string]float64) (string, error) {
	template = strings.TrimSpace(template)
	if template == "" {
		return "", errors.New("formula is empty")
	}
	var out strings.Builder
	for i := 0; i < len(template); {
		if template[i] != '[' {
			out.WriteByte(template[i])
			i++
			continue
		}
		end := strings.IndexByte(template[i+1:], ']')
		if end < 0 {
			return "", errors.New("missing closing ]")
		}
		end += i + 1
		expr := strings.TrimSpace(template[i+1 : end])
		if expr == "" {
			return "", errors.New("empty [] expression")
		}
		value, err := evalFormulaExpression(expr, vars)
		if err != nil {
			return "", err
		}
		if math.Abs(value-math.Round(value)) < 1e-10 {
			out.WriteString(strconv.FormatInt(int64(math.Round(value)), 10))
		} else {
			out.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
		}
		i = end + 1
	}
	return strings.TrimSpace(out.String()), nil
}

type formulaParser struct {
	s    string
	pos  int
	vars map[string]float64
}

func evalFormulaExpression(expr string, vars map[string]float64) (float64, error) {
	p := &formulaParser{s: expr, vars: vars}
	value, err := p.parseExpression()
	if err != nil {
		return 0, err
	}
	p.skipSpace()
	if p.pos != len(p.s) {
		return 0, fmt.Errorf("unexpected token near %q", p.s[p.pos:])
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("expression result is not finite")
	}
	return value, nil
}

func (p *formulaParser) parseExpression() (float64, error) {
	left, err := p.parseTerm()
	if err != nil { return 0, err }
	for {
		p.skipSpace()
		if !p.consume('+') && !p.consume('-') { return left, nil }
		op := p.s[p.pos-1]
		right, err := p.parseTerm()
		if err != nil { return 0, err }
		if op == '+' { left += right } else { left -= right }
	}
}

func (p *formulaParser) parseTerm() (float64, error) {
	left, err := p.parseFactor()
	if err != nil { return 0, err }
	for {
		p.skipSpace()
		if p.pos >= len(p.s) || !strings.ContainsRune("*/%", rune(p.s[p.pos])) { return left, nil }
		op := p.s[p.pos]
		p.pos++
		right, err := p.parseFactor()
		if err != nil { return 0, err }
		switch op {
		case '*': left *= right
		case '/':
			if right == 0 { return 0, errors.New("division by zero") }
			left /= right
		case '%':
			if right == 0 { return 0, errors.New("modulo by zero") }
			left = math.Mod(left, right)
		}
	}
}

func (p *formulaParser) parseFactor() (float64, error) {
	p.skipSpace()
	if p.consume('+') { return p.parseFactor() }
	if p.consume('-') { v, err := p.parseFactor(); return -v, err }
	if p.consume('(') {
		v, err := p.parseExpression()
		if err != nil { return 0, err }
		p.skipSpace()
		if !p.consume(')') { return 0, errors.New("missing closing parenthesis") }
		return v, nil
	}
	if p.pos >= len(p.s) { return 0, errors.New("expected number or variable") }
	if unicode.IsLetter(rune(p.s[p.pos])) {
		name := strings.ToUpper(string(p.s[p.pos]))
		p.pos++
		value, ok := p.vars[name]
		if !ok { return 0, fmt.Errorf("variable %s is not defined", name) }
		return value, nil
	}
	start := p.pos
	seenDot := false
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c >= '0' && c <= '9' { p.pos++; continue }
		if c == '.' && !seenDot { seenDot = true; p.pos++; continue }
		break
	}
	if start == p.pos { return 0, fmt.Errorf("unexpected character %q", p.s[p.pos]) }
	v, err := strconv.ParseFloat(p.s[start:p.pos], 64)
	if err != nil { return 0, errors.New("invalid number") }
	return v, nil
}

func (p *formulaParser) skipSpace() { for p.pos < len(p.s) && unicode.IsSpace(rune(p.s[p.pos])) { p.pos++ } }
func (p *formulaParser) consume(c byte) bool { if p.pos < len(p.s) && p.s[p.pos] == c { p.pos++; return true }; return false }
