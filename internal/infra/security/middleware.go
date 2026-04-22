package security

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// RequestIDHeader is the header name for request ID.
	RequestIDHeader = "X-Request-ID"
	// AuthorizationHeader is the header name for authorization.
	AuthorizationHeader = "Authorization"
	// ContentTypeHeader is the header name for content type.
	ContentTypeHeader = "Content-Type"
	// AllowedMethods are the allowed HTTP methods.
	AllowedMethods = "GET, POST, PUT, DELETE, OPTIONS"
	// DefaultMaxAge is the default max age for CORS cache.
	DefaultMaxAge = 3600
)

// Errors specific to security middleware.
var (
	ErrUnauthorized         = errors.New("unauthorized")
	ErrInvalidToken         = errors.New("invalid token")
	ErrTokenExpired         = errors.New("token expired")
	ErrInvalidSignature     = errors.New("invalid signature")
	ErrNoAuthorization      = errors.New("no authorization header")
	ErrInvalidClaims        = errors.New("invalid claims")
	ErrMissingRequiredClaim = errors.New("missing required claim")
)

// ContextKey is a type for context keys.
type ContextKey string

// Context keys used by middleware.
const (
	ContextKeyRequestID ContextKey = "request_id"
	ContextKeyUserID    ContextKey = "user_id"
	ContextKeyClaims    ContextKey = "claims"
	ContextKeyIP        ContextKey = "client_ip"
	ContextKeyError     ContextKey = "error"
)

// Context is the middleware context that holds request-specific data.
// It's similar to standard library's context.Context but simpler for our needs.
type Context struct {
	context.Context
	values map[any]any
	mu     sync.RWMutex
}

// NewContext creates a new Context with the given parent context.
func NewContext(ctx context.Context) *Context {
	return &Context{
		Context: ctx,
		values:  make(map[any]any),
	}
}

// Value returns the value for the given key.
func (c *Context) Value(key any) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[key]
}

