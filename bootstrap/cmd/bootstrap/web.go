package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos-bootstrap/internal/tui"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
)

// WebStep is a JSON-safe representation of tui.Step for the web API.
type WebStep struct {
	Name        string       `json:"name"`
	Title       string       `json:"title"`
	Kind        tui.StepKind `json:"kind"`
	IsDone      bool         `json:"isDone"`
	AutoAdvance bool         `json:"autoAdvance"`
	Body        string       `json:"body,omitempty"`
	Fields      []WebField   `json:"fields,omitempty"`
}

// WebField is a simplified version of tui.Field
type WebField struct {
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Optional    bool   `json:"optional"`
	Value       string `json:"value,omitempty"`
}

func makeWebSteps(model *tui.Model) []WebStep {
	var out []WebStep

	for _, s := range model.Steps {
		var wf []WebField

		// Special case for ConfigureServer_* steps
		if strings.HasPrefix(s.Name, "ConfigureServer_") {
			serverIP := strings.TrimPrefix(s.Name, "ConfigureServer_")

			// Roles dropdown
			wf = append(wf, WebField{
				Label:    "Role",
				Optional: false,
				// Use Value to reflect current choice if already set
				Value: "",
			})

			// Disk dropdown — we don’t pre-fill here (frontend fetches live)
			wf = append(wf, WebField{
				Label:    "Install Disk",
				Optional: false,
				Value:    "",
			})

			out = append(out, WebStep{
				Name:        s.Name,
				Title:       s.Title,
				Kind:        tui.StepForm,
				IsDone:      s.IsDone,
				AutoAdvance: s.AutoAdvance,
				Body:        fmt.Sprintf("Server %s configuration", serverIP),
				Fields:      wf,
			})
			continue
		}

		for _, f := range s.Fields {
			wf = append(wf, WebField{
				Label:       f.Label,
				Placeholder: f.Placeholder,
				Optional:    f.Optional,
				Value:       f.Input.Value(), // safe string
			})
		}
		out = append(out, WebStep{
			Name:        s.Name,
			Title:       s.Title,
			Kind:        s.Kind,
			IsDone:      s.IsDone,
			AutoAdvance: s.AutoAdvance,
			Body:        s.Body,
			Fields:      wf,
		})
	}
	return out
}

// web starts the HTTP API and static frontend server.
// It takes a pre-initialized model (from tui.NewWizard or similar).
func web(model *tui.Model, program *tea.Program) {
	// Optional log file for Gin and web logs
	f, _ := os.OpenFile("./stolos.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close()

	// Use existing model from caller
	webModel := model

	// Redirect Gin's access logs to the same file
	gin.DefaultWriter = f
	r := gin.Default()

	// ──────────────────────────────
	// API ROUTES (prefix /api)
	// ──────────────────────────────
	api := r.Group("/api")

	api.GET("/steps", func(c *gin.Context) {
		c.JSON(http.StatusOK, makeWebSteps(webModel))
	})

	api.POST("/steps/:name/next", func(c *gin.Context) {
		name := c.Param("name")
		_, step := tui.FindStepByName(model, name)
		if step == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "step not found"})
			return
		}

		// Parse posted form data
		var payload struct {
			Fields []WebField `json:"fields"`
		}
		if err := c.BindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}

		// If this is a form step, update field values in TUI model
		if step.Kind == tui.StepForm && len(payload.Fields) > 0 {
			for i := range step.Fields {
				if i < len(payload.Fields) {
					step.Fields[i].Input.SetValue(payload.Fields[i].Value)
				}
			}
		}

		// Simulate pressing "enter" key in TUI
		program.Send(tea.KeyMsg{Type: tea.KeyEnter})

		c.JSON(http.StatusOK, makeWebSteps(model))
	})

	api.GET("/currentstep", func(c *gin.Context) {
		if model == nil || len(model.Steps) == 0 {
			c.JSON(http.StatusOK, gin.H{"index": -1})
			return
		}

		current := model.Steps[model.CurrentStepIndex]
		c.JSON(http.StatusOK, gin.H{
			"index":  model.CurrentStepIndex,
			"body":   current.Body,
			"isDone": current.IsDone,
		})
	})

	// List of configured nodes
	api.GET("/nodes", func(c *gin.Context) {
		nodes := []map[string]string{}
		for ip, selectedDisk := range saveState.MachinesDisks {
			nodes = append(nodes, map[string]string{
				"id":   ip,
				"disk": selectedDisk,
			})
		}
		c.JSON(http.StatusOK, nodes)
	})

	// Get available disks for one node
	api.GET("/nodes/:id/disks", func(c *gin.Context) {
		ip := c.Param("id")
		disks, err := talos.GetDisks(context.Background(), ip)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		type DiskInfo struct {
			Name  string `json:"name"`
			Model string `json:"model"`
			UUID  string `json:"uuid"`
			WWID  string `json:"wwid"`
			Size  uint64 `json:"size"`
		}

		var out []DiskInfo
		for _, d := range disks {
			out = append(out, DiskInfo{
				Name:  d.DeviceName,
				Model: d.Model,
				UUID:  d.Uuid,
				WWID:  d.Wwid,
				Size:  d.Size / _gigabyte,
			})
		}

		c.JSON(http.StatusOK, out)
	})

	api.GET("/logs", func(c *gin.Context) {
		c.JSON(http.StatusOK, webModel.Logs)
	})

	// ──────────────────────────────
	// STATIC FRONTEND (Vue/Vuetify SPA)
	// ──────────────────────────────
	r.Static("/assets", "./webui-dist/assets")

	// All unmatched routes → index.html (SPA fallback)
	r.NoRoute(func(c *gin.Context) {
		c.File("./webui-dist/index.html")
	})

	// ──────────────────────────────
	// START SERVER
	// ──────────────────────────────
	if err := r.Run(":9123"); err != nil {
		panic(err)
	}
}
