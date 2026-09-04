package service

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PublicInstanceResponse is the safe representation of a public instance,
// excluding sensitive fields like verify_hash, owner_uuid, sn.
type PublicInstanceResponse struct {
	InstanceUUID string           `json:"instance_uuid"`
	Name         string           `json:"name"`
	Type         string           `json:"type"`
	Description  string           `json:"description,omitempty"`
	Online       bool             `json:"online"`
	Properties   model.Properties `json:"properties"`
	LastSeen     int64            `json:"last_seen"`
	IsFavorited  bool             `json:"is_favorited,omitempty"`
}

type PublicInstanceService struct {
	instanceRepo repository.InstanceRepository
	publicRepo   repository.PublicInstanceRepository
}

func NewPublicInstanceService(db *gorm.DB) *PublicInstanceService {
	return &PublicInstanceService{
		instanceRepo: repository.NewInstanceRepository(db),
		publicRepo:   repository.NewPublicInstanceRepository(db),
	}
}

// GetPublicInstances returns all public instances, optionally filtered by type.
// userUUID can be empty for anonymous requests.
func (s *PublicInstanceService) GetPublicInstances(deviceType string, userUUID string) ([]PublicInstanceResponse, error) {
	var instances []model.Instance
	var err error

	if deviceType != "" {
		instances, err = s.publicRepo.FindPublicByType(deviceType)
	} else {
		instances, err = s.publicRepo.FindAllPublic()
	}
	if err != nil {
		return nil, err
	}

	// If user is logged in, fetch their favorites for is_favorited flag
	var favoriteSet map[string]bool
	if userUUID != "" {
		favorites, favErr := s.publicRepo.FindFavoritesByUser(userUUID)
		if favErr == nil {
			favoriteSet = make(map[string]bool, len(favorites))
			for _, f := range favorites {
				favoriteSet[f.InstanceUUID] = true
			}
		}
	}

	result := make([]PublicInstanceResponse, 0, len(instances))
	for _, inst := range instances {
		resp := toPublicResponse(inst)
		if favoriteSet != nil {
			resp.IsFavorited = favoriteSet[inst.InstanceUUID]
		}
		result = append(result, resp)
	}

	return result, nil
}

// GetPublicInstance returns a single public instance by UUID.
func (s *PublicInstanceService) GetPublicInstance(instanceUUID string, userUUID string) (*PublicInstanceResponse, error) {
	instance, err := s.publicRepo.FindPublicByUUID(instanceUUID)
	if err != nil {
		return nil, fmt.Errorf("public instance not found")
	}

	resp := toPublicResponse(*instance)
	if userUUID != "" {
		favorited, _ := s.publicRepo.IsFavorited(userUUID, instanceUUID)
		resp.IsFavorited = favorited
	}

	return &resp, nil
}

// TogglePublic sets the is_public flag on an instance.
func (s *PublicInstanceService) TogglePublic(instanceUUID string, isPublic bool) error {
	exists, err := s.instanceRepo.Exists(instanceUUID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("instance not found")
	}

	return s.instanceRepo.UpdateFields(instanceUUID, map[string]interface{}{
		"is_public": isPublic,
	})
}

// AddFavorite adds a public instance to the user's favorites.
func (s *PublicInstanceService) AddFavorite(userUUID, instanceUUID string) error {
	// Verify the instance is public
	_, err := s.publicRepo.FindPublicByUUID(instanceUUID)
	if err != nil {
		return fmt.Errorf("instance not found or not public")
	}

	favorite := &model.PublicInstanceFavorite{
		UserUUID:     userUUID,
		InstanceUUID: instanceUUID,
		CreatedAt:    time.Now().Unix(),
	}
	return s.publicRepo.AddFavorite(favorite)
}

// RemoveFavorite removes a public instance from the user's favorites.
func (s *PublicInstanceService) RemoveFavorite(userUUID, instanceUUID string) error {
	return s.publicRepo.RemoveFavorite(userUUID, instanceUUID)
}

// GetFavorites returns the user's favorited public instances.
func (s *PublicInstanceService) GetFavorites(userUUID string) ([]PublicInstanceResponse, error) {
	favorites, err := s.publicRepo.FindFavoritesByUser(userUUID)
	if err != nil {
		return nil, err
	}

	favUUIDs := make([]string, 0, len(favorites))
	for _, fav := range favorites {
		favUUIDs = append(favUUIDs, fav.InstanceUUID)
	}
	instancesByUUID := make(map[string]model.Instance, len(favUUIDs))
	instances, err := s.publicRepo.FindPublicByUUIDs(favUUIDs)
	if err != nil {
		return nil, err
	}
	for _, inst := range instances {
		instancesByUUID[inst.InstanceUUID] = inst
	}

	result := make([]PublicInstanceResponse, 0, len(favorites))
	for _, fav := range favorites {
		instance, ok := instancesByUUID[fav.InstanceUUID]
		if !ok {
			continue // skip deleted or unpublicized instances
		}
		resp := toPublicResponse(instance)
		resp.IsFavorited = true
		result = append(result, resp)
	}

	return result, nil
}

func toPublicResponse(inst model.Instance) PublicInstanceResponse {
	return PublicInstanceResponse{
		InstanceUUID: inst.InstanceUUID,
		Name:         inst.Name,
		Type:         inst.Type,
		Description:  inst.Description,
		Online:       inst.Online,
		Properties:   inst.Properties,
		LastSeen:     inst.LastSeen,
	}
}