// Set sets the value for the given key.
func (c *Context) Set(key any, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// Handler is a function that handles a request.
// It takes a Context and returns an error.
type Handler func(ctx *Context) error

// Middleware is a function that wraps a Handler.
// It takes a Handler and returns a Handler.
type Middleware func(next Handler) Handler

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx *Context) string {
	if v := ctx.Value(ContextKeyRequestID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserID retrieves the user ID from the context.
func GetUserID(ctx *Context) string {
	if v := ctx.Value(ContextKeyUserID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetIPAddress retrieves the client IP address from the context.
func GetIPAddress(ctx *Context) string {
	if v := ctx.Value(ContextKeyIP); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetClientKey retrieves the client key from the context.
// This is used for rate limiting.
func GetClientKey(ctx *Context) string {
	if userID := GetUserID(ctx); userID != "" {
		return "user:" + userID
	}
	return GetIPAddress(ctx)
}

// JWTClaims represents the claims in a JWT token.
type JWTClaims struct {
	// Subject is the user ID.
	Subject string `json:"sub,omitempty"`
	// Issuer is the token issuer.
	Issuer string `json:"iss,omitempty"`
	// Audience is the token audience.
	Audience string `json:"aud,omitempty"`
	// ExpiresAt is the expiration time (Unix timestamp).
	ExpiresAt int64 `json:"exp,omitempty"`
	// IssuedAt is the issued at time (Unix timestamp).
	IssuedAt int64 `json:"iat,omitempty"`
	// NotBefore is the not before time (Unix timestamp).
	NotBefore int64 `json:"nbf,omitempty"`
}

// JWTAuthConfig holds the JWT authentication configuration.
type JWTAuthConfig struct {
	// SecretKey is the secret key for signing tokens.
	SecretKey []byte
	// Issuer is the token issuer.
	Issuer string
	// Audience is the expected token audience.
	Audience string
	// Subject is the expected token subject.
	Subject string
	// Expiration is the token expiration time.
	Expiration time.Duration
	// Leeway is the time leeway for token expiration validation.
	Leeway time.Duration
}

// NewJWTAuthConfig creates a new JWT auth config with defaults.
func NewJWTAuthConfig(secretKey string) *JWTAuthConfig {
	return &JWTAuthConfig{
		SecretKey:  []byte(secretKey),
		Issuer:     "devilfish",
		Expiration: 24 * time.Hour,
		Leeway:     1 * time.Minute,
	}
}

// GenerateToken generates a new JWT token for the given user ID.
func (c *JWTAuthConfig) GenerateToken(userID string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		Subject:   userID,
		Issuer:    c.Issuer,
		ExpiresAt: now.Add(c.Expiration).Unix(),
		IssuedAt:  now.Unix(),
	}

	// Encode claims as JSON
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Base64url encode claims
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signature
	h := hmac.New(sha256.New, c.SecretKey)
	h.Write([]byte(claimsEncoded))
	signature := h.Sum(nil)
	signatureEncoded := base64.RawURLEncoding.EncodeToString(signature)

	// Return token in format: header.payload.signature
	return fmt.Sprintf("%s.%s", claimsEncoded, signatureEncoded), nil
}

// ValidateToken validates a JWT token and returns the claims.
func (c *JWTAuthConfig) ValidateToken(tokenString string) (*JWTClaims, error) {
	// Split token into parts
	parts := strings.SplitN(tokenString, ".", 3)
	if len(parts) != 2 {
		return nil, ErrInvalidToken
	}

	claimsEncoded := parts[0]
	signatureEncoded := parts[1]

	// Verify signature
	h := hmac.New(sha256.New, c.SecretKey)
	h.Write([]byte(claimsEncoded))
	expectedSig := h.Sum(nil)
	expectedSigEncoded := base64.RawURLEncoding.EncodeToString(expectedSig)

	if !hmac.Equal([]byte(signatureEncoded), []byte(expectedSigEncoded)) {
		return nil, ErrInvalidSignature
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(claimsEncoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// Validate expiration
	if claims.ExpiresAt > 0 {
		now := time.Now().Unix()
		leeway := int64(c.Leeway.Seconds())
		if claims.ExpiresAt+leeway < now {
			return nil, ErrTokenExpired
		}
	}

	// Validate issuer
	if c.Issuer != "" && claims.Issuer != c.Issuer {
		return nil, ErrInvalidClaims
	}

	// Validate subject
	if c.Subject != "" && claims.Subject != c.Subject {
		return nil, ErrInvalidClaims
	}

	return &claims, nil
}

// JWTValidationMiddleware creates a JWT validation middleware.
func JWTValidationMiddleware(config *JWTAuthConfig) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			authHeader := ctx.Value("auth_header")
			if authHeader == nil {
				authHeader = ""
			}

			authString, ok := authHeader.(string)
			if !ok || authString == "" {
				return ErrNoAuthorization
			}

			// Extract token from "Bearer <token>" format
			parts := strings.SplitN(authString, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return ErrInvalidToken
			}

			claims, err := config.ValidateToken(parts[1])
			if err != nil {
				return fmt.Errorf("%w: %v", ErrUnauthorized, err)
			}

			ctx.Set(ContextKeyUserID, claims.Subject)
			ctx.Set(ContextKeyClaims, claims)

			return next(ctx)
		}
	}
}

// CORSConfig holds the CORS configuration.
type CORSConfig struct {
	// AllowedOrigins is the list of allowed origins.
	AllowedOrigins []string
	// AllowedMethods is the list of allowed HTTP methods.
	AllowedMethods []string
	// AllowedHeaders is the list of allowed headers.
	AllowedHeaders []string
	// ExposedHeaders is the list of exposed headers.
	ExposedHeaders []string
	// AllowCredentials indicates whether credentials are allowed.
	AllowCredentials bool
	// MaxAge is the max age for preflight cache.
	MaxAge int
}

// NewCORSConfig creates a new CORS config with defaults.
func NewCORSConfig(origins []string) *CORSConfig {
	return &CORSConfig{
		AllowedOrigins:   origins,
		AllowedMethods:   strings.Split(AllowedMethods, ", "),
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID"},
		ExposedHeaders:   []string{},
		AllowCredentials: true,
		MaxAge:           DefaultMaxAge,
	}
}

// CORS returns a CORS middleware handler.
func CORS(config *CORSConfig) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			origin := ctx.Value("origin")
			if origin == nil {
				origin = ""
			}

			originStr, ok := origin.(string)
			if !ok {
				originStr = ""
			}

			// Check if origin is allowed
			if !isOriginAllowed(config.AllowedOrigins, originStr) {
				// Don't expose CORS headers for disallowed origins
				return next(ctx)
			}

			// Set CORS headers
			ctx.Set("cors_origin", "*")
			if len(config.AllowedOrigins) > 0 && config.AllowedOrigins[0] != "*" {
				ctx.Set("cors_origin", originStr)
			}

			ctx.Set("cors_allow_methods", strings.Join(config.AllowedMethods, ", "))
			ctx.Set("cors_allow_headers", strings.Join(config.AllowedHeaders, ", "))

			if config.AllowCredentials {
				ctx.Set("cors_allow_credentials", "true")
			}

			if config.MaxAge > 0 {
				ctx.Set("cors_max_age", fmt.Sprintf("%d", config.MaxAge))
			}

			return next(ctx)
		}
	}
}

