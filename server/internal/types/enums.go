package types

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

// stringEnum is a generic type for string-backed enums stored in SQLite TEXT columns.
type stringEnum string

func (s *stringEnum) Scan(v any) error {
	switch v := v.(type) {
	case string:
		*s = stringEnum(v)
		return nil
	case nil:
		return errors.New("cannot scan NULL into string enum")
	default:
		return fmt.Errorf("cannot scan %T into string enum", v)
	}
}

func (s stringEnum) Value() (driver.Value, error) {
	return string(s), nil
}

// RecurrenceType describes how a task recurs.
type RecurrenceType string

const (
	RecurrenceTypeOneOff               RecurrenceType = "one_off"
	RecurrenceTypeCompletionBased      RecurrenceType = "completion_based"
	RecurrenceTypeFixedNonAccumulating RecurrenceType = "fixed_non_accumulating"
	RecurrenceTypeFixedAccumulating    RecurrenceType = "fixed_accumulating"
	RecurrenceTypeOnDependency         RecurrenceType = "on_dependency"
)

func AllRecurrenceTypes() []RecurrenceType {
	return []RecurrenceType{
		RecurrenceTypeOneOff,
		RecurrenceTypeCompletionBased,
		RecurrenceTypeFixedNonAccumulating,
		RecurrenceTypeFixedAccumulating,
		RecurrenceTypeOnDependency,
	}
}

func (s *RecurrenceType) Scan(v any) error { return (*stringEnum)(s).Scan(v) }

func (s RecurrenceType) Value() (driver.Value, error) { return stringEnum(s).Value() }

// StatusCategory classifies a task status.
type StatusCategory string

const (
	StatusCategoryInitial      StatusCategory = "initial"
	StatusCategoryIntermediate StatusCategory = "intermediate"
	StatusCategoryCompletion   StatusCategory = "completion"
)

func AllStatusCategories() []StatusCategory {
	return []StatusCategory{
		StatusCategoryInitial,
		StatusCategoryIntermediate,
		StatusCategoryCompletion,
	}
}

func (s *StatusCategory) Scan(v any) error { return (*stringEnum)(s).Scan(v) }

func (s StatusCategory) Value() (driver.Value, error) { return stringEnum(s).Value() }

// SpaceRole defines a user's role within a space.
type SpaceRole string

const (
	SpaceRoleAdmin  SpaceRole = "admin"
	SpaceRoleMember SpaceRole = "member"
	SpaceRoleViewer SpaceRole = "viewer"
)

func AllSpaceRoles() []SpaceRole {
	return []SpaceRole{
		SpaceRoleAdmin,
		SpaceRoleMember,
		SpaceRoleViewer,
	}
}

func (s *SpaceRole) Scan(v any) error { return (*stringEnum)(s).Scan(v) }

func (s SpaceRole) Value() (driver.Value, error) { return stringEnum(s).Value() }

// AuthTokenKind distinguishes session tokens from API tokens.
type AuthTokenKind string

const (
	AuthTokenKindSession AuthTokenKind = "session"
	AuthTokenKindAPI     AuthTokenKind = "api"
)

func AllAuthTokenKinds() []AuthTokenKind {
	return []AuthTokenKind{
		AuthTokenKindSession,
		AuthTokenKindAPI,
	}
}

func (s *AuthTokenKind) Scan(v any) error { return (*stringEnum)(s).Scan(v) }

func (s AuthTokenKind) Value() (driver.Value, error) { return stringEnum(s).Value() }

// StoredRelationKind is the canonical relation kind stored in the database.
// The API layer maps directed pairs (parent_of/child_of) to these canonical kinds.
type StoredRelationKind string

const (
	RelationKindParent     StoredRelationKind = "parent"
	RelationKindBlocks     StoredRelationKind = "blocks"
	RelationKindTriggers   StoredRelationKind = "triggers"
	RelationKindSpawns     StoredRelationKind = "spawns"
	RelationKindRelatesTo  StoredRelationKind = "relates_to"
	RelationKindDuplicates StoredRelationKind = "duplicates"
)

func AllStoredRelationKinds() []StoredRelationKind {
	return []StoredRelationKind{
		RelationKindParent,
		RelationKindBlocks,
		RelationKindTriggers,
		RelationKindSpawns,
		RelationKindRelatesTo,
		RelationKindDuplicates,
	}
}

// CopyOnSpawn reports whether relations of this kind should be copied
// when spawning a new task from a recurring template. Relations describing
// a task's role in a workflow (parent, blocks, triggers, relates_to) are
// copied; relations specific to a particular instance (spawns, duplicates)
// are not.
func (k StoredRelationKind) CopyOnSpawn() bool {
	switch k {
	case RelationKindParent, RelationKindBlocks, RelationKindTriggers, RelationKindRelatesTo:
		return true
	case RelationKindSpawns, RelationKindDuplicates:
		return false
	}
	return false
}

func (s *StoredRelationKind) Scan(v any) error { return (*stringEnum)(s).Scan(v) }

func (s StoredRelationKind) Value() (driver.Value, error) { return stringEnum(s).Value() }
