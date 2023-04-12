package points

import (
	"ctr-ship/pool"
	"net/http"
)

func CargoDeployer(pool pool.Worker) {
	http.HandleFunc("/deployment/cargo-deployer", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, pool) {
			return
		}

		err := pool.UpgradeCargo()
		if err != nil {
			Failed(w, 400, err.Error())
			return
		}
		Success(w, struct{}{})
	})
}
