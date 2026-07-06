package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/contract"
	"forwarder-factory/internal/deploy"
	"forwarder-factory/internal/monitor"
	"forwarder-factory/internal/network"
	"forwarder-factory/internal/openapi"
	"forwarder-factory/internal/wallet"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type App struct {
	Wallets   *wallet.Service
	Deploy    *deploy.Service
	Contracts *contract.Service
	Monitor   *monitor.Service
}

func NewRouter(app *App) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowedHeaders: []string{"*"}}))

	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/api/networks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, app.Wallets.ListNetworks())
	})
	r.Get("/api/networks/{name}/status", func(w http.ResponseWriter, r *http.Request) {
		status, err := app.Wallets.GetEnvStatus(chi.URLParam(r, "name"))
		writeResult(w, err, status)
	})

	r.Get("/api/wallets/generate", func(w http.ResponseWriter, r *http.Request) {
		networkName := r.URL.Query().Get("network")
		if networkName == "" {
			writeErr(w, apperror.BadRequest("Query param ?network= is required"))
			return
		}
		wallets, err := app.Wallets.GenerateForNetwork(networkName)
		if err != nil {
			writeErr(w, err)
			return
		}
		if r.URL.Query().Get("format") == "env" {
			text, err := app.Wallets.ToEnvText(wallets)
			if err != nil {
				writeErr(w, err)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(text))
			return
		}
		snippet, err := app.Wallets.ToEnvSnippet(wallets)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"wallets": wallets, "snippet": snippet})
	})
	r.Get("/api/wallets/balance", func(w http.ResponseWriter, r *http.Request) {
		networkName := r.URL.Query().Get("network")
		address := r.URL.Query().Get("address")
		if networkName == "" {
			writeErr(w, apperror.BadRequest("Query param ?network= is required"))
			return
		}
		if address == "" {
			writeErr(w, apperror.BadRequest("Query param ?address= is required"))
			return
		}
		bal, err := app.Wallets.CheckBalance(r.Context(), networkName, address)
		writeResult(w, err, bal)
	})
	r.Get("/api/wallets/status", func(w http.ResponseWriter, r *http.Request) {
		networkName := r.URL.Query().Get("network")
		if networkName == "" {
			writeErr(w, apperror.BadRequest("Query param ?network= is required"))
			return
		}
		status, err := app.Wallets.GetEnvStatus(networkName)
		writeResult(w, err, status)
	})

	r.Post("/api/deploy/compile", func(w http.ResponseWriter, _ *http.Request) {
		if err := app.Deploy.Compile(); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"compiled": true})
	})
	r.Post("/api/deploy", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Network       string `json:"network"`
			Verify        *bool  `json:"verify"`
			CompleteSetup *bool  `json:"completeSetup"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Network == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "network is required"})
			return
		}
		verify := true
		if body.Verify != nil {
			verify = *body.Verify
		}
		completeSetup := false
		if body.CompleteSetup != nil {
			completeSetup = *body.CompleteSetup
		}
		res, err := app.Deploy.Deploy(r.Context(), body.Network, verify, completeSetup)
		writeResult(w, err, res)
	})

	r.Get("/api/contracts/functions", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, app.Contracts.ListFunctions())
	})
	r.Get("/api/contracts/{network}/info", func(w http.ResponseWriter, r *http.Request) {
		networkName, err := parseNetworkParam(chi.URLParam(r, "network"))
		if err != nil {
			writeErr(w, err)
			return
		}
		info, err := app.Contracts.GetFactoryInfo(r.Context(), networkName)
		writeResult(w, err, info)
	})
	r.Post("/api/contracts/call", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Network      string            `json:"network"`
			FunctionName string            `json:"functionName"`
			Args         map[string]string `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Network == "" || body.FunctionName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "network and functionName are required"})
			return
		}
		if body.Args == nil {
			body.Args = map[string]string{}
		}
		res, err := app.Contracts.Call(r.Context(), body.Network, body.FunctionName, body.Args)
		writeResult(w, err, res)
	})

	r.Get("/api/monitor/wallets", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, app.Monitor.ListMonitoredWallets())
	})
	r.Put("/api/monitor/wallets", func(w http.ResponseWriter, r *http.Request) {
		var body monitor.WalletPushRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		res, err := app.Monitor.ReplaceWallets(r.Context(), body)
		writeResult(w, err, res)
	})
	r.Post("/api/monitor/wallets", func(w http.ResponseWriter, r *http.Request) {
		var body monitor.WalletPushRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		res, err := app.Monitor.PushWallets(r.Context(), body)
		writeResult(w, err, res)
	})
	r.Delete("/api/monitor/wallets", func(w http.ResponseWriter, r *http.Request) {
		var body monitor.WalletRemoveRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		res, err := app.Monitor.RemoveWallets(r.Context(), body)
		writeResult(w, err, res)
	})
	r.Get("/api/monitor/status", func(w http.ResponseWriter, r *http.Request) {
		networkName := r.URL.Query().Get("network")
		if networkName != "" {
			status, err := app.Monitor.GetStatus(networkName)
			writeResult(w, err, status)
			return
		}
		writeJSON(w, http.StatusOK, app.Monitor.ListRunning())
	})
	r.Get("/api/monitor/addresses", func(w http.ResponseWriter, r *http.Request) {
		networkName := r.URL.Query().Get("network")
		if networkName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "network query param is required"})
			return
		}
		addrs, err := app.Monitor.ResolveAddresses(r.Context(), networkName)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"network": networkName, "addresses": addrs})
	})
	r.Post("/api/monitor/start", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Network string `json:"network"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Network == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "network is required"})
			return
		}
		status, err := app.Monitor.Start(r.Context(), body.Network)
		writeResult(w, err, status)
	})
	r.Post("/api/monitor/stop", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Network string `json:"network"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Network == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "network is required"})
			return
		}
		status, err := app.Monitor.Stop(body.Network)
		writeResult(w, err, status)
	})

	r.Get("/api/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		spec, err := openapi.Build(requestOrigin(r))
		if err != nil {
			writeErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(spec)
	})

	docsPath := filepath.Join(mustWd(), "web", "docs")
	r.Handle("/docs-assets/*", http.StripPrefix("/docs-assets/", http.FileServer(http.Dir(docsPath))))

	r.Get("/", serveSwaggerUI)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
			return
		}
		http.NotFound(w, r)
	})

	return r
}

func serveSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerHTML))
}

const swaggerHTML = `<!DOCTYPE html>
<html>
<head>
  <title>Forwarder Factory API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <link rel="stylesheet" href="/docs-assets/sidebar.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script src="/docs-assets/sidebar.js"></script>
<script>
window.onload = function() {
  SwaggerUIBundle({
    url: '/api/openapi.json',
    dom_id: '#swagger-ui',
    deepLinking: true,
    docExpansion: 'list',
    tryItOutEnabled: true,
    tagsSorter: 'alpha',
    validatorUrl: null
  });
};
</script>
</body>
</html>`

func parseNetworkParam(value string) (string, error) {
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		names := make([]string, 0, len(network.All))
		for _, n := range network.All {
			names = append(names, n.Name)
		}
		return "", apperror.BadRequest("Invalid network \"" + value + "\". Use a real network key, e.g. bnbTestnet. Supported: " + strings.Join(names, ", "))
	}
	return value, nil
}

func requestOrigin(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	return proto + "://" + r.Host
}

func writeResult(w http.ResponseWriter, err error, payload interface{}) {
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func writeErr(w http.ResponseWriter, err error) {
	if ae, ok := apperror.AsAppError(err); ok {
		writeJSON(w, ae.StatusCode, map[string]string{"error": ae.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func mustWd() string {
	wd, _ := os.Getwd()
	if wd == "" {
		return "."
	}
	return wd
}
