package qproxy

import (
	"context"
	"net/http"
	"net/http/pprof"

	grpcgw_runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/jacksontj/dataman/metrics/promhandler"
	"github.com/pkg/errors"

	"github.com/wish/qproxy/gateway"
	"github.com/wish/qproxy/rpc"
)

// AddRoutes matches a mux with handlers
func AddRoutes(mux *http.ServeMux, server *QProxyServer) error {
	localClient := &gateway.QProxyDirectClient{server}
	gwmux := grpcgw_runtime.NewServeMux()

	if err := rpc.RegisterQProxyHandlerClient(context.Background(), gwmux, localClient); err != nil {
		return errors.Wrap(err, "registering qproxy handler client")
	}

	mux.Handle("/v1/", CompressionHandler{gwmux.ServeHTTP})
	// Don't need to use our CompressionHandler, as prometheus has their own
	mux.Handle("/metrics", promhandler.Handler(server.metricsRegistry))

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
