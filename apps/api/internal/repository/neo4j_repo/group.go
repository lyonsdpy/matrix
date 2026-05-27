package neo4j_repo

import (
	"context"
	"fmt"
	"matrix/api/domain"
	"sync"

	"github.com/google/uuid"
)

type GroupGraphRepo struct {
	mu           sync.RWMutex
	userRepo     *UserGraphRepo
	groups       map[string]*domain.Group
	userGroupIdx map[string][]*domain.Group // userID -> []*Group
	memberIdx    map[string][]*domain.User  // groupID -> []*User
	childrenIdx  map[string][]*domain.Group // parentID -> []*Group
}

func NewGroupGraphRepo(userRepo *UserGraphRepo) *GroupGraphRepo {
	r := &GroupGraphRepo{
		userRepo:     userRepo,
		groups:       make(map[string]*domain.Group),
		userGroupIdx: make(map[string][]*domain.Group),
		memberIdx:    make(map[string][]*domain.User),
		childrenIdx:  make(map[string][]*domain.Group),
	}
	return r
}

func (r *GroupGraphRepo) GetGroup(_ context.Context, id string) (*domain.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.groups[id]
	if !ok {
		return nil, fmt.Errorf("group %s not found", id)
	}
	return g, nil
}

func (r *GroupGraphRepo) ListGroups(_ context.Context, first int, _ string) ([]*domain.Group, bool, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]*domain.Group, 0, len(r.groups))
	for _, g := range r.groups {
		all = append(all, g)
	}
	if len(all) <= first {
		return all, false, "", nil
	}
	return all[:first], true, all[first-1].ID, nil
}

func (r *GroupGraphRepo) CreateGroup(_ context.Context, name string) (*domain.Group, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New().String()
	g := &domain.Group{ID: id, Name: name}
	r.groups[id] = g
	return g, nil
}

func (r *GroupGraphRepo) GetUserGroups(ctx context.Context, userID string) ([]*domain.UserGroupLink, error) {
	u, err := r.userRepo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	links := make([]*domain.UserGroupLink, 0, len(r.userGroupIdx[userID]))
	for _, g := range r.userGroupIdx[userID] {
		links = append(links, &domain.UserGroupLink{
			User:     u,
			Group:    g,
			Relation: &domain.GraphRelation{ID: userID + "-" + g.ID, FromID: userID, ToID: g.ID, Type: "MEMBER_OF"},
		})
	}
	return links, nil
}

func (r *GroupGraphRepo) GetGroupMembers(_ context.Context, groupID string) ([]*domain.UserGroupLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g := r.groups[groupID]
	links := make([]*domain.UserGroupLink, 0, len(r.memberIdx[groupID]))
	for _, u := range r.memberIdx[groupID] {
		links = append(links, &domain.UserGroupLink{
			User:     u,
			Group:    g,
			Relation: &domain.GraphRelation{ID: u.ID + "-" + groupID, FromID: u.ID, ToID: groupID, Type: "MEMBER_OF"},
		})
	}
	return links, nil
}

func (r *GroupGraphRepo) GetGroupChildren(_ context.Context, parentID string) ([]*domain.GroupGroupLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	links := make([]*domain.GroupGroupLink, 0, len(r.childrenIdx[parentID]))
	for _, g := range r.childrenIdx[parentID] {
		links = append(links, &domain.GroupGroupLink{
			Target:   g,
			Relation: &domain.GraphRelation{ID: parentID + "-" + g.ID, FromID: parentID, ToID: g.ID, Type: "CONTAINS"},
		})
	}
	return links, nil
}

func (r *GroupGraphRepo) AddUserToGroup(ctx context.Context, userID, groupID string) error {
	u, err := r.userRepo.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.groups[groupID]
	if !ok {
		return fmt.Errorf("group %s not found", groupID)
	}
	r.userGroupIdx[userID] = append(r.userGroupIdx[userID], g)
	r.memberIdx[groupID] = append(r.memberIdx[groupID], u)
	return nil
}

func (r *GroupGraphRepo) AddGroupToGroup(_ context.Context, parentID, childID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[parentID]; !ok {
		return fmt.Errorf("parent group %s not found", parentID)
	}
	child, ok := r.groups[childID]
	if !ok {
		return fmt.Errorf("child group %s not found", childID)
	}
	r.childrenIdx[parentID] = append(r.childrenIdx[parentID], child)
	return nil
}
