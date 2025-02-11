package points

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"ship/deployment"
	"ship/pool"
	u "ship/utils"
	"time"
)

func Connection(worker pool.Worker) {
	http.HandleFunc("GET /connection/{key}", func(w http.ResponseWriter, r *http.Request) {
		deps := worker.GetDeployment()
		re := regexp.MustCompile(`^[a-f0-9]*$`)
		if !re.MatchString(r.PathValue("key")) {
			http.Error(w, "echo 'bad key'", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		_, err := w.Write(deps.CargoShell(r.PathValue("key"), r.URL.Query().Get("pprof")))
		if err != nil {
			log.Printf("/connection -> failed response write, err: %q", err)
		}
	})

	http.HandleFunc("POST /connection/done", func(w http.ResponseWriter, r *http.Request) {
		dt := new(struct {
			Host string `json:"host"`
			Key  string `json:"key"`
		})
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(r.Body)
		err := json.NewDecoder(r.Body).Decode(dt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !pool.CheckConnectKey(dt.Key) {
			http.Error(w, "no access by connect key", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		randomBytes := make([]byte, 32)
		_, err = rand.Read(randomBytes)
		if err != nil {
			http.Error(w, "rand err: "+err.Error(), http.StatusInternalServerError)
			return
		}
		hash := sha256.Sum256(randomBytes)

		var node *pool.NodeConf
		if node = worker.GetNode(dt.Host); node != nil {
			node.Key = hex.EncodeToString(hash[:])
			node.IP = u.GetIP(r)
		} else {
			node = &pool.NodeConf{
				Key:  hex.EncodeToString(hash[:]),
				IP:   u.GetIP(r),
				Name: dt.Host,
			}
		}

		if node.Name == "" {
			http.Error(w, "node apply failed: name is empty", http.StatusBadRequest)
			return
		}

		err = node.Save(worker)
		if err != nil {
			http.Error(w, "failed save conf: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println("added node:", node.Name)

		w.WriteHeader(200)
		_, _ = w.Write([]byte(node.Key))
	})

	http.HandleFunc("POST /stream", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Key")
		ip := u.GetIP(r)
		nodeConf := worker.GetNodeByKey(key)
		if nodeConf == nil {
			message := "point /stream - no access, ip: " + u.GetIP(r)
			log.Println(message)
			Failed(w, http.StatusForbidden, message)
			return
		}

		execs := func() []deployment.Request {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			ticker := time.NewTicker(time.Second / 2)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					execs := worker.GetQueue(ip)
					if len(execs) > 0 {
						return execs
					}
				}
			}
		}()

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println(err)
			}
		}(r.Body)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("point /stream - read error:", err)
			Failed(w, http.StatusBadRequest, err.Error())
			return
		}

		err = worker.Parsing(nodeConf, ip, bodyBytes)
		if err != nil {
			log.Println("point /stream - parsing err:", err)
			Failed(w, http.StatusBadRequest, err.Error())
			return
		}

		Success(w, execs)
	})
}