// isOriginAllowed checks if the given origin is allowed.
func isOriginAllowed(allowedOrigins []string, origin string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, o := range allowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}

	return false
}

// RequestIDMiddleware creates a request ID middleware.
// It generates a unique request ID for each request.
func RequestIDMiddleware() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			// Check if request ID already exists
			reqID := GetRequestID(ctx)
			if reqID == "" {
				// Generate new request ID
				reqID = generateRequestID()
				ctx.Set(ContextKeyRequestID, reqID)
			}

			ctx.Set("request_id", reqID)

			return next(ctx)
		}
	}
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to time-based ID
		return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano())
	}

	hash := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// LoggingMiddleware creates a logging middleware.
// It logs incoming requests and their results.
func LoggingMiddleware(logger Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			start := time.Now()
			reqID := GetRequestID(ctx)
			userID := GetUserID(ctx)

			logger.Info("request started",
				"request_id", reqID,
				"user_id", userID,
				"client_ip", GetIPAddress(ctx),
			)

			err := next(ctx)

			duration := time.Since(start)
			if err != nil {
				logger.Error("request failed",
					"request_id", reqID,
					"user_id", userID,
					"error", err.Error(),
					"duration", duration.String(),
				)
			} else {
				logger.Info("request completed",
					"request_id", reqID,
					"user_id", userID,
					"duration", duration.String(),
				)
			}

			return err
		}
	}
}

// RecoveryMiddleware recovers from panics and returns an error.
func RecoveryMiddleware(logger Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic recovered: %v", r)
					logger.Error("panic recovered",
						"request_id", GetRequestID(ctx),
						"panic", r,
					)
				}
			}()

			return next(ctx)
		}
	}
}

