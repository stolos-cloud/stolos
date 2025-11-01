package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/services/templates"
	"k8s.io/client-go/rest"
)

type TemplatesHandler struct {
	k8sClient *rest.Config
}

type DetailTemplate struct {
	templates.Template
	JsonSchema  templates.JsonSchema `json:"jsonSchema"`
	DefaultYaml string               `json:"defaultYaml"`
}

func NewTemplatesHandler(k8s *rest.Config) *TemplatesHandler {
	return &TemplatesHandler{
		k8sClient: k8s,
	}
}

// GetTemplatesList godoc
// @Summary Get templates list
// @Description returns a list of available templates on the cluster
// @Tags templates
// @Produce json
// @Success 200 {object} []templates.Template
// @Failure 500 {object} string "error"
// @Router /templates [get]
func (h *TemplatesHandler) GetTemplatesList(c *gin.Context) {

	templatesList, err := templates.ListTemplates(h.k8sClient)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, templatesList)
}

// GetTemplate godoc
// @Summary Get a detailed template
// @Description Get a template from a CRD and returns it, its json schema and a default yaml
// @Tags templates
// @Param id path string true "template CRD name"
// @Produce json
// @Success 200 {object} DetailTemplate
// @Failure 500 {object} string "error"
// @Router /templates/{id} [get]
func (h *TemplatesHandler) GetTemplate(c *gin.Context) {

	template, err := templates.GetTemplate(h.k8sClient, c.Param("name"))
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
		JsonSchema:  jsonSchema,
	}

	c.JSON(http.StatusOK, detailTemplate)
}
