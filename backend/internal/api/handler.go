package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	Query(ctx context.Context, documentID, question string) (*query.QueryResult, error)
	QueryWithProgress(ctx context.Context, documentID, question string, progress query.ProgressFunc) (*query.QueryResult, error)
}

// ─── response types ───────────────────────────────────────────────────────────

type ingestResponse struct {
	DocumentID string     `json:"document_id"`
	ChunkCount int        `json:"chunk_count"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type citationResponse struct {
	Index        int    `json:"index"`
	DocumentName string `json:"document_name"`
	Passage      string `json:"passage"`
	PageStart    int    `json:"page_start,omitempty"`
	PageEnd      int    `json:"page_end,omitempty"`
	Section      string `json:"section,omitempty"`
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
	api.POST("/query/stream", h.handleQueryStream)
	api.GET("/health", h.handleHealth)
}

// ─── handleIngest ─────────────────────────────────────────────────────────────

// maxIngestBytes is the hard limit on uploaded PDF size (20 MB).
const maxIngestBytes = 20 * 1024 * 1024

// handleIngest accepts multipart/form-data with a "file" field.
func (h *APIHandler) handleIngest(c *gin.Context) {
	// Cap the request body before multipart parsing to enforce the 20 MB limit.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxIngestBytes)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, errorEnvelope{Error: apiError{
				Code:    "FILE_TOO_LARGE",
				Message: "file exceeds the 20 MB limit",
			}})
			return
		}
		c.JSON(http.StatusBadRequest, errorEnvelope{Error: apiError{
			Code:    "INVALID_REQUEST",
			Message: "multipart field 'file' is required",
		}})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, errorEnvelope{Error: apiError{
				Code:    "FILE_TOO_LARGE",
				Message: "file exceeds the 20 MB limit",
			}})
			return
		}
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

	resp := ingestResponse{DocumentID: result.DocumentID, ChunkCount: result.ChunkCount}
	if !result.ExpiresAt.IsZero() {
		resp.ExpiresAt = &result.ExpiresAt
	}
	c.JSON(http.StatusOK, resp)
}

func (h *APIHandler) handleIngestError(c *gin.Context, err error) {
	log.Printf("ingest failed: %v", err)
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
	DocumentID string `json:"document_id" binding:"required,uuid"`
	Question   string `json:"question" binding:"required"`
}

// handleQuery accepts a JSON body with "document_id" and "question" fields.
func (h *APIHandler) handleQuery(c *gin.Context) {
	var req queryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope{Error: apiError{
			Code:    "INVALID_REQUEST",
			Message: "request body must be JSON with a valid 'document_id' and a non-empty 'question'",
		}})
		return
	}

	result, err := h.querySvc.Query(c.Request.Context(), req.DocumentID, req.Question)
	if err != nil {
		h.handleQueryError(c, err)
		return
	}

	c.JSON(http.StatusOK, queryResponse{
		Answer:    result.Answer,
		Citations: toCitationResponses(result.Citations),
		NoResults: false,
	})
}

func toCitationResponses(cs []query.Citation) []citationResponse {
	out := make([]citationResponse, len(cs))
	for i, ci := range cs {
		out[i] = citationResponse{
			Index:        ci.Index,
			DocumentName: ci.DocumentName,
			Passage:      ci.Passage,
			PageStart:    ci.PageStart,
			PageEnd:      ci.PageEnd,
			Section:      ci.Section,
		}
	}
	return out
}

// ─── handleQueryStream ────────────────────────────────────────────────────────

type stageEvent struct {
	Stage    query.StageName    `json:"stage"`
	Passages []citationResponse `json:"passages,omitempty"`
}

// handleQueryStream runs the same pipeline as handleQuery but reports progress
// as Server-Sent Events:
//
//	event: stage   data: {"stage":"searching"}
//	event: stage   data: {"stage":"sources","passages":[...]}
//	event: stage   data: {"stage":"generating"}
//	event: answer  data: <queryResponse>     (also used for no_results)
//	event: error   data: <errorEnvelope>
//
// Only pipeline stages are streamed, never LLM tokens, so the answer is sent
// once, after citation validation (INV-02).
func (h *APIHandler) handleQueryStream(c *gin.Context) {
	var req queryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope{Error: apiError{
			Code:    "INVALID_REQUEST",
			Message: "request body must be JSON with a valid 'document_id' and a non-empty 'question'",
		}})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable proxy buffering
	c.Status(http.StatusOK)

	send := func(event string, v any) {
		data, _ := json.Marshal(v)
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		c.Writer.Flush()
	}

	result, err := h.querySvc.QueryWithProgress(c.Request.Context(), req.DocumentID, req.Question,
		func(s query.Stage) {
			send("stage", stageEvent{Stage: s.Name, Passages: toCitationResponses(s.Passages)})
		})
	if err != nil {
		_, body := queryErrorResponse(err)
		if answer, ok := body.(queryResponse); ok {
			send("answer", answer)
		} else {
			send("error", body)
		}
		return
	}

	send("answer", queryResponse{
		Answer:    result.Answer,
		Citations: toCitationResponses(result.Citations),
		NoResults: false,
	})
}

func (h *APIHandler) handleQueryError(c *gin.Context, err error) {
	c.JSON(queryErrorResponse(err))
}

// queryErrorResponse maps a query pipeline error to its HTTP status and body:
// an errorEnvelope, or a queryResponse with no_results for NoResultsError.
func queryErrorResponse(err error) (int, any) {
	var noResultsErr *query.NoResultsError
	if !errors.As(err, &noResultsErr) {
		log.Printf("query failed: %v", err)
	}
	var emptyErr *query.EmptyQueryError
	var notFoundErr *query.DocumentNotFoundError
	var llmErr *query.LLMUnavailableError
	var embedErr *ingester.EmbeddingUnavailableError
	var dbErr *store.DatabaseUnavailableError

	switch {
	case errors.As(err, &emptyErr):
		return http.StatusBadRequest, errorEnvelope{Error: apiError{
			Code:    "EMPTY_QUERY",
			Message: emptyErr.Error(),
		}}
	case errors.As(err, &notFoundErr):
		return http.StatusNotFound, errorEnvelope{Error: apiError{
			Code:    "DOCUMENT_NOT_FOUND",
			Message: "this document is no longer available — please upload it again",
		}}
	case errors.As(err, &noResultsErr):
		// INV-06 result: 200 with no_results flag, not an error status.
		return http.StatusOK, queryResponse{
			Answer:    "",
			Citations: []citationResponse{},
			NoResults: true,
		}
	case errors.As(err, &llmErr):
		return http.StatusServiceUnavailable, errorEnvelope{Error: apiError{
			Code:    "LLM_UNAVAILABLE",
			Message: "LLM service is temporarily unavailable",
		}}
	case errors.As(err, &embedErr):
		return http.StatusServiceUnavailable, errorEnvelope{Error: apiError{
			Code:    "EMBEDDING_UNAVAILABLE",
			Message: "embedding service is temporarily unavailable",
		}}
	case errors.As(err, &dbErr):
		return http.StatusServiceUnavailable, errorEnvelope{Error: apiError{
			Code:    "DATABASE_UNAVAILABLE",
			Message: "database is temporarily unavailable",
		}}
	default:
		return http.StatusInternalServerError, errorEnvelope{Error: apiError{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		}}
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
