package point

import (
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"ctr-ship/web"
	"io/ioutil"
	"log"
	"net/http"
)

func layout() []byte {
	tpl, err := ioutil.ReadFile("./web/layout.html")
	if err != nil {
		log.Println("failed loading of layout, err:", err)
		return nil
	}
	return tpl
}

func Main(pool pool.Nodes) {
	wrapper := layout()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, pool) {
			return
		}
		var content []byte
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")

			if u.Env().Environment == "ide" {
				wrapper = layout()
			}
			content = wrapper

			w.WriteHeader(200)
		} else {
			content = []byte("404\n")
			w.WriteHeader(404)
		}
		_, err := w.Write(content)
		if err != nil {
			return
		}
	})
}
