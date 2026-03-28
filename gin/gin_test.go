package gin_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/bright-room/idem"
	idemgin "github.com/bright-room/idem/gin"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestWrapMiddleware_CacheMissThenHit(t *testing.T) {
	mw, err := idem.New()
	if err != nil {
		t.Fatal(err)
	}

	var callCount atomic.Int64
	r := gin.New()
	r.POST("/orders", idemgin.WrapMiddleware(mw), func(c *gin.Context) {
		n := callCount.Add(1)
		c.JSON(http.StatusCreated, gin.H{"n": n})
	})

	// First request — cache miss, handler executes.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/orders", nil)
	req1.Header.Set("Idempotency-Key", "key-1")
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("first request: want status %d, got %d", http.StatusCreated, w1.Code)
	}

	var body1 map[string]any
	if err := json.Unmarshal(w1.Body.Bytes(), &body1); err != nil {
		t.Fatalf("first request: unmarshal: %v (body=%q)", err, w1.Body.String())
	}

	if body1["n"] != float64(1) {
		t.Fatalf("first request: want n=1, got %v", body1["n"])
	}

	// Second request — cache hit, handler NOT executed.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/orders", nil)
	req2.Header.Set("Idempotency-Key", "key-1")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("second request: want status %d, got %d", http.StatusCreated, w2.Code)
	}

	var body2 map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &body2); err != nil {
		t.Fatalf("second request: unmarshal: %v (body=%q)", err, w2.Body.String())
	}

	if body2["n"] != float64(1) {
		t.Fatalf("second request: want n=1 (cached), got %v", body2["n"])
	}

	if callCount.Load() != 1 {
		t.Fatalf("handler should be called once, got %d", callCount.Load())
	}
}

func TestWrapMiddleware_WithoutKey(t *testing.T) {
	mw, err := idem.New()
	if err != nil {
		t.Fatal(err)
	}

	var callCount atomic.Int64
	r := gin.New()
	r.POST("/orders", idemgin.WrapMiddleware(mw), func(c *gin.Context) {
		callCount.Add(1)
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	// First request without key — handler executes.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/orders", nil)
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("want status %d, got %d", http.StatusCreated, w1.Code)
	}

	// Second request without key — handler executes again (no caching).
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/orders", nil)
	r.ServeHTTP(w2, req2)

	if callCount.Load() != 2 {
		t.Fatalf("handler should be called twice without idempotency key, got %d", callCount.Load())
	}
}

func TestWrapMiddleware_AbortOnCacheHit(t *testing.T) {
	mw, err := idem.New()
	if err != nil {
		t.Fatal(err)
	}

	var handlerCalls atomic.Int64
	r := gin.New()
	r.POST("/orders", idemgin.WrapMiddleware(mw), func(c *gin.Context) {
		handlerCalls.Add(1)
		c.JSON(http.StatusCreated, gin.H{"order": 1})
	})

	// First request — populates cache.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/orders", nil)
	req1.Header.Set("Idempotency-Key", "abort-key")
	r.ServeHTTP(w1, req1)

	// Second request — cache hit, handler must not run again.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/orders", nil)
	req2.Header.Set("Idempotency-Key", "abort-key")
	r.ServeHTTP(w2, req2)

	if handlerCalls.Load() != 1 {
		t.Fatalf("handler should be called exactly once, got %d", handlerCalls.Load())
	}

	// Verify cached response is returned.
	if w2.Code != http.StatusCreated {
		t.Fatalf("cache hit: want status %d, got %d", http.StatusCreated, w2.Code)
	}
}

func TestWrapMiddleware_GinResponseWriterStatus(t *testing.T) {
	mw, err := idem.New()
	if err != nil {
		t.Fatal(err)
	}

	var capturedStatus int
	var capturedWritten bool

	r := gin.New()
	r.POST("/orders", idemgin.WrapMiddleware(mw), func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
		capturedStatus = c.Writer.Status()
		capturedWritten = c.Writer.Written()
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/orders", nil)
	req.Header.Set("Idempotency-Key", "status-key")
	r.ServeHTTP(w, req)

	if capturedStatus != http.StatusCreated {
		t.Fatalf("c.Writer.Status(): want %d, got %d", http.StatusCreated, capturedStatus)
	}

	if !capturedWritten {
		t.Fatal("c.Writer.Written(): want true, got false")
	}
}

func TestWrapMiddleware_DifferentKeysAreIndependent(t *testing.T) {
	mw, err := idem.New()
	if err != nil {
		t.Fatal(err)
	}

	var callCount atomic.Int64
	r := gin.New()
	r.POST("/orders", idemgin.WrapMiddleware(mw), func(c *gin.Context) {
		n := callCount.Add(1)
		c.JSON(http.StatusCreated, gin.H{"n": n})
	})

	// Request with key-a.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/orders", nil)
	req1.Header.Set("Idempotency-Key", "key-a")
	r.ServeHTTP(w1, req1)

	// Request with key-b — different key, handler must execute again.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/orders", nil)
	req2.Header.Set("Idempotency-Key", "key-b")
	r.ServeHTTP(w2, req2)

	if callCount.Load() != 2 {
		t.Fatalf("handler should be called twice for different keys, got %d", callCount.Load())
	}

	var body1, body2 map[string]any
	if err := json.Unmarshal(w1.Body.Bytes(), &body1); err != nil {
		t.Fatalf("key-a: unmarshal: %v", err)
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &body2); err != nil {
		t.Fatalf("key-b: unmarshal: %v", err)
	}

	if body1["n"] == body2["n"] {
		t.Fatal("different keys should produce different responses")
	}
}

func TestWrapMiddleware_RouteGroup(t *testing.T) {
	mw, err := idem.New()
	if err != nil {
		t.Fatal(err)
	}

	var callCount atomic.Int64
	r := gin.New()
	api := r.Group("/api", idemgin.WrapMiddleware(mw))
	api.POST("/payments", func(c *gin.Context) {
		n := callCount.Add(1)
		c.JSON(http.StatusCreated, gin.H{"n": n})
	})

	// First request.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/payments", nil)
	req1.Header.Set("Idempotency-Key", "group-key")
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("want status %d, got %d", http.StatusCreated, w1.Code)
	}

	// Second request — cache hit.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/payments", nil)
	req2.Header.Set("Idempotency-Key", "group-key")
	r.ServeHTTP(w2, req2)

	if callCount.Load() != 1 {
		t.Fatalf("handler should be called once, got %d", callCount.Load())
	}
}
