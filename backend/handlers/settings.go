package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/imadMansour85/Jetyard/backend/config"
	natsclient "github.com/imadMansour85/Jetyard/backend/nats"
)

type SettingsHandler struct {
	currentSettings *config.NATSSettings
}

func NewSettingsHandler(s *config.NATSSettings) *SettingsHandler {
	return &SettingsHandler{currentSettings: s}
}

// GET /api/settings/nats — returns current settings with sensitive fields redacted.
func (h *SettingsHandler) GetNATSSettings(w http.ResponseWriter, r *http.Request) {
	safe := h.currentSettings.SafeCopy()
	writeJSON(w, http.StatusOK, safe)
}

// POST /api/settings/nats/test — tries connecting with the provided settings.
func (h *SettingsHandler) TestNATSSettings(w http.ResponseWriter, r *http.Request) {
	var s config.NATSSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if s.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	c, err := natsclient.NewClientWithSettings(&s)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.Close()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection successful",
	})
}

// POST /api/settings/nats/save — saves settings to disk and restarts the process
// in place (via exec, replacing the current process image) so the new config
// takes effect on the next connection attempt. This does not depend on an
// external supervisor (e.g. Docker's restart policy) to bring the process
// back — it restarts itself directly.
func (h *SettingsHandler) SaveNATSSettings(w http.ResponseWriter, r *http.Request) {
	var s config.NATSSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if s.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	// Validate the new settings before saving
	c, err := natsclient.NewClientWithSettings(&s)
	if err != nil {
		writeError(w, http.StatusBadRequest, "connection test failed: "+err.Error())
		return
	}
	c.Close()

	if err := s.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save settings: "+err.Error())
		return
	}

	log.Printf("NATS settings saved to disk — restarting process to apply")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Settings saved. Reconnecting…",
	})

	// Flush the response before restarting so the client receives it.
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	go restartProcess()
}

// restartProcess replaces the current process image with a fresh copy of
// itself, so the new settings take effect without relying on an external
// supervisor to notice the process exited and bring it back.
func restartProcess() {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("restart failed: could not resolve executable path: %v — exiting instead", err)
		os.Exit(0)
		return
	}

	if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
		log.Printf("restart failed: exec failed: %v — exiting instead", err)
		os.Exit(0)
	}
}
