package scim

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/slashdevops/idp-scim-sync/internal/model"
	"github.com/slashdevops/idp-scim-sync/pkg/aws"
)

// This implement core.SCIMService interface

//go:generate go run go.uber.org/mock/mockgen@v0.5.0 -package=mocks -destination=../../mocks/scim/scim_mocks.go -source=scim.go AWSSCIMProvider

// AWSSCIMProvider interface to consume aws package methods
type AWSSCIMProvider interface {
	// ListUsers lists users in SCIM Provider
	ListUsers(ctx context.Context, filter string) (*aws.ListUsersResponse, error)

	// CreateOrGetUser creates a user in SCIM Provider
	CreateOrGetUser(ctx context.Context, u *aws.CreateUserRequest) (*aws.CreateUserResponse, error)

	// PutUser updates a user in SCIM Provider
	PutUser(ctx context.Context, usr *aws.PutUserRequest) (*aws.PutUserResponse, error)

	// DeleteUser deletes a user in SCIM Provider
	DeleteUser(ctx context.Context, id string) error

	// GetUser gets a user in SCIM Provider
	GetUser(ctx context.Context, userID string) (*aws.GetUserResponse, error)

	// GetUserByUserName gets a user in SCIM Provider
	GetUserByUserName(ctx context.Context, userName string) (*aws.GetUserResponse, error)

	// ListGroups lists groups in SCIM Provider
	ListGroups(ctx context.Context, filter string) (*aws.ListGroupsResponse, error)

	// CreateOrGetGroup creates a group in SCIM Provider
	CreateOrGetGroup(ctx context.Context, g *aws.CreateGroupRequest) (*aws.CreateGroupResponse, error)

	// DeleteGroup deletes a group in SCIM Provider
	DeleteGroup(ctx context.Context, id string) error

	// PatchGroup patches a group in SCIM Provider
	PatchGroup(ctx context.Context, pgr *aws.PatchGroupRequest) error
}

// MaxPatchGroupMembersPerRequest is the Maximum members in group members in a single request.
const MaxPatchGroupMembersPerRequest = 100

// ErrSCIMProviderNil is returned when the SCIMProvider is nil
var ErrSCIMProviderNil = fmt.Errorf("scim: Provider is nil")

// Provider represents a SCIM provider
type Provider struct {
	scim AWSSCIMProvider
}

// NewProvider creates a new SCIM provider
func NewProvider(scim AWSSCIMProvider) (*Provider, error) {
	if scim == nil {
		return nil, ErrSCIMProviderNil
	}

	return &Provider{scim: scim}, nil
}

// GetGroups returns groups from SCIM Provider
func (s *Provider) GetGroups(ctx context.Context) (*model.GroupsResult, error) {
	groupsResponse, err := s.scim.ListGroups(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("scim: error listing groups: %w", err)
	}

	groups := make([]*model.Group, len(groupsResponse.Resources))
	for i, group := range groupsResponse.Resources {
		g := model.GroupBuilder().
			WithSCIMID(group.ID).
			WithName(group.DisplayName).
			WithIPID(group.ExternalID).
			Build()

		groups[i] = g

	}

	groupsResult := model.GroupsResultBuilder().WithResources(groups).Build()
	slog.Debug("scim: GetGroups()", "groups", len(groups))

	return groupsResult, nil
}

// CreateGroups creates groups in SCIM Provider
func (s *Provider) CreateGroups(ctx context.Context, gr *model.GroupsResult) (*model.GroupsResult, error) {
	if gr == nil {
		return nil, fmt.Errorf("scim: error creating groups, groups result is nil")
	}

	groups := make([]*model.Group, len(gr.Resources))

	for i, group := range gr.Resources {
		groupRequest := &aws.CreateGroupRequest{
			DisplayName: group.Name,
			ExternalID:  group.IPID,
		}

		slog.Warn("creating group", "group", group.Name)

		r, err := s.scim.CreateOrGetGroup(ctx, groupRequest)
		if err != nil {
			return nil, fmt.Errorf("scim: error creating group: %w", err)
		}

		g := model.GroupBuilder().
			WithSCIMID(r.ID).
			WithName(group.Name).
			WithIPID(group.IPID).
			WithEmail(group.Email).
			Build()

		groups[i] = g
	}

	groupsResult := model.GroupsResultBuilder().WithResources(groups).Build()
	slog.Debug("scim: CreateGroups()", "groups", len(groups))

	return groupsResult, nil
}

