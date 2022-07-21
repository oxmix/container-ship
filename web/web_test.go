package web

import (
	"ctr-ship/pool"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()

	Success(w, struct{}{})

	res := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Error(err)
		}
	}(res.Body)
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}
	o := &struct {
		Ok   bool        `json:"ok"`
		Data interface{} `json:"data,omitempty"`
	}{}
	err = json.Unmarshal(data, o)
	if err != nil {
		t.Error(err)
		return
	}
	if !o.Ok {
		t.Error("Success->Ok expected", true, "got", false)
	}
}

func TestFailed(t *testing.T) {
	w := httptest.NewRecorder()

	Failed(w, 400, "some-message")

	res := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Error(err)
		}
	}(res.Body)
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}
	o := &struct {
		Ok      bool   `json:"ok"`
		Message string `json:"message"`
	}{}
	err = json.Unmarshal(data, o)
	if err != nil {
		t.Error(err)
		return
	}
	if o.Ok {
		t.Error("Failed->Ok expected", false, "got", true)
	}
	if o.Message != "some-message" {
		t.Error("Failed->Message expected", "some-message", "got", o.Message)
	}
}

func TestCheckRequest(t *testing.T) {
	w := httptest.NewRecorder()
	pn := pool.NewPoolNodes(t.TempDir())

	if !CheckRequest(w, &http.Request{RemoteAddr: "127.0.0.1:1234"}, pn) {
		t.Error("CheckRequest expected", true, "got", false)
	}
}