// Chain chains multiple middlewares together.
// It applies middlewares in order from left to right.
func Chain(middlewares ...Middleware) Middleware {
	return func(next Handler) Handler {
		// Apply middlewares in reverse order
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// HTTPMiddleware converts a security Middleware to an HTTP handler.
func HTTPMiddleware(m Middleware, logger Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create context
		ctx := NewContext(r.Context())

		// Set request values
		ctx.Set("method", r.Method)
		ctx.Set("path", r.URL.Path)
		ctx.Set("auth_header", r.Header.Get(AuthorizationHeader))
		ctx.Set("origin", r.Header.Get("Origin"))
		ctx.Set("content_type", r.Header.Get(ContentTypeHeader))
		ctx.Set("remote_addr", r.RemoteAddr)

		// Extract IP address
		ip := extractIP(r)
		ctx.Set(ContextKeyIP, ip)
		ctx.Set("request_id", r.Header.Get(RequestIDHeader))
		ctx.Set("origin", r.Header.Get("Origin"))

		// Wrap the HTTP response
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Apply middleware
		handler := m(func(ctx *Context) error {
			// Call the actual handler
			return serveHTTP(rw, r, ctx)
		})

		err := handler(ctx)

		// Write response
		if err != nil {
			writeError(w, err, logger)
			return
		}

		// Apply CORS headers from context
		if origin := ctx.Value("cors_origin"); origin != nil {
			w.Header().Set("Access-Control-Allow-Origin", origin.(string))
		}
		if creds := ctx.Value("cors_allow_credentials"); creds != nil {
			w.Header().Set("Access-Control-Allow-Credentials", creds.(string))
		}
		if methods := ctx.Value("cors_allow_methods"); methods != nil {
			w.Header().Set("Access-Control-Allow-Methods", methods.(string))
		}
		if headers := ctx.Value("cors_allow_headers"); headers != nil {
			w.Header().Set("Access-Control-Allow-Headers", headers.(string))
		}

		// Write request ID
		if reqID := GetRequestID(ctx); reqID != "" {
			w.Header().Set(RequestIDHeader, reqID)
		}

		// Write response body
		if body := ctx.Value("response_body"); body != nil {
			w.Header().Set(ContentTypeHeader, "application/json")
			json.NewEncoder(w).Encode(body)
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// serveHTTP is the default HTTP handler.
// It returns a 404 for the base route.
func serveHTTP(rw *responseWriter, r *http.Request, ctx *Context) error {
	if r.URL.Path == "/" {
		ctx.Set("response_body", map[string]string{
			"status": "ok",
			"name":   "DevilFish",
			"doc":    "https://github.com/devilfish",
		})
		return nil
	}

	rw.statusCode = http.StatusNotFound
	ctx.Set("response_body", map[string]string{
		"error": "not found",
	})
	return nil
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, err error, logger Logger) {
	w.Header().Set(ContentTypeHeader, "application/json")

	var statusCode int
	switch {
	case errors.Is(err, ErrUnauthorized):
		statusCode = http.StatusUnauthorized
	case errors.Is(err, ErrTokenExpired):
		statusCode = http.StatusUnauthorized
	default:
		statusCode = http.StatusInternalServerError
	}

	w.WriteHeader(statusCode)
	logger.Error("request error",
		"error", err.Error(),
	)

	json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

// extractIP extracts the client IP from the request.
// It checks X-Forwarded-For header first, then falls back to RemoteAddr.
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// Handle case where there's no port (IPv6 address)
	var ip string
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) <= 2 {
		// Most likely an IPv6 address without port
		ip = r.RemoteAddr
	} else {
		// Get the IP part (everything except last two parts :port)
		ip = strings.Join(parts[:len(parts)-1], ":")
	}

	return ip
}

// Logger is an interface for logging.
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// LoggerFromContext extracts a Logger from the context.
func LoggerFromContext(ctx *Context) Logger {
	if l := ctx.Value("logger"); l != nil {
		if logger, ok := l.(Logger); ok {
			return logger
		}
	}
	return &defaultLogger{}
}

// defaultLogger is a no-op logger.
type defaultLogger struct{}

func (d *defaultLogger) Info(msg string, args ...interface{})  {}
func (d *defaultLogger) Error(msg string, args ...interface{}) {}
func (d *defaultLogger) Debug(msg string, args ...interface{}) {}

// timeSince is a helper that returns time.Since for testing.
// It should be time.Since, but we define it as a function to avoid conflicts.
func timeSince(t time.Time) time.Duration {
	return time.Since(t)
}
