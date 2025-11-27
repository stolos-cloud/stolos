package handlers

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stolos-cloud/stolos/backend/internal/middleware"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"github.com/stolos-cloud/stolos/backend/internal/services/gitops"
	"github.com/stolos-cloud/stolos/backend/internal/services/k8s"
	"github.com/stolos-cloud/stolos/backend/internal/services/templates"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const templateGroup = "stolos.cloud"

type TemplatesHandler struct {
	k8sClient     *k8s.K8sClient
	gitOpsService *gitops.GitOpsService
	db            *gorm.DB
}

type DetailTemplate struct {
	templates.Template
	JsonSchema  templates.JsonSchema `json:"jsonSchema"`
	DefaultYaml string               `json:"defaultYaml"`
}

func NewTemplatesHandler(k8s *k8s.K8sClient, gitOpsService *gitops.GitOpsService, db *gorm.DB) *TemplatesHandler {
	return &TemplatesHandler{
		k8sClient:     k8s,
		gitOpsService: gitOpsService,
		db:            db,
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
// @Security BearerAuth
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
// @Security BearerAuth
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
// @Param namespace query string true "deploy to which namespace"
// @Param request body string true "CRD yaml"
// @Router /templates/{id}/validate/{instance_name} [post]
// @Security BearerAuth
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
// @Param namespace query string true "deploy to which namespace"
// @Param request body string true "CRD yaml"
// @Router /templates/{id}/apply/{instance_name} [post]
// @Security BearerAuth
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

	userNamespace, err := gorm.G[models.Namespace](h.db).Where("name = ?", c.Query("namespace")).First(context.Background())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to find namespace"})
		return
	}

	claims, err := middleware.GetClaimsFromContext(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if !slices.Contains(claims.Namespaces, userNamespace.ID) && claims.Role != models.RoleAdmin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User cannot deploy to this namespace"})
	}

	if cr["metadata"] == nil {
		cr["metadata"] = make(map[string]interface{})
	}
	cr["metadata"].(map[string]interface{})["name"] = c.Param("instance_name")
	cr["metadata"].(map[string]interface{})["namespace"] = userNamespace.Name

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

	c.JSON(http.StatusOK, gin.H{"status": "ok", "cr": cr})
}