// UpdateGroups updates groups in SCIM Provider
func (s *Provider) UpdateGroups(ctx context.Context, gr *model.GroupsResult) (*model.GroupsResult, error) {
	groups := make([]*model.Group, len(gr.Resources))

	for i, group := range gr.Resources {
		groupRequest := &aws.PatchGroupRequest{
			Group: aws.Group{
				ID:          group.SCIMID,
				DisplayName: group.Name,
			},
			Patch: aws.Patch{
				Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
				Operations: []*aws.Operation{
					{
						OP: "replace",
						Value: map[string]string{
							"id":         group.SCIMID,
							"externalId": group.IPID,
						},
					},
				},
			},
		}

		slog.Warn("updating group", "group", group.Name, "email", group.Email)

		if err := s.scim.PatchGroup(ctx, groupRequest); err != nil {
			return nil, fmt.Errorf("scim: error updating groups: %w", err)
		}

		// return the same group
		g := model.GroupBuilder().
			WithSCIMID(group.SCIMID).
			WithName(group.Name).
			WithIPID(group.IPID).
			WithEmail(group.Email).
			Build()

		groups[i] = g
	}

	groupsResult := model.GroupsResultBuilder().WithResources(groups).Build()

	slog.Debug("scim: UpdateGroups()", "groups", len(groups))

	return groupsResult, nil
}

// DeleteGroups deletes groups in SCIM Provider
func (s *Provider) DeleteGroups(ctx context.Context, gr *model.GroupsResult) error {
	for _, group := range gr.Resources {
		slog.Warn("deleting group", "group", group.Name, "email", group.Email)

		if err := s.scim.DeleteGroup(ctx, group.SCIMID); err != nil {
			return fmt.Errorf("scim: error deleting group: %s, %w", group.SCIMID, err)
		}
	}
	return nil
}

// GetUsers returns users from SCIM Provider
func (s *Provider) GetUsers(ctx context.Context) (*model.UsersResult, error) {
	usersResponse, err := s.scim.ListUsers(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("scim: error listing users: %w", err)
	}

	users := make([]*model.User, len(usersResponse.Resources))
	for i, user := range usersResponse.Resources {
		u := buildUser(user)
		users[i] = u
	}

	usersResult := model.UsersResultBuilder().WithResources(users).Build()
	slog.Debug("scim: GetUsers()", "users", len(users))

	return usersResult, nil
}

// CreateUsers creates users in SCIM Provider
func (s *Provider) CreateUsers(ctx context.Context, ur *model.UsersResult) (*model.UsersResult, error) {
	users := make([]*model.User, len(ur.Resources))

	for i, user := range ur.Resources {
		userRequest := buildCreateUserRequest(user)

		slog.Warn("creating user", "user", user.DisplayName, "email", user.GetPrimaryEmailAddress())

		cogu, err := s.scim.CreateOrGetUser(ctx, userRequest)
		if err != nil {
			return nil, fmt.Errorf("scim: error creating user: %w", err)
		}

		user.SCIMID = cogu.ID
		user.SetHashCode()

		users[i] = user
	}

	usersResult := model.UsersResultBuilder().WithResources(users).Build()
	slog.Debug("scim: CreateUsers()", "users", len(users))

	return usersResult, nil
}

