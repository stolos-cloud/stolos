package api

import (
	"github.com/google/uuid"
	"github.com/stolos-cloud/stolos/backend/internal/models"
)

type UserResponse struct {
	ID         uuid.UUID       `json:"id"`
	Email      string          `json:"email"`
	Role       models.Role     `json:"role"`
	Namespaces []NamespaceInfo `json:"namespaces"`
}

type NamespaceInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type NamespaceResponse struct {
	ID    uuid.UUID      `json:"id"`
	Name  string         `json:"name"`
	Users []UserResponse `json:"users,omitempty"`
}

func ToUserResponse(user *models.User) UserResponse {
	namespaces := make([]NamespaceInfo, len(user.Namespaces))
	for i, ns := range user.Namespaces {
		namespaces[i] = NamespaceInfo{
			ID:   ns.ID,
			Name: ns.Name,
		}
	}

	return UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		Role:       user.Role,
		Namespaces: namespaces,
	}
}

func ToNamespaceResponse(namespace *models.Namespace, includeUsers bool) NamespaceResponse {
	response := NamespaceResponse{
		ID:   namespace.ID,
		Name: namespace.Name,
	}

	if includeUsers {
		users := make([]UserResponse, len(namespace.Users))
		for i, user := range namespace.Users {
			users[i] = ToUserResponse(&user)
		}
		response.Users = users
	}

	return response
}
