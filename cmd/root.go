package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/logging"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/metrics"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/server"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/token"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aws-outbound-jwt-proxy",
	Short: "AWS outbound JWT proxy",
	Long: `A proxy that implements AWS outbound identity federation by attaching
short-lived, signed JSON Web Tokens (JWTs) to outbound HTTP requests from
AWS workloads to external services.

Instead of having applications manage long-term API keys or passwords for
third-party services, the proxy obtains a web identity token from AWS STS
on the workload's behalf and injects it into outgoing requests. The external
service verifies the JWT against AWS's public OIDC discovery keys (signature,
expiration, audience, and subject) before granting access.

The proxy handles token acquisition, caching, and renewal transparently so
application code does not need to integrate with AWS STS directly.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cmd.Flags(), os.LookupEnv)
		if err != nil {
			return err
		}
		logging.Install(logging.New(cfg))

		provider, err := metrics.New(cfg)
		if err != nil {
			return fmt.Errorf("init metrics: %w", err)
		}
		instruments, err := provider.Instruments()
		if err != nil {
			return fmt.Errorf("build instruments: %w", err)
		}

		tokenInst, err := token.NewInstruments(provider.Meter())
		if err != nil {
			return fmt.Errorf("build token instruments: %w", err)
		}
		defer func() { _ = tokenInst.Close() }()

		tokenSource, err := token.New(cmd.Context(), cfg, tokenInst)
		if err != nil {
			return fmt.Errorf("init token source: %w", err)
		}
		var audienceResolver token.AudienceResolver
		var resolverName string
		if len(cfg.TokenAudiences) > 0 {
			audienceResolver = token.StaticAudiences(cfg.TokenAudiences)
			resolverName = "static"
		} else {
			audienceResolver = token.HostAudience{}
			resolverName = "host"
		}
		slog.Info("audience resolver selected", "resolver", resolverName)

		metricsSrv := &http.Server{
			Addr:    cfg.MetricsListenAddr,
			Handler: provider.Handler(cfg.MetricsPath),
		}
		var metricsErr chan error
		if provider.Enabled() {
			ln, err := listenTCP(cfg.MetricsListenAddr)
			if err != nil {
				return fmt.Errorf("bind --metrics-listen-addr %q: %w", cfg.MetricsListenAddr, err)
			}
			slog.Info("metrics listener starting", "addr", cfg.MetricsListenAddr, "path", cfg.MetricsPath)
			metricsErr = make(chan error, 1)
			go func() {
				if err := metricsSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					metricsErr <- err
				}
				close(metricsErr)
			}()
		}

		srv := server.New(cfg, instruments, tokenSource, audienceResolver)
		slog.Info("server starting", "addr", srv.Addr, "tls", cfg.TLSEnabled())

		proxyErr := make(chan error, 1)
		go func() {
			var err error
			if cfg.TLSEnabled() {
				err = srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
			} else {
				err = srv.ListenAndServe()
			}
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				proxyErr <- err
			}
			close(proxyErr)
		}()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-stop:
			slog.Info("shutdown requested")
		case err := <-metricsErr:
			if err != nil {
				return fmt.Errorf("metrics listener: %w", err)
			}
		case err := <-proxyErr:
			if err != nil {
				return fmt.Errorf("proxy listener: %w", err)
			}
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		_ = metricsSrv.Shutdown(shutdownCtx)
		_ = provider.Shutdown(shutdownCtx)
		return nil
	},
}

func init() {
	config.BindFlags(rootCmd.Flags())
}

func listenTCP(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
