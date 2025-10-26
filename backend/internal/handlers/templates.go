package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/templates"
	"k8s.io/client-go/rest"
)

type TemplatesHandler struct{}

type DetailTemplate struct {
	templates.Template
	JsonSchema  string
	DefaultYaml string
}

func (h *TemplatesHandler) GetTemplatesList(c *gin.Context) {

	// Create in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load in-cluster config: %v", err)
		return
	}

	templatesList, err := templates.ListTemplates(config)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, templatesList)
}

func (h *TemplatesHandler) GetTemplate(c *gin.Context) {
	// Create in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load in-cluster config: %v", err)
		return
	}

	template, err := templates.GetTemplate(config, c.Param("name"))
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	defaultYaml, err := template.GetDefaultYaml()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	jsonSchema, err := template.GetJsonSchema()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	detailTemplate := DetailTemplate{
		Template:    template,
		DefaultYaml: defaultYaml,
		JsonSchema:  string(jsonSchema),
	}

	c.JSON(http.StatusOK, detailTemplate)
}
