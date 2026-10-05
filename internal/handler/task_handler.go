// Package handler mengubah HTTP request menjadi panggilan service.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"taskapi/internal/domain"
	"taskapi/internal/service"
)

type TaskHandler struct {
	svc      *service.TaskService
	notifier *service.Notifier
}

func NewTaskHandler(svc *service.TaskService, notifier *service.Notifier) *TaskHandler {
	return &TaskHandler{svc: svc, notifier: notifier}
}

type taskPayload struct {
	Title    string    `json:"title"`
	Desc     string    `json:"desc"`
	Status   string    `json:"status"`
	Priority int       `json:"priority"`
	DueDate  time.Time `json:"due_date"`
}

type listResponse struct {
	Data   []*domain.Task `json:"data"`
	Total  int            `json:"total"`
	Offset int            `json:"offset"`
	Limit  int            `json:"limit"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("handler: encode response failed: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func respondErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("handler: internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *TaskHandler) Routes(r chi.Router) {
	r.Get("/tasks", h.List)
	r.Post("/tasks", h.Create)
	r.Get("/tasks/{id}", h.Get)
	r.Put("/tasks/{id}", h.Update)
	r.Delete("/tasks/{id}", h.Delete)
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var p taskPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "body tidak boleh kosong")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	t := &domain.Task{
		Title:    p.Title,
		Desc:     p.Desc,
		Status:   p.Status,
		Priority: p.Priority,
		DueDate:  p.DueDate,
	}

	created, err := h.svc.Create(r.Context(), t)
	if err != nil {
		respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return
	}

	t, err := h.svc.Get(r.Context(), id)
	if err != nil {
		respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	status := r.URL.Query().Get("status")
	q := r.URL.Query().Get("q")

	tasks, total, err := h.svc.List(r.Context(), status, q, offset, limit)
	if err != nil {
		respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Data: tasks, Total: total, Offset: offset, Limit: limit})
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return
	}

	var p taskPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	incoming := &domain.Task{
		ID:       id,
		Title:    p.Title,
		Desc:     p.Desc,
		Status:   p.Status,
		Priority: p.Priority,
		DueDate:  p.DueDate,
	}

	oldStatus := ""
	if cur, cerr := h.svc.Get(r.Context(), id); cerr == nil {
		oldStatus = cur.Status
	}

	updated, err := h.svc.Update(r.Context(), incoming)
	if err != nil {
		respondErr(w, err)
		return
	}

	if oldStatus != "" && updated.Status != oldStatus {
		h.notifier.SendStatusChanged(r.Context(), updated)
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
