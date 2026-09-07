package main

import (
	"math"
	"strings"
	"testing"
)

func TestEvalFormulaExpression(t *testing.T) {
	v, err := evalFormulaExpression("A + B*2 - (C/2)", map[string]float64{"A": 2, "B": 3, "C": 4})
	if err != nil { t.Fatal(err) }
	if math.Abs(v-6) > 1e-12 { t.Fatalf("value=%v", v) }
}

func TestExpandCoordinateFormula(t *testing.T) {
	got, err := expandCoordinateFormula("N 59 54.[A+B][C][D]", map[string]float64{"A": 1, "B": 2, "C": 4, "D": 5})
	if err != nil { t.Fatal(err) }
	if got != "N 59 54.345" { t.Fatalf("got=%q", got) }
}

func TestSolveFinalCoordinate(t *testing.T) {
	got, err := solveFinalCoordinate(finalCoordinateRequest{
		Variables: map[string]float64{"A": 1, "B": 2, "C": 4, "D": 5, "E": 6},
		Latitude: "N 59 54.[A+B][C][D]",
		Longitude: "E 010 45.[E][B][A]",
	})
	if err != nil { t.Fatal(err) }
	if got.LatitudeExpanded != "N 59 54.345" || got.LongitudeExpanded != "E 010 45.621" { t.Fatalf("expanded=%+v", got) }
	if math.Abs(got.Point.Latitude-59.90575) > 1e-6 { t.Fatalf("lat=%v", got.Point.Latitude) }
}

func TestSolveFinalCoordinateRejectsUndefinedVariable(t *testing.T) {
	_, err := solveFinalCoordinate(finalCoordinateRequest{Latitude: "N 59 54.[A]", Longitude: "E 10 45.000"})
	if err == nil || !strings.Contains(err.Error(), "variable A is not defined") { t.Fatalf("err=%v", err) }
}

func TestFormulaRejectsDivisionByZero(t *testing.T) {
	_, err := evalFormulaExpression("1/(A-A)", map[string]float64{"A": 2})
	if err == nil || !strings.Contains(err.Error(), "division by zero") { t.Fatalf("err=%v", err) }
}
