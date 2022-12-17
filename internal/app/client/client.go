package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/dewadg/freedom/internal/certresolver"
	"github.com/dewadg/freedom/internal/config"
	"github.com/sirupsen/logrus"
)

func Run(cfg *config.Config) {
	var httpServer, httpsServer *http.Server
	var certResolver certresolver.Resolver

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

		certResolver = certresolver.NewFileResolver(cfg)
		if err := certResolver.Init(); err != nil {
			logrus.WithError(err).Fatal("failed to init certificate resolver")
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
		if err := httpsServer.ListenAndServeTLS(certResolver.Cert(), certResolver.PrivateKey()); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	targetURL, _ := url.Parse(cfg.ProxyPass.Target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logrus.WithFields(logrus.Fields{
			"agent":          r.UserAgent(),
			"remote_address": r.RemoteAddr,
			"timestamp":      time.Now().Format(time.RFC3339),
			"target":         cfg.ProxyPass.Target + r.URL.Path,
		})

		r.Host = targetURL.Host
		r.URL.Host = targetURL.Host
		r.URL.Scheme = targetURL.Scheme

		proxy.ServeHTTP(w, r)

		log.Info("incoming request")
	})
}
