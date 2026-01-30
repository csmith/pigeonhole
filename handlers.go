package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"
)

func handlePost(store *store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cubby := r.Context().Value(contextKeyCubby).(string)
		content, err := readBodyContent(r)
		if err != nil {
			slog.Error("Failed to read request body", "error", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		slog.Info("Adding message", "cubby", cubby, "client", r.RemoteAddr)
		_, err = store.insert(cubby, content)
		if err != nil {
			slog.Error("Failed to add message", "cubby", cubby, "error", err)
			http.Error(w, "Failed to store message", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
}

func handleGet(store *store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cubby := r.Context().Value(contextKeyCubby).(string)
		messages, err := store.retrieve(cubby)
		if err != nil {
			slog.Error("Failed to get messages", "cubby", cubby, "error", err)
			http.Error(w, "Failed to retrieve messages", http.StatusInternalServerError)
			return
		}

		slog.Debug("Retrieving messages", "cubby", cubby, "client", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		if err := enc.Encode(messages); err != nil {
			slog.Error("Failed to encode messages", "error", err)
		}
	})
}

func handleDelete(store *store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cubby := r.Context().Value(contextKeyCubby).(string)
		if notafterStr := r.URL.Query().Get("notafter"); notafterStr != "" {
			notafter, err := time.Parse(time.RFC3339, notafterStr)
			if err != nil {
				slog.Error("Failed to parse notafter timestamp", "value", notafterStr, "error", err)
				http.Error(w, "Invalid timestamp format", http.StatusBadRequest)
				return
			}

			if err := store.removeOldest(cubby, notafter); err != nil {
				slog.Error("Failed to delete messages before timestamp", "cubby", cubby, "notafter", notafter, "error", err)
				http.Error(w, "Failed to delete messages", http.StatusInternalServerError)
				return
			}

			slog.Info("Deleted old messages", "cubby", cubby, "notafter", notafter)
		} else {
			if err := store.clear(cubby); err != nil {
				slog.Error("Failed to delete messages", "cubby", cubby, "error", err)
				http.Error(w, "Failed to delete messages", http.StatusInternalServerError)
				return
			}

			slog.Info("Deleted all messages", "cubby", cubby)
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

var fieldNames = []string{
	"message",
	"text",
	"content",
}

func readBodyContent(r *http.Request) (string, error) {
	// Try to parse the media type, but if it fails just carry on anyway;
	// in that case we'll just read the entire body.
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))

	switch mediaType {
	case "application/x-www-form-urlencoded":
		if err := r.ParseForm(); err != nil {
			return "", err
		}
		for _, n := range fieldNames {
			if v := r.FormValue(n); v != "" {
				return v, nil
			}
		}
		return r.Form.Encode(), nil

	case "multipart/form-data":
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			return "", err
		}
		for _, n := range fieldNames {
			if v := r.FormValue(n); v != "" {
				return v, nil
			}
		}
		return r.Form.Encode(), nil

	case "application/json":
		var data map[string]any
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return "", err
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return "", err
		}
		for _, n := range fieldNames {
			if v, ok := data[n].(string); ok {
				return v, nil
			}
		}
		return string(body), nil

	default:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
}
