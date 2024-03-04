package spicedbx

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.infratographer.com/permissions-api/internal/types"
)

func TestSchema(t *testing.T) {
	t.Parallel()

	type testInput struct {
		namespace     string
		resourceTypes []types.ResourceType
	}

	type testResult struct {
		success string
		err     error
	}

	type testCase struct {
		name    string
		input   testInput
		checkFn func(*testing.T, testResult)
	}

	resourceTypes := []types.ResourceType{
		{
			Name: "user",
		},
		{
			Name: "client",
		},
		{
			Name: "role",
			Relationships: []types.ResourceTypeRelationship{
				{
					Relation: "subject",
					Types: []types.TargetType{
						{Name: "user"},
						{Name: "client"},
					},
				},
			},
		},
		{
			Name: "role_binding",
			Relationships: []types.ResourceTypeRelationship{
				{
					Relation: "parent",
					Types: []types.TargetType{
						{Name: "tenant"},
					},
				},
				{
					Relation: "loadbalancer_create_rel",
					Types: []types.TargetType{
						{Name: "role", SubjectRelation: "subject"},
					},
				},
				{
					Relation: "loadbalancer_get_rel",
					Types: []types.TargetType{
						{Name: "role", SubjectRelation: "subject"},
					},
				},
				{
					Relation: "port_create_rel",
					Types: []types.TargetType{
						{Name: "role", SubjectRelation: "subject"},
					},
				},
				{
					Relation: "port_get_rel",
					Types: []types.TargetType{
						{Name: "role", SubjectRelation: "subject"},
					},
				},
			},
			Actions: []types.Action{
				{
					Name: "loadbalancer_create",
					Conditions: []types.Condition{
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation: "loadbalancer_create_rel",
							},
							RoleBinding: &types.ConditionRoleBinding{},
						},
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation:   "role",
								ActionName: "loadbalancer_create",
							},
						},
					},
				},
				{
					Name: "loadbalancer_get",
					Conditions: []types.Condition{
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation: "loadbalancer_get_rel",
							},
							RoleBinding: &types.ConditionRoleBinding{},
						},
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation:   "parent",
								ActionName: "loadbalancer_get",
							},
						},
					},
				},
				{
					Name: "port_create",
					Conditions: []types.Condition{
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation: "port_create_rel",
							},
							RoleBinding: &types.ConditionRoleBinding{},
						},
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation:   "parent",
								ActionName: "port_create",
							},
						},
					},
				},
				{
					Name: "port_get",
					Conditions: []types.Condition{
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation: "port_get_rel",
							},
							RoleBinding: &types.ConditionRoleBinding{},
						},
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation:   "parent",
								ActionName: "port_get",
							},
						},
					},
				},
			},
		},
		{
			Name: "loadbalancer",
			Relationships: []types.ResourceTypeRelationship{
				{
					Relation: "owner",
					Types: []types.TargetType{
						{Name: "tenant"},
					},
				},
				{
					Relation: "loadbalancer_get_rel",
					Types: []types.TargetType{
						{Name: "role", SubjectRelation: "subject"},
					},
				},
			},
			Actions: []types.Action{
				{
					Name: "loadbalancer_get",
					Conditions: []types.Condition{
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation: "loadbalancer_get_rel",
							},
							RoleBinding: &types.ConditionRoleBinding{},
						},
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation:   "owner",
								ActionName: "loadbalancer_get",
							},
						},
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation:   "grant",
								ActionName: "loadbalancer_create",
							},
						},
					},
				},
			},
		},
		{
			Name: "port",
			Relationships: []types.ResourceTypeRelationship{
				{
					Relation: "owner",
					Types: []types.TargetType{
						{Name: "tenant"},
					},
				},
				{
					Relation: "port_get_rel",
					Types: []types.TargetType{
						{Name: "role", SubjectRelation: "subject"},
					},
				},
			},
			Actions: []types.Action{
				{
					Name: "port_get",
					Conditions: []types.Condition{
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation: "port_get_rel",
							},
							RoleBinding: &types.ConditionRoleBinding{},
						},
						{
							RelationshipAction: &types.ConditionRelationshipAction{
								Relation:   "owner",
								ActionName: "port_get",
							},
						},
					},
				},
			},
		},
	}

	schemaOutput := `definition foo/user {
}
definition foo/client {
}
definition foo/role {
    relation loadbalancer_create: foo/user:* | foo/client:*
    relation loadbalancer_get: foo/user:* | foo/client:*
    relation port_create: foo/user:* | foo/client:*
    relation port_get: foo/user:* | foo/client:*
}
definition foo/role_binding {
    relation role: foo/role
    relation subject: foo/user | foo/client | foo/group#member
    permission loadbalancer_create = role->loadbalancer_create
    permission loadbalancer_get = role->loadbalancer_get
    permission port_create = role->port_create
    permission port_get = role->port_get
}
definition foo/group {
    relation member: foo/user | foo/client | foo/group#member
    relation parent: foo/group | foo/tenant
    relation grant: foo/role_binding
    permission loadbalancer_create = parent->loadbalancer_create + grant->loadbalancer_create
    permission loadbalancer_get = grant->loadbalancer_get + parent->loadbalancer_get
    permission port_create = grant->port_create + parent->port_create
    permission port_get = grant->port_get + parent->port_get
}
definition foo/tenant {
    relation parent: foo/tenant
    relation member: foo/user | foo/client | foo/group#member | foo/tenant#member
    relation grant: foo/role_binding
    permission loadbalancer_create = parent->loadbalancer_create + grant->loadbalancer_create
    permission loadbalancer_get = grant->loadbalancer_get + parent->loadbalancer_get
    permission port_create = grant->port_create + parent->port_create
    permission port_get = grant->port_get + parent->port_get
}
definition foo/loadbalancer {
    relation owner: foo/tenant
    relation grant: foo/role_binding
    permission loadbalancer_get = owner->loadbalancer_get + grant->loadbalancer_create
}
definition foo/port {
    relation owner: foo/tenant
    relation grant: foo/role_binding
    permission port_get = grant->port_get + owner->port_get
}
`

	testCases := []testCase{
		{
			name: "NoNamespace",
			input: testInput{
				namespace:     "",
				resourceTypes: resourceTypes,
			},
			checkFn: func(t *testing.T, res testResult) {
				assert.ErrorIs(t, res.err, ErrorNoNamespace)
				assert.Empty(t, res.success)
			},
		},
		{
			name: "SucccessNamespace",
			input: testInput{
				namespace:     "foo",
				resourceTypes: resourceTypes,
			},
			checkFn: func(t *testing.T, res testResult) {
				assert.NoError(t, res.err)
				assert.Equal(t, schemaOutput, res.success)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var result testResult

			result.success, result.err = GenerateSchema(tc.input.namespace, tc.input.resourceTypes)

			tc.checkFn(t, result)
		})
	}
}
