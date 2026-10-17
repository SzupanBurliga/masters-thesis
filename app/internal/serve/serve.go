// Wspólny start serwerów HTTP (REST, GraphQL): TLS 1.3 zależnie od flagi.
package serve

import (
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"thesis/app/internal/config"
)

// HTTP uruchamia serwer. Domyślnie po TLS używa HTTP/1.1 (HTTP2_ENABLED=false), żeby konfiguracje
// C0 (HTTP/1.1 bez TLS) i C1 (TLS) różniły się wyłącznie warstwą TLS, a nie wersją HTTP.
func HTTP(cfg config.Config, h http.Handler) {
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: h, ReadHeaderTimeout: 10 * time.Second}
	if cfg.TLS {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS13}
		if !cfg.HTTP2 {
			srv.TLSConfig.NextProtos = []string{"http/1.1"}
			srv.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){}
		}
		log.Fatal(srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile))
	}
	log.Fatal(srv.ListenAndServe())
}