// UpdateUsers updates users in SCIM Provider given a list of users
func (s *Provider) UpdateUsers(ctx context.Context, ur *model.UsersResult) (*model.UsersResult, error) {
	users := make([]*model.User, len(ur.Resources))

	for i, user := range ur.Resources {
		if user.SCIMID == "" {
			return nil, fmt.Errorf("scim: error updating user, user ID is empty: %s", user.SCIMID)
		}

		userRequest := buildPutUserRequest(user)

		slog.Warn("updating user", "user", user.DisplayName, "email", user.GetPrimaryEmailAddress())

		pur, err := s.scim.PutUser(ctx, userRequest)
		if err != nil {
			return nil, fmt.Errorf("scim: error updating user: %w", err)
		}

		// update the user SCIM ID from the put user response
		user.SCIMID = pur.ID
		user.SetHashCode()

		users[i] = user
	}

	usersResult := model.UsersResultBuilder().WithResources(users).Build()
	slog.Debug("scim: UpdateUsers()", "users", len(users))

	return usersResult, nil
}

// DeleteUsers deletes users in SCIM Provider given a list of users
func (s *Provider) DeleteUsers(ctx context.Context, ur *model.UsersResult) error {
	for _, user := range ur.Resources {
		slog.Warn("deleting user", "user", user.DisplayName, "email", user.GetPrimaryEmailAddress())

		if err := s.scim.DeleteUser(ctx, user.SCIMID); err != nil {
			return fmt.Errorf("scim: error deleting user: %s, %w", user.SCIMID, err)
		}
	}

	return nil
}

type patchValue struct {
	Value string `json:"value"`
}

// CreateGroupsMembers creates groups members in SCIM Provider given a list of groups members
func (s *Provider) CreateGroupsMembers(ctx context.Context, gmr *model.GroupsMembersResult) (*model.GroupsMembersResult, error) {
	groupsMembers := make([]*model.GroupMembers, len(gmr.Resources))

	for i, groupMembers := range gmr.Resources {
		members := make([]*model.Member, len(groupMembers.Resources))
		membersIDValue := make([]patchValue, len(groupMembers.Resources))

		for j, member := range groupMembers.Resources {
			if member.SCIMID == "" {
				u, err := s.scim.GetUserByUserName(ctx, member.Email)
				if err != nil {
					return nil, fmt.Errorf("scim: error getting user by email: %w", err)
				}
				member.SCIMID = u.ID
			}

			membersIDValue[j] = patchValue{
				Value: member.SCIMID,
			}

			m := model.MemberBuilder().
				WithIPID(member.IPID).
				WithSCIMID(member.SCIMID).
				WithEmail(member.Email).
				WithStatus(member.Status).
				Build()

			slog.Warn("adding member to group", "group", groupMembers.Group.Name, "email", member.Email)
			members[j] = m
		}

		gm := model.GroupMembersBuilder().
			WithGroup(groupMembers.Group).
			WithResources(members).
			Build()

		groupsMembers[i] = gm

		patchOperations := patchGroupOperations("add", "members", membersIDValue, groupMembers)

		if len(patchOperations) > 1 {
			slog.Warn("group with more than 'max_members_per_request' members, sending multiple requests",
				"max_members_per_request", MaxPatchGroupMembersPerRequest,
				"group", groupMembers.Group.Name,
				"members", len(membersIDValue),
				"requests", len(patchOperations),
			)
		}

		for _, patchGroupRequest := range patchOperations {
			if err := s.scim.PatchGroup(ctx, patchGroupRequest); err != nil {
				return nil, fmt.Errorf("scim: error patching group: %w", err)
			}
		}
	}

	groupsMembersResult := model.GroupsMembersResultBuilder().WithResources(groupsMembers).Build()
	slog.Debug("scim: CreateGroupsMembers()", "groups_members", len(groupsMembers))

	return groupsMembersResult, nil
}

// DeleteGroupsMembers deletes groups members in SCIM Provider given a list of groups members
func (s *Provider) DeleteGroupsMembers(ctx context.Context, gmr *model.GroupsMembersResult) error {
	for _, groupMembers := range gmr.Resources {
		membersIDValue := []patchValue{}

		for _, member := range groupMembers.Resources {
			membersIDValue = append(membersIDValue, patchValue{
				Value: member.SCIMID,
			})
			slog.Warn("removing member from group", "group", groupMembers.Group.Name, "email", member.Email)
		}

		patchOperations := patchGroupOperations("remove", "members", membersIDValue, groupMembers)

		if len(patchOperations) > 1 {
			slog.Warn("group with more than 'max_members_per_request' members, sending multiple requests",
				"max_members_per_request", MaxPatchGroupMembersPerRequest,
				"group", groupMembers.Group.Name,
				"members", len(membersIDValue),
				"requests", len(patchOperations),
			)
		}

		for _, patchGroupRequest := range patchOperations {
			if err := s.scim.PatchGroup(ctx, patchGroupRequest); err != nil {
				return fmt.Errorf("scim: error patching group: %w", err)
			}
		}
	}

	return nil
}

