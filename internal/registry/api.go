package registry

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	shutdownTimeout   = 30 * time.Second
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 1 << 20
)

type API struct {
	storage *Storage
	logger  *slog.Logger
	server  *http.Server
}

type addDomainRequest struct {
	Domain string `json:"domain"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type domainResponse struct {
	Domain    string `json:"domain"`
	CreatedAt int64  `json:"created_at"`
	Verified  bool   `json:"verified"`
}

type domainsResponse struct {
	Domains []domainResponse `json:"domains"`
}

func NewAPI(storage *Storage, logger *slog.Logger) *API {
	return &API{
		storage: storage,
		logger:  logger,
	}
}

func (a *API) Start(ctx context.Context, addr string) error {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(a.loggingMiddleware)
	r.Use(middleware.Recoverer)

	r.Post("/domains", a.handleAddDomain)
	r.Delete("/domains/{domain}", a.handleRemoveDomain)
	r.Get("/domains/{domain}/verify", a.handleVerifyDomain)
	r.Get("/domains", a.handleListDomains)

	a.server = &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	errCh := make(chan error, 1)

	go func() {
		a.logger.Info("starting registry API server", "addr", addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		a.logger.Info("context cancelled, shutting down")
		return a.shutdown()
	case sig := <-sigCh:
		a.logger.Info("received signal, shutting down", "signal", sig)
		return a.shutdown()
	case err := <-errCh:
		return err
	}
}

func (a *API) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	a.logger.Info("shutting down registry API server")
	return a.server.Shutdown(ctx)
}

func (a *API) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		a.logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

func (a *API) handleAddDomain(w http.ResponseWriter, r *http.Request) {
	var req addDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Domain == "" {
		a.respondError(w, http.StatusBadRequest, "domain is required")
		return
	}

	if err := a.storage.AddDomain(r.Context(), req.Domain); err != nil {
		if errors.Is(err, ErrDomainExists) {
			a.respondError(w, http.StatusBadRequest, "domain already exists")
			return
		}
		if errors.Is(err, ErrDomainInvalid) {
			a.respondError(w, http.StatusBadRequest, "invalid domain")
			return
		}
		a.logger.Error("failed to add domain", "error", err, "domain", req.Domain)
		a.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (a *API) handleRemoveDomain(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")

	if err := a.storage.RemoveDomain(r.Context(), domain); err != nil {
		if errors.Is(err, ErrDomainNotFound) {
			a.respondError(w, http.StatusNotFound, "domain not found")
			return
		}
		if errors.Is(err, ErrDomainInvalid) {
			a.respondError(w, http.StatusBadRequest, "invalid domain")
			return
		}
		a.logger.Error("failed to remove domain", "error", err, "domain", domain)
		a.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *API) handleVerifyDomain(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")

	if err := a.storage.VerifyDomain(r.Context(), domain); err != nil {
		if errors.Is(err, ErrDomainNotFound) {
			a.respondError(w, http.StatusNotFound, "domain not found")
			return
		}
		if errors.Is(err, ErrDomainInvalid) {
			a.respondError(w, http.StatusBadRequest, "invalid domain")
			return
		}
		a.logger.Error("failed to verify domain", "error", err, "domain", domain)
		a.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *API) handleListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := a.storage.ListDomains(r.Context())
	if err != nil {
		a.logger.Error("failed to list domains", "error", err)
		a.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if domains == nil {
		domains = []Domain{}
	}

	resp := domainsResponse{
		Domains: make([]domainResponse, len(domains)),
	}

	for i := range len(domains) {
		resp.Domains[i] = domainResponse{
			Domain:    domains[i].Domain,
			CreatedAt: domains[i].CreatedAt,
			Verified:  domains[i].Verified,
		}
	}

	a.respondJSON(w, http.StatusOK, resp)
}

func (a *API) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		a.logger.Error("failed to encode json response", "error", err)
	}
}

func (a *API) respondError(w http.ResponseWriter, status int, message string) {
	a.respondJSON(w, status, errorResponse{Error: message})
}
