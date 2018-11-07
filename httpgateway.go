package qproxy

import (
	"context"
	"net/http"
	"net/http/pprof"

	grpcgw_runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kahuang/QProxy/rpc"
)

// AddRoutes matches a mux with handlers
func AddRoutes(mux *http.ServeMux, server *server) error {
	localClient := &QProxyDirectClient{server}
	gwmux := grpcgw_runtime.NewServeMux()

	if err := rpc.RegisterQProxyHandlerClient(context.Background(), gwmux, localClient); err != nil {
		return errors.Wrap(err, "registering qproxy handler client")
	}

	mux.Handle("/v1/", CompressionHandler{server.status(server.countPlain(gwmux.ServeHTTP).ServeHTTP)})

	mux.Handle("/v2/", CompressionHandler{server.status(server.requireAuth(agw.ServeHTTP).ServeHTTP)})
	mux.Handle("/v2/mongo_proxy/healthcheck", CompressionHandler{func(w http.ResponseWriter, req *http.Request) {}})

	// Don't need to use our CompressionHandler, as prometheus has their own
	mux.Handle("/metrics", promhttp.Handler())

	// TODO: Implement
	// mux.HandleFunc("/version", server.version)

	// register all debug routes on our muxer
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))

	return nil
}

func (s *server) status(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		lw := NewStatusRecorder(w)
		h(lw, req)
		s.metrics.HTTPStatus(lw.Status())
	}
}

// StatusRecorder is a simple http status recorder
type StatusRecorder struct {
	http.ResponseWriter
	http.Flusher

	status int
}

// NewStatusRecorder returns an initialized StatusRecorder, with 200 as the
// default status.
func NewStatusRecorder(w http.ResponseWriter) *StatusRecorder {
	if flusher, ok := w.(http.Flusher); ok {
		return &StatusRecorder{ResponseWriter: w, Flusher: flusher, status: http.StatusOK}
	} else {
		return &StatusRecorder{ResponseWriter: w, status: http.StatusOK}
	}
}

// Status returns the cached http status value.
func (sr *StatusRecorder) Status() int {
	return sr.status
}

// WriteHeader caches the status, then calls the underlying ResponseWriter.
func (sr *StatusRecorder) WriteHeader(status int) {
	sr.status = status
	sr.ResponseWriter.WriteHeader(status)
}
