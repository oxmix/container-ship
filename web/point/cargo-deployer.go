package point

import (
	"ctr-ship/pool"
	"ctr-ship/web"
	"net/http"
)

func CargoDeployer(pool pool.Nodes) {
	http.HandleFunc("/deployment/cargo-deployer", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, pool) {
			return
		}

		err := pool.UpgradeCargo()
		if err != nil {
			web.Failed(w, 400, err.Error())
			return
		}
		web.Success(w, struct{}{})
	})
}
