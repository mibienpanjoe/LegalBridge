package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mibienpanjoe/legalbridge/internal/ingester"
	"github.com/mibienpanjoe/legalbridge/internal/query"
	"github.com/mibienpanjoe/legalbridge/internal/store"
)

// ─── service interfaces ───────────────────────────────────────────────────────

// ingestionService is satisfied by *ingester.Ingester.
type ingestionService interface {
	Ingest(ctx context.Context, data []byte, filename string) (*ingester.IngestResult, error)
}

// queryService is satisfied by *query.QueryProcessor.
type queryService interface {
	Query(ctx context.Context, question string) (*query.QueryResult, error)
}

// ─── response types ───────────────────────────────────────────────────────────

type ingestResponse struct {
	DocumentID string `json:"document_id"`
	ChunkCount int    `json:"chunk_count"`
}

type citationResponse struct {
	Index        int    `json:"index"`
	DocumentName string `json:"document_name"`
	Passage      string `json:"passage"`
}

type queryResponse struct {
	Answer    string             `json:"answer"`
	Citations []citationResponse `json:"citations"`
	NoResults bool               `json:"no_results"`
}

type healthResponse struct {
	Status    string `json:"status"`
	Database  string `json:"database"`
	Timestamp string `json:"timestamp"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ─── handler ─────────────────────────────────────────────────────────────────

// APIHandler holds the wired-up service dependencies.
type APIHandler struct {
	ingestion ingestionService
	querySvc  queryService
	store     store.Store
}

// NewAPIHandler constructs an APIHandler with the given dependencies.
func NewAPIHandler(ing ingestionService, qs queryService, s store.Store) *APIHandler {
	return &APIHandler{ingestion: ing, querySvc: qs, store: s}
}

// RegisterRoutes attaches all /api/* routes to r.
func (h *APIHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	api.POST("/ingest", h.handleIngest)
	api.POST("/query", h.handleQuery)
	api.GET("/health", h.handleHealth)
}

// ─── handleIngest ─────────────────────────────────────────────────────────────

// handleIngest accepts multipart/form-data with a "file" field.
func (h *APIHandler) handleIngest(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope{Error: apiError{
			Code:    "INVALID_REQUEST",
			Message: "multipart field 'file' is required",
		}})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope{Error: apiError{
			Code:    "READ_ERROR",
			Message: "failed to read uploaded file",
		}})
		return
	}

	// Detect content type from actual bytes — client-supplied Content-Type is
	// not trusted.
	mimeType := http.DetectContentType(data)
	if mimeType != "application/pdf" {
		c.JSON(http.StatusBadRequest, errorEnvelope{Error: apiError{
			Code:    "INVALID_FILE_TYPE",
			Message: "expected application/pdf, got " + mimeType,
		}})
		return
	}

	result, err := h.ingestion.Ingest(c.Request.Context(), data, header.Filename)
	if err != nil {
		h.handleIngestError(c, err)
		return
	}

	c.JSON(http.StatusOK, ingestResponse{
		DocumentID: result.DocumentID,
		ChunkCount: result.ChunkCount,
	})
}

func (h *APIHandler) handleIngestError(c *gin.Context, err error) {
	var extractErr *ingester.ExtractionFailedError
	var embedErr *ingester.EmbeddingUnavailableError
	var dbErr *store.DatabaseUnavailableError

	switch {
	case errors.As(err, &extractErr):
		c.JSON(http.StatusUnprocessableEntity, errorEnvelope{Error: apiError{
			Code:    "EXTRACTION_FAILED",
			Message: extractErr.Error(),
		}})
	case errors.As(err, &embedErr):
		c.JSON(http.StatusServiceUnavailable, errorEnvelope{Error: apiError{
			Code:    "EMBEDDING_UNAVAILABLE",
			Message: "embedding service is temporarily unavailable",
		}})
	case errors.As(err, &dbErr):
		c.JSON(http.StatusServiceUnavailable, errorEnvelope{Error: apiError{
			Code:    "DATABASE_UNAVAILABLE",
			Message: "database is temporarily unavailable",
		}})
	default:
		c.JSON(http.StatusInternalServerError, errorEnvelope{Error: apiError{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		}})
	}
}

// ─── handleQuery ──────────────────────────────────────────────────────────────

type queryRequest struct {
	Question string `json:"question" binding:"required"`
}

// handleQuery accepts a JSON body with a "question" field.
func (h *APIHandler) handleQuery(c *gin.Context) {
	var req queryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope{Error: apiError{
			Code:    "INVALID_REQUEST",
			Message: "request body must be JSON with a non-empty 'question' field",
		}})
		return
	}

	result, err := h.querySvc.Query(c.Request.Context(), req.Question)
	if err != nil {
		h.handleQueryError(c, err)
		return
	}

	citations := make([]citationResponse, len(result.Citations))
	for i, ci := range result.Citations {
		citations[i] = citationResponse{
			Index:        ci.Index,
			DocumentName: ci.DocumentName,
			Passage:      ci.Passage,
		}
	}

	c.JSON(http.StatusOK, queryResponse{
		Answer:    result.Answer,
		Citations: citations,
		NoResults: false,
	})
}

func (h *APIHandler) handleQueryError(c *gin.Context, err error) {
	var emptyErr *query.EmptyQueryError
	var noResultsErr *query.NoResultsError
	var llmErr *query.LLMUnavailableError
	var embedErr *ingester.EmbeddingUnavailableError
	var dbErr *store.DatabaseUnavailableError

	switch {
	case errors.As(err, &emptyErr):
		c.JSON(http.StatusBadRequest, errorEnvelope{Error: apiError{
			Code:    "EMPTY_QUERY",
			Message: emptyErr.Error(),
		}})
	case errors.As(err, &noResultsErr):
		// INV-06 result: 200 with no_results flag, not an error status.
		c.JSON(http.StatusOK, queryResponse{
			Answer:    "",
			Citations: []citationResponse{},
			NoResults: true,
		})
	case errors.As(err, &llmErr):
		c.JSON(http.StatusServiceUnavailable, errorEnvelope{Error: apiError{
			Code:    "LLM_UNAVAILABLE",
			Message: "LLM service is temporarily unavailable",
		}})
	case errors.As(err, &embedErr):
		c.JSON(http.StatusServiceUnavailable, errorEnvelope{Error: apiError{
			Code:    "EMBEDDING_UNAVAILABLE",
			Message: "embedding service is temporarily unavailable",
		}})
	case errors.As(err, &dbErr):
		c.JSON(http.StatusServiceUnavailable, errorEnvelope{Error: apiError{
			Code:    "DATABASE_UNAVAILABLE",
			Message: "database is temporarily unavailable",
		}})
	default:
		c.JSON(http.StatusInternalServerError, errorEnvelope{Error: apiError{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		}})
	}
}

// ─── handleHealth ─────────────────────────────────────────────────────────────

// handleHealth calls Store.Ping and returns the service health status.
func (h *APIHandler) handleHealth(c *gin.Context) {
	dbStatus := "ok"
	httpStatus := http.StatusOK

	if err := h.store.Ping(c.Request.Context()); err != nil {
		dbStatus = "error"
		httpStatus = http.StatusServiceUnavailable
	}

	overallStatus := "ok"
	if dbStatus != "ok" {
		overallStatus = "error"
	}

	c.JSON(httpStatus, healthResponse{
		Status:    overallStatus,
		Database:  dbStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
