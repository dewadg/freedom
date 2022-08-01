package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dewadg/freedom/internal/config"
	"github.com/sirupsen/logrus"
)

func Run(cfg *config.Config) {
	var httpServer, httpsServer *http.Server

	if cfg.SSL.Enabled {
		httpAddr := "127.0.0.1:8000"
		if os.Getenv("APP_ENV") == "production" {
			httpAddr = "0.0.0.0:80"
		}
		httpServer = &http.Server{
			Addr:    httpAddr,
			Handler: handleRedirect(cfg),
		}

		httpsAddr := "127.0.0.1:44300"
		if os.Getenv("APP_ENV") == "production" {
			httpsAddr = "0.0.0.0:443"
		}
		httpsServer = &http.Server{
			Addr:    httpsAddr,
			Handler: handleProxyPass(cfg),
		}
	} else {
		httpAddr := "127.0.0.1:8000"
		if os.Getenv("APP_ENV") == "production" {
			httpAddr = "0.0.0.0:80"
		}
		httpServer = &http.Server{
			Addr:    httpAddr,
			Handler: handleProxyPass(cfg),
		}
	}

	doneChan := make(chan os.Signal, 1)
	signal.Notify(doneChan, os.Kill, os.Interrupt)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Error("failed to start http server")
			doneChan <- nil
		}
	}()

	go func() {
		if httpsServer == nil {
			return
		}
		if err := httpsServer.ListenAndServeTLS(cfg.SSL.Cert, cfg.SSL.PrivateKey); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Error("failed to start https server")
			doneChan <- nil
		}
	}()

	logrus.WithFields(logrus.Fields{
		"target":              cfg.ProxyPass.Target,
		"exposed_address":     cfg.ProxyPass.ExposedAddress,
		"exposed_address_ssl": cfg.ProxyPass.ExposedAddressSSL,
		"ssl_enabled":         cfg.SSL.Enabled,
	}).Info("freedom started")
	<-doneChan

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logrus.Info("freedom shutting down")
	_ = httpServer.Shutdown(ctx)
	if httpsServer != nil {
		_ = httpsServer.Shutdown(ctx)
	}
	logrus.Info("freedom shut down successfully")
}

func handleRedirect(cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+cfg.ProxyPass.ExposedAddressSSL+r.URL.Path, http.StatusMovedPermanently)
	})
}

func handleProxyPass(cfg *config.Config) http.Handler {
	proxy := newProxyPasser(cfg)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logrus.WithFields(logrus.Fields{
			"agent":          r.UserAgent(),
			"remote_address": r.RemoteAddr,
			"timestamp":      time.Now().Format(time.RFC3339),
			"target":         cfg.ProxyPass.Target + r.URL.Path,
		})

		start := time.Now()
		resp, err := proxy.Call(r.Context(), r)
		if err != nil {
			log.WithError(err).Error("request to target error")
			_, _ = fmt.Fprintf(w, err.Error())
			return
		}
		log = log.WithField("latency", time.Now().Sub(start).Milliseconds())

		for key, values := range resp.headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.statusCode)
		_, _ = w.Write(resp.body)

		log.Info("incoming request")
	})
}
