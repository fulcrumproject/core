package authz

import (
	"github.com/fulcrumproject/core/pkg/auth"
)

const (
	ObjectTypeParticipant       ObjectType = "participant"
	ObjectTypeAgent             ObjectType = "agent"
	ObjectTypeAgentType         ObjectType = "agent_type"
	ObjectTypeService           ObjectType = "service"
	ObjectTypeServiceType       ObjectType = "service_type"
	ObjectTypeServiceGroup      ObjectType = "service_group"
	ObjectTypeServiceOptionType ObjectType = "service_option_type"
	ObjectTypeServiceOption     ObjectType = "service_option"
	ObjectTypeServicePoolSet    ObjectType = "service_pool_set"
	ObjectTypeServicePool       ObjectType = "service_pool"
	ObjectTypeServicePoolValue  ObjectType = "service_pool_value"
	ObjectTypeJob               ObjectType = "job"
	ObjectTypeMetricType        ObjectType = "metric_type"
	ObjectTypeMetricEntry       ObjectType = "metric_entry"
	ObjectTypeEvent             ObjectType = "event_entry"
	ObjectTypeToken             ObjectType = "token"
)

const (
	// Standard CRUD actions
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"

	// Special actions
	ActionUpdateStatus  Action = "update_status"
	ActionGenerateToken Action = "generate_token"
	ActionClaim         Action = "claim"
	ActionComplete      Action = "complete"
	ActionFail          Action = "fail"
	ActionListPending   Action = "list_pending"
	ActionLease         Action = "lease"
	ActionAck           Action = "ack"
)

// Default authorization rules for the system
var Rules = []AuthorizationRule{
	// Participant permissions
	{Object: ObjectTypeParticipant, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeParticipant, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeParticipant, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeParticipant, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin}},

	// Agent permissions
	{Object: ObjectTypeAgent, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeAgent, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeAgent, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeAgent, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeAgent, Action: ActionUpdateStatus, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},

	// AgentType permissions
	{Object: ObjectTypeAgentType, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeAgentType, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeAgentType, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeAgentType, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin}},

	// Service permissions
	{Object: ObjectTypeService, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeService, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeService, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeService, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},

	// ServiceType permissions
	{Object: ObjectTypeServiceType, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeServiceType, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeServiceType, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeServiceType, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin}},

	// ServiceGroup permissions
	{Object: ObjectTypeServiceGroup, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServiceGroup, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServiceGroup, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServiceGroup, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},

	// ServiceOptionType permissions (global resources - types readable by all, writable by admin only)
	{Object: ObjectTypeServiceOptionType, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeServiceOptionType, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeServiceOptionType, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeServiceOptionType, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin}},

	// ServiceOption permissions (provider-scoped - admin, participant for own provider, agent for own provider)
	{Object: ObjectTypeServiceOption, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeServiceOption, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeServiceOption, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeServiceOption, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},

	// ServicePoolSet permissions (provider-scoped - admin, participant for own provider)
	{Object: ObjectTypeServicePoolSet, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServicePoolSet, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServicePoolSet, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServicePoolSet, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},

	// ServicePool permissions (provider-scoped via pool set - admin, participant for own provider)
	{Object: ObjectTypeServicePool, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeServicePool, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServicePool, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServicePool, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},

	// ServicePoolValue permissions (provider-scoped via pool - admin, participant for own provider, agent for own provider)
	{Object: ObjectTypeServicePoolValue, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeServicePoolValue, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServicePoolValue, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeServicePoolValue, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},

	// Job permissions
	{Object: ObjectTypeJob, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeJob, Action: ActionClaim, Roles: []auth.Role{auth.RoleAgent}},
	{Object: ObjectTypeJob, Action: ActionComplete, Roles: []auth.Role{auth.RoleAgent}},
	{Object: ObjectTypeJob, Action: ActionFail, Roles: []auth.Role{auth.RoleAgent}},
	{Object: ObjectTypeJob, Action: ActionListPending, Roles: []auth.Role{auth.RoleAgent}},

	// MetricType permissions
	{Object: ObjectTypeMetricType, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent}},
	{Object: ObjectTypeMetricType, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeMetricType, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeMetricType, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin}},

	// MetricEntry permissions
	{Object: ObjectTypeMetricEntry, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeMetricEntry, Action: ActionCreate, Roles: []auth.Role{auth.RoleAgent}},

	// Event permissions
	{Object: ObjectTypeEvent, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeEvent, Action: ActionLease, Roles: []auth.Role{auth.RoleAdmin}},
	{Object: ObjectTypeEvent, Action: ActionAck, Roles: []auth.Role{auth.RoleAdmin}},

	// Token permissions
	{Object: ObjectTypeToken, Action: ActionRead, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeToken, Action: ActionCreate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeToken, Action: ActionUpdate, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeToken, Action: ActionDelete, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
	{Object: ObjectTypeToken, Action: ActionGenerateToken, Roles: []auth.Role{auth.RoleAdmin, auth.RoleParticipant}},
}