// GetGroupsMembers returns a list of groups and their members from the SCIM Provider
// NOTE: this method doesn't work because unfortunately the SCIM API doesn't support
// list the members of a group, or get a group and their members at the same time
// reference: https://docs.aws.amazon.com/singlesignon/latest/developerguide/listgroups.html
func (s *Provider) GetGroupsMembers(ctx context.Context, gr *model.GroupsResult) (*model.GroupsMembersResult, error) {
	groupMembers := make([]*model.GroupMembers, 0)

	for _, group := range gr.Resources {
		// https://docs.aws.amazon.com/singlesignon/latest/developerguide/listgroups.html
		f := fmt.Sprintf("displayName eq %q", group.Name)
		lgr, err := s.scim.ListGroups(ctx, f)
		if err != nil {
			return nil, fmt.Errorf("scim: error listing groups: %w", err)
		}

		for _, gr := range lgr.Resources {
			members := make([]*model.Member, len(gr.Members))

			for j, member := range gr.Members {
				u, err := s.scim.GetUser(ctx, member.Value)
				if err != nil {
					return nil, fmt.Errorf("scim: error getting user: %s, error %w", member.Value, err)
				}

				m := model.MemberBuilder().
					WithSCIMID(member.Value).
					WithEmail(u.Emails[0].Value).
					Build()

				members[j] = m
			}

			gms := model.GroupMembersBuilder().
				WithGroup(group).
				WithResources(members).
				Build()

			groupMembers = append(groupMembers, gms)
		}
	}

	slog.Debug("scim: GetGroupsMembers()", "groups_members", len(groupMembers))
	groupsMembersResult := model.GroupsMembersResultBuilder().WithResources(groupMembers).Build()

	return groupsMembersResult, nil
}

// GetGroupsMembersBruteForce returns a list of groups and their members from the SCIM Provider
// NOTE: this is an bad alternative to the method GetGroupsMembers,  because read the note in the method.
func (s *Provider) GetGroupsMembersBruteForce(ctx context.Context, gr *model.GroupsResult, ur *model.UsersResult) (*model.GroupsMembersResult, error) {
	groupMembers := make([]*model.GroupMembers, len(gr.Resources))

	// brute force implemented here thanks to the fxxckin' aws sso scim api
	for i, group := range gr.Resources {
		members := make([]*model.Member, 0)

		for _, user := range ur.Resources {

			// https://docs.aws.amazon.com/singlesignon/latest/developerguide/listgroups.html
			filter := fmt.Sprintf("id eq %q and members eq %q", group.SCIMID, user.SCIMID)
			lgr, err := s.scim.ListGroups(ctx, filter)
			if err != nil {
				return nil, fmt.Errorf("scim: error listing groups: %w", err)
			}

			// AWS SSO SCIM API, it doesn't return the member into the Resources array
			if lgr.TotalResults > 0 {
				m := model.MemberBuilder().
					WithIPID(user.IPID).
					WithSCIMID(user.SCIMID).
					WithEmail(user.GetPrimaryEmailAddress()).
					Build()

				if user.Active {
					m.Status = "ACTIVE"
				}

				members = append(members, m)
			}
		}

		gms := model.GroupMembersBuilder().
			WithGroup(group).
			WithResources(members).
			Build()

		groupMembers[i] = gms
	}

	slog.Debug("scim: GetGroupsMembersBruteForce()", "groups_members", len(groupMembers))
	groupsMembersResult := model.GroupsMembersResultBuilder().WithResources(groupMembers).Build()

	return groupsMembersResult, nil
}
