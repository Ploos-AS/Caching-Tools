package main

import "testing"

func TestA1Z26(t *testing.T) { got := a1z26("Cache!"); want := []int{3,1,3,8,5}; if len(got.Values)!=len(want) { t.Fatalf("values=%v",got.Values) }; for i:=range want { if got.Values[i]!=want[i] { t.Fatalf("values=%v",got.Values) } }; if got.Sum!=20 { t.Fatalf("sum=%d",got.Sum) } }
func TestCaesar(t *testing.T) { if got:=caesar("Abc Xyz!",13); got!="Nop Klm!" { t.Fatalf("got=%q",got) }; if got:=caesar("ABC",-1); got!="ZAB" { t.Fatalf("got=%q",got) } }
func TestROT47(t *testing.T) { if got:=rot47("Hello!"); got!="w6==@P" { t.Fatalf("got=%q",got) }; if got:=rot47(rot47("Geocache 123")); got!="Geocache 123" { t.Fatalf("roundtrip=%q",got) } }
func TestDigitChecksum(t *testing.T) { sum,root:=digitChecksum("GC 12-345"); if sum!=15||root!=6 { t.Fatalf("sum=%d root=%d",sum,root) } }
func TestSubstitute(t *testing.T) { got,err:=substitute("Abba!","ABC","XYZ"); if err!=nil { t.Fatal(err) }; if got!="Xyyx!" { t.Fatalf("got=%q",got) }; if _,err:=substitute("A","ABC","XY"); err==nil { t.Fatal("expected length error") } }
func TestMorseRoundTrip(t *testing.T) { encoded,err:=morse("CACHE 42",false); if err!=nil { t.Fatal(err) }; decoded,err:=morse(encoded,true); if err!=nil { t.Fatal(err) }; if decoded!="CACHE 42" { t.Fatalf("decoded=%q encoded=%q",decoded,encoded) }; if _,err:=morse("... --- ... --..--",true); err==nil { t.Fatal("expected unsupported code error") } }
func TestBaconRoundTrip(t *testing.T) { encoded,err:=bacon("CACHE",false); if err!=nil { t.Fatal(err) }; decoded,err:=bacon(encoded,true); if err!=nil { t.Fatal(err) }; if decoded!="CACHE" { t.Fatalf("decoded=%q encoded=%q",decoded,encoded) }; if _,err:=bacon("AAAA",true); err==nil { t.Fatal("expected length error") } }
func TestConvertBase(t *testing.T) { got,err:=convertBase("FF 10",16,10); if err!=nil { t.Fatal(err) }; if got!="255 16" { t.Fatalf("got=%q",got) }; got,err=convertBase("255",10,2); if err!=nil||got!="11111111" { t.Fatalf("got=%q err=%v",got,err) }; if _,err:=convertBase("2",2,10); err==nil { t.Fatal("expected invalid digit") } }
func TestPhoneKeypad(t *testing.T) { got:=phoneKeypad("CACHE"); want:=[]int{2,2,2,4,3}; for i:=range want { if got.Values[i]!=want[i] { t.Fatalf("values=%v",got.Values) } }; if got.Sum!=13 { t.Fatalf("sum=%d",got.Sum) } }
