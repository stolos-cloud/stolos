package handlers

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/middleware"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"github.com/stolos-cloud/stolos/backend/internal/services/k8s"
	"github.com/stolos-cloud/stolos/backend/internal/services/templates"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const templateGroup = "stolos.cloud"

type TemplatesHandler struct {
	k8sClient *k8s.K8sClient
	db        *gorm.DB
}

type DetailTemplate struct {
	templates.Template
	JsonSchema  templates.JsonSchema `json:"jsonSchema"`
	DefaultYaml string               `json:"defaultYaml"`
}

func NewTemplatesHandler(k8s *k8s.K8sClient, db *gorm.DB) *TemplatesHandler {
	return &TemplatesHandler{
		k8sClient: k8s,
		db:        db,
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

	templatesList, err := templates.ListTemplates(h.k8sClient, templateGroup)
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

// ValidateTemplate godoc
// @Summary Validate a template deployment
// @Description Validate a template deployment
// @Tags templates
// @Accept plain
// @Produce json
// @Success 200
// @Failure 500 {object} string "error"
// @Failure 422 {object} string "validation error"
// @Param id path string true "template CRD name"
// @Param instance_name query string true "deployment name"
// @Param team query string true "deploy to which team"
// @Param request body string true "CRD yaml"
// @Router /templates/{id}/validate/{instance_name} [post]
func (h *TemplatesHandler) ValidateTemplate(c *gin.Context) {
	h.doApplyAction(c, true)
}

// ApplyTemplate godoc
// @Summary Applies a template deployment
// @Description Applies a template deployment
// @Tags templates
// @Accept plain
// @Produce json
// @Success 200
// @Failure 500 {object} string "error"
// @Failure 422 {object} string "validation error"
// @Param id path string true "template CRD name"
// @Param instance_name query string true "deployment name"
// @Param team query string true "deploy to which team"
// @Param request body string true "CRD yaml"
// @Router /templates/{id}/apply/{instance_name} [post]
func (h *TemplatesHandler) ApplyTemplate(c *gin.Context) {
	h.doApplyAction(c, false)
}

func (h *TemplatesHandler) doApplyAction(c *gin.Context, onlyDryRun bool) {
	var cr map[string]interface{}
	body, err := c.GetRawData()

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	crdTemplate, err := templates.GetTemplate(h.k8sClient, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "failed to find template"})
		return
	}

	if err := yaml.Unmarshal(body, &cr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid YAML: %v", err)})
		return
	}

	userTeam, err := gorm.G[models.Team](h.db).Where("name = ?", c.Param("team")).First(context.Background())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to find team"})
		return
	}

	claims, err := middleware.GetClaimsFromContext(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if !slices.Contains(claims.Teams, userTeam.ID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User cannot deploy to this team"})
	}

	if cr["metadata"] == nil {
		cr["metadata"] = make(map[string]interface{})
	}
	cr["metadata"].(map[string]interface{})["name"] = c.Param("instance_name")
	cr["metadata"].(map[string]interface{})["namespace"] = k8s.K8sTeamsPrefix + userTeam.Name

	apiVersion := crdTemplate.GetCRD().Spec.Group + "/" + crdTemplate.GetCRD().Spec.Versions[0].Name
	cr["kind"] = crdTemplate.GetCRD().Spec.Names.Kind
	cr["apiVersion"] = apiVersion

	gvr := schema.GroupVersionResource{
		Resource: crdTemplate.GetCRD().Spec.Names.Plural,
		Group:    crdTemplate.GetCRD().Spec.Group,
		Version:  crdTemplate.GetCRD().Spec.Versions[0].Name,
	}

	if err := h.k8sClient.ApplyCR(cr, gvr, onlyDryRun); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
