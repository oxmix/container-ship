package utils

import (
	"net/http"
	"testing"
)

func TestGetIP(t *testing.T) {
	if GetIP(&http.Request{RemoteAddr: "1.1.1.1:1234"}) != "1.1.1.1" {
		t.Error("GetIP.1 expected", true, "got", false)
	}

	expected := "1.1.1.1"
	got := GetIP(&http.Request{RemoteAddr: expected + ":1234"})
	if expected != got {
		t.Error("GetIP.1 expected", expected, "got", got)
	}

	expected = "2.2.2.2"
	h := http.Header{}
	h.Set("X-Real-IP", expected)
	got = GetIP(&http.Request{RemoteAddr: "127.0.0.1:1234", Header: h})
	if expected != got {
		t.Error("GetIP.3 expected", expected, "got", got)
	}

	expected = "3.3.3.3"
	h = http.Header{}
	h.Set("X-Real-IP", "1.1.1.1")
	got = GetIP(&http.Request{RemoteAddr: "3.3.3.3:1234", Header: h})
	if expected != got {
		t.Error("GetIP.3 expected", expected, "got", got)
	}
}

func TestCidrMatch(t *testing.T) {
	if !CidrMatch("127.0.0.1/8", "127.1.2.3") {
		t.Error("CidrMatch.1 expected", true, "got", false)
	}

	if !CidrMatch("0.0.0.0/0", "1.1.1.1") {
		t.Error("CidrMatch.2 expected", true, "got", false)
	}

	if !CidrMatch("fc00::/7", "fc00::ff00") {
		t.Error("CidrMatch.3 expected", true, "got", false)
	}
}