// ListDeployments lists all deployments in Kubernetes using filters.
//
// @Summary      List deployments
// @Description  Returns a list of deployments filtered by template, namespace, and API group.
// @Tags         deployments
// @Accept       json
// @Produce      json
// @Param        template    query  string  false   "Template name"
// @Param        namespace   query  string  false  "Kubernetes namespace to filter on"
// @param		 onlyMine    query  bool    false  "Only my templates"
// @Success      200         {array}  templates.Deployment{}     "List of deployments"
// @Failure      500         {object}  map[string]string "Internal server error"
// @Router       /deployments/list [get]
// @Security BearerAuth
func (h *TemplatesHandler) ListDeployments(c *gin.Context) {
	templateName := c.Query("template")
	if strings.Contains(templateName, ".") {
		templateName = strings.Split(templateName, ".")[0]
	}
	filter := k8s.K8sResourceFilter{
		Namespace: c.Query("namespace"),
		Kind:      templateName,
		Group:     templateGroup,
	}

	deployments, err := templates.ListDeploymentsForFilter(h.k8sClient, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if c.Query("onlyMine") == "true" {
		claims, err := middleware.GetClaimsFromContext(c)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		var myNamespaces []string
		if err := h.db.
			Table("user_namespaces").
			Joins("JOIN namespaces ON namespaces.id = user_namespaces.namespace_id").
			Where("user_namespaces.user_id = ?", claims.UserID).
			Pluck("CONCAT('app-', namespaces.name)", &myNamespaces).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		fmt.Printf("myNamespaces: %+v\n", myNamespaces)

		allDeployments := deployments
		deployments = []templates.Deployment{}

		for _, deployment := range allDeployments {
			if slices.Contains(myNamespaces, deployment.Namespace) {
				deployments = append(deployments, deployment)
			}
		}
	}

	c.JSON(http.StatusOK, deployments)
}

// GetDeployment retrieves a specific deployment from Kubernetes.
//
// @Summary      Get deployment
// @Description  Returns a single deployment by template, deployment name, and namespace.
// @Tags         deployments
// @Accept       json
// @Produce      json
// @Param        template     query  string  true   "Template name (CRD resource)"
// @Param        deployment   query  string  true   "Deployment name"
// @Param        namespace    query  string  true   "Kubernetes namespace"
// @Success      200          {object} interface{} "Deployment object"
// @Failure      400          {object} map[string]string "Missing parameters"
// @Failure      500          {object} map[string]string "Internal server error"
// @Router       /deployments/get [get]
// @Security BearerAuth
func (h *TemplatesHandler) GetDeployment(c *gin.Context) {
	templateName := c.Query("template")
	if strings.Contains(templateName, ".") {
		templateName = strings.Split(templateName, ".")[0]
	}
	deploymentName := c.Query("deployment")
	namespace := c.Query("namespace")

	if templateName == "" || deploymentName == "" || namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing parameters"})
		return
	}

	crdTemplate, err := templates.GetTemplate(h.k8sClient, templateName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	deployment, err := h.k8sClient.DynamicClient.Resource(schema.GroupVersionResource{
		Group:    crdTemplate.GetCRD().Spec.Group,
		Version:  crdTemplate.GetCRD().Spec.Versions[0].Name,
		Resource: crdTemplate.GetCRD().Spec.Names.Plural,
	}).Namespace(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// DeleteDeployment deletes a specific deployment from Kubernetes.
//
// @Summary      Delete deployment
// @Description  Deletes a deployment identified by template, deployment name, and namespace.
// @Tags         deployments
// @Accept       json
// @Produce      json
// @Param        template     query  string  true   "Template name (CRD resource)"
// @Param        deployment   query  string  true   "Deployment name"
// @Param        namespace    query  string  true   "Kubernetes namespace"
// @Success      200          {object} map[string]string "Deletion confirmation"
// @Failure      400          {object} map[string]string "Missing parameters"
// @Failure      500          {object} map[string]string "Internal server error"
// @Router       /deployment/delete [post]
// @Security BearerAuth
func (h *TemplatesHandler) DeleteDeployment(c *gin.Context) {
	templateName := c.Query("template")
	if strings.Contains(templateName, ".") {
		templateName = strings.Split(templateName, ".")[0]
	}
	deploymentName := c.Query("deployment")
	namespace := c.Query("namespace")

	if templateName == "" || deploymentName == "" || namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing parameters"})
		return
	}

	userNamespace, err := gorm.G[models.Namespace](h.db).Where("name = ?", namespace).First(context.Background())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to find namespace"})
		return
	}

	claims, err := middleware.GetClaimsFromContext(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if !slices.Contains(claims.Namespaces, userNamespace.ID) && claims.Role != models.RoleAdmin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User cannot deploy to this namespace"})
		return
	}

	crdTemplate, err := templates.GetTemplate(h.k8sClient, templateName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	err = h.k8sClient.DynamicClient.Resource(schema.GroupVersionResource{
		Group:    crdTemplate.GetCRD().Spec.Group,
		Version:  crdTemplate.GetCRD().Spec.Versions[0].Name,
		Resource: crdTemplate.GetCRD().Spec.Names.Plural,
	}).Namespace(namespace).Delete(context.TODO(), deploymentName, metav1.DeleteOptions{})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateTemplateFromScaffold godoc
// @Summary Create a template directory based on pre-existing scaffold directory
// @Description Creates a new template directory by copying files from the chosen scaffold directory.
// @Tags templates
// @Param scaffoldName query string true "name of the scaffold directory. see scaffolds API to list available options."
// @Param templateName query string true "name of the template directory to create."
// @Produce json
// @Success 200 {object} string "done"
// @Failure 500 {object} string "error"
// @Router /templates/create [post]
// @Security BearerAuth
func (h *TemplatesHandler) CreateTemplateFromScaffold(c *gin.Context) {

	scaffoldName := c.Query("scaffoldName")
	templateName := c.Query("templateName")

	scaffolds, err := h.gitOpsService.GetTemplateScaffolds()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list scaffolds"})
		return
	}

	if !slices.Contains(scaffolds, scaffoldName) {
		c.JSON(http.StatusNotFound, gin.H{"error": "scaffold does not exist"})
		return
	}

	err = h.gitOpsService.DuplicateDirectory(fmt.Sprintf("scaffolds/%s", scaffoldName), fmt.Sprintf("templates/%s", templateName), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, "done")
}
