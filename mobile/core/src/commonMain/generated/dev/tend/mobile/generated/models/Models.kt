/* 
 * NOTE: This file is auto generated. Do not edit the file manually!
 * 
 * Tend API
 * Version 0.0.0
 * 
 * Generated reproducibly; timestamp omitted.
 * OpenAPI KMP Gen (version 1.3.0) by kroegerama
 */
@file:Suppress("ArrayInDataClass", "RedundantVisibilityModifier", "unused", "ConstPropertyName")

package dev.tend.mobile.generated.models

import androidx.compose.runtime.Immutable
import kotlin.Boolean
import kotlin.Int
import kotlin.Long
import kotlin.String
import kotlin.Suppress
import kotlin.collections.List
import kotlin.collections.emptyList
import kotlin.time.Instant
import kotlinx.datetime.LocalDate
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
@Immutable
public enum class ActivityAction {
  @SerialName("created")
  CREATED,
  @SerialName("updated")
  UPDATED,
  @SerialName("deleted")
  DELETED,
}

@Serializable
@Immutable
public data class ActivityDetail(
  @SerialName("field")
  public val `field`: String,
  @SerialName("from")
  public val from: String,
  @SerialName("to")
  public val to: String,
)

@Serializable
@Immutable
public enum class ActivityEntityType {
  @SerialName("task")
  TASK,
  @SerialName("space")
  SPACE,
  @SerialName("member")
  MEMBER,
  @SerialName("tag")
  TAG,
  @SerialName("status")
  STATUS,
  @SerialName("effort_level")
  EFFORT_LEVEL,
  @SerialName("priority_level")
  PRIORITY_LEVEL,
  @SerialName("relation")
  RELATION,
}

@Serializable
@Immutable
public data class ActivityLogEntry(
  @SerialName("id")
  public val id: String,
  @SerialName("spaceSlug")
  public val spaceSlug: String,
  @SerialName("actorId")
  public val actorId: String,
  @SerialName("tokenId")
  public val tokenId: String,
  @SerialName("tokenName")
  public val tokenName: String,
  @SerialName("entityType")
  public val entityType: ActivityEntityType,
  @SerialName("entityId")
  public val entityId: String,
  @SerialName("action")
  public val action: ActivityAction,
  @SerialName("details")
  public val details: List<ActivityDetail> = emptyList(),
  @SerialName("createdAt")
  public val createdAt: Instant,
)

@Serializable
@Immutable
public data class ActivityLogPage(
  @SerialName("items")
  public val items: List<ActivityLogEntry> = emptyList(),
  @SerialName("nextCursor")
  public val nextCursor: String,
)

@Serializable
@Immutable
public data class ApiError(
  @SerialName("code")
  public val code: String,
  @SerialName("message")
  public val message: String,
)

@Serializable
@Immutable
public data class AuthToken(
  @SerialName("id")
  public val id: String,
  @SerialName("name")
  public val name: String,
  @SerialName("kind")
  public val kind: AuthTokenKind,
  @SerialName("createdAt")
  public val createdAt: Instant,
)

@Serializable
@Immutable
public data class AuthTokenCreate(
  @SerialName("name")
  public val name: String,
)

@Serializable
@Immutable
public data class AuthTokenCreateResponse(
  @SerialName("token")
  public val token: String,
  @SerialName("authToken")
  public val authToken: AuthToken,
)

@Serializable
@Immutable
public enum class AuthTokenKind {
  @SerialName("session")
  SESSION,
  @SerialName("api")
  API,
}

@Serializable
@Immutable
public data class AuthTokenList(
  @SerialName("items")
  public val items: List<AuthToken> = emptyList(),
)

@Serializable
@Immutable
public data class Space(
  @SerialName("slug")
  public val slug: String,
  @SerialName("name")
  public val name: String,
  @SerialName("description")
  public val description: String,
  @SerialName("createdAt")
  public val createdAt: Instant,
  @SerialName("updatedAt")
  public val updatedAt: Instant,
)

@Serializable
@Immutable
public data class SpaceCreate(
  @SerialName("slug")
  public val slug: String,
  @SerialName("name")
  public val name: String,
  @SerialName("description")
  public val description: String? = null,
)

@Serializable
@Immutable
public data class SpaceList(
  @SerialName("items")
  public val items: List<Space> = emptyList(),
)

@Serializable
@Immutable
public data class SpaceMember(
  @SerialName("userId")
  public val userId: String,
  @SerialName("userName")
  public val userName: String,
  @SerialName("userEmail")
  public val userEmail: String,
  @SerialName("role")
  public val role: SpaceRole,
  @SerialName("createdAt")
  public val createdAt: Instant,
)

@Serializable
@Immutable
public data class SpaceMemberCreate(
  @SerialName("userId")
  public val userId: String,
  @SerialName("role")
  public val role: SpaceRole,
)

@Serializable
@Immutable
public data class SpaceMemberList(
  @SerialName("items")
  public val items: List<SpaceMember> = emptyList(),
)

@Serializable
@Immutable
public data class SpaceMemberUpdate(
  @SerialName("role")
  public val role: SpaceRole,
)

@Serializable
@Immutable
public enum class SpaceRole {
  @SerialName("admin")
  ADMIN,
  @SerialName("member")
  MEMBER,
  @SerialName("viewer")
  VIEWER,
}

@Serializable
@Immutable
public data class SpaceUpdate(
  @SerialName("slug")
  public val slug: String? = null,
  @SerialName("name")
  public val name: String? = null,
  @SerialName("description")
  public val description: String? = null,
)

@Serializable
@Immutable
public data class Tag(
  @SerialName("name")
  public val name: String,
  @SerialName("createdAt")
  public val createdAt: Instant,
)

@Serializable
@Immutable
public data class TagCreate(
  @SerialName("name")
  public val name: String,
)

@Serializable
@Immutable
public data class TagList(
  @SerialName("items")
  public val items: List<Tag> = emptyList(),
)

@Serializable
@Immutable
public data class TagUpdate(
  @SerialName("name")
  public val name: String,
)

@Serializable
@Immutable
public data class Task(
  @SerialName("id")
  public val id: String,
  @SerialName("spaceSlug")
  public val spaceSlug: String,
  @SerialName("title")
  public val title: String,
  @SerialName("description")
  public val description: String,
  @SerialName("status")
  public val status: String,
  @SerialName("effort")
  public val effort: String,
  @SerialName("priority")
  public val priority: String,
  @SerialName("recurrenceType")
  public val recurrenceType: TaskRecurrenceType,
  @SerialName("recurrenceRule")
  public val recurrenceRule: String,
  @SerialName("lastCompletedAt")
  public val lastCompletedAt: Instant,
  @SerialName("assigneeIds")
  public val assigneeIds: List<String> = emptyList(),
  @SerialName("rotationPool")
  public val rotationPool: List<String> = emptyList(),
  @SerialName("tags")
  public val tags: List<String> = emptyList(),
  @SerialName("relations")
  public val relations: List<TaskRelation> = emptyList(),
  @SerialName("due")
  public val due: TaskDue,
  @SerialName("overdueActionRule")
  public val overdueActionRule: TaskOverdueActionRule,
  @SerialName("createdAt")
  public val createdAt: Instant,
  @SerialName("updatedAt")
  public val updatedAt: Instant,
)

@Serializable
@Immutable
public data class TaskCreate(
  /**
   * Task title (1–500 chars).
   */
  @SerialName("title")
  public val title: String,
  /**
   * Task description (max 10000 chars).
   */
  @SerialName("description")
  public val description: String? = null,
  /**
   * Initial status name. Defaults to the space's first status.
   */
  @SerialName("status")
  public val status: String? = null,
  /**
   * Effort level name.
   */
  @SerialName("effort")
  public val effort: String? = null,
  /**
   * Priority level name.
   */
  @SerialName("priority")
  public val priority: String? = null,
  /**
   * Recurrence type (e.g. one_off, completion_based, fixed_accumulating).
   */
  @SerialName("recurrenceType")
  public val recurrenceType: TaskRecurrenceType? = null,
  /**
   * Recurrence rule (RRULE string, max 500 chars).
   */
  @SerialName("recurrenceRule")
  public val recurrenceRule: String? = null,
  @SerialName("assigneeIds")
  public val assigneeIds: List<String>? = null,
  @SerialName("rotationPool")
  public val rotationPool: List<String>? = null,
  @SerialName("tags")
  public val tags: List<String>? = null,
  @SerialName("due")
  public val due: TaskDue? = null,
  @SerialName("overdueActionRule")
  public val overdueActionRule: TaskOverdueActionRule? = null,
)

@Serializable
@Immutable
public data class TaskDue(
  @SerialName("at")
  public val at: LocalDate,
  @SerialName("timezone")
  public val timezone: String,
)

@Serializable
@Immutable
public data class TaskEffortLevel(
  @SerialName("name")
  public val name: String,
  @SerialName("position")
  public val position: Long,
)

@Serializable
@Immutable
public data class TaskEffortLevelInput(
  @SerialName("name")
  public val name: String,
)

@Serializable
@Immutable
public data class TaskEffortLevelList(
  @SerialName("items")
  public val items: List<TaskEffortLevel> = emptyList(),
)

@Serializable
@Immutable
public data class TaskEffortLevelReplace(
  @SerialName("items")
  public val items: List<TaskEffortLevelInput> = emptyList(),
)

@Serializable
@Immutable
public enum class TaskOverdueAction {
  @SerialName("advance_recurrence")
  ADVANCE_RECURRENCE,
  @SerialName("set_status")
  SET_STATUS,
  @SerialName("clear_due_date")
  CLEAR_DUE_DATE,
}

@Serializable
@Immutable
public data class TaskOverdueActionRule(
  /**
   * Grace period in days after the due date before the action fires.
   *       null means act immediately when the task becomes overdue.
   */
  @SerialName("after")
  public val after: Int,
  @SerialName("action")
  public val action: TaskOverdueAction,
  /**
   * Required when action is set_status; the status name to transition to.
   */
  @SerialName("status")
  public val status: String? = null,
)

@Serializable
@Immutable
public data class TaskPage(
  @SerialName("items")
  public val items: List<Task> = emptyList(),
  @SerialName("nextCursor")
  public val nextCursor: String,
)

@Serializable
@Immutable
public data class TaskPriorityLevel(
  @SerialName("name")
  public val name: String,
  @SerialName("position")
  public val position: Long,
)

@Serializable
@Immutable
public data class TaskPriorityLevelInput(
  @SerialName("name")
  public val name: String,
)

@Serializable
@Immutable
public data class TaskPriorityLevelList(
  @SerialName("items")
  public val items: List<TaskPriorityLevel> = emptyList(),
)

@Serializable
@Immutable
public data class TaskPriorityLevelReplace(
  @SerialName("items")
  public val items: List<TaskPriorityLevelInput> = emptyList(),
)

@Serializable
@Immutable
public enum class TaskRecurrenceType {
  @SerialName("one_off")
  ONE_OFF,
  @SerialName("completion_based")
  COMPLETION_BASED,
  @SerialName("fixed_non_accumulating")
  FIXED_NON_ACCUMULATING,
  @SerialName("fixed_accumulating")
  FIXED_ACCUMULATING,
  @SerialName("on_dependency")
  ON_DEPENDENCY,
}

@Serializable
@Immutable
public data class TaskRelation(
  @SerialName("kind")
  public val kind: TaskRelationKind,
  @SerialName("relatedTaskId")
  public val relatedTaskId: String,
  @SerialName("createdAt")
  public val createdAt: Instant,
)

@Serializable
@Immutable
public data class TaskRelationCreate(
  /**
   * Relation kind (e.g. parent_of, child_of, blocks, blocked_by, relates_to, duplicates, triggers, triggered_by).
   */
  @SerialName("kind")
  public val kind: TaskRelationKind,
  /**
   * Task ID of the related task.
   */
  @SerialName("relatedTaskId")
  public val relatedTaskId: String,
)

@Serializable
@Immutable
public enum class TaskRelationKind {
  @SerialName("parent_of")
  PARENT_OF,
  @SerialName("child_of")
  CHILD_OF,
  @SerialName("blocks")
  BLOCKS,
  @SerialName("blocked_by")
  BLOCKED_BY,
  @SerialName("relates_to")
  RELATES_TO,
  @SerialName("duplicates")
  DUPLICATES,
  @SerialName("triggers")
  TRIGGERS,
  @SerialName("triggered_by")
  TRIGGERED_BY,
  @SerialName("spawns")
  SPAWNS,
  @SerialName("spawned_by")
  SPAWNED_BY,
}

@Serializable
@Immutable
public data class TaskStatus(
  @SerialName("name")
  public val name: String,
  @SerialName("category")
  public val category: TaskStatusCategory,
  @SerialName("position")
  public val position: Long,
)

@Serializable
@Immutable
public enum class TaskStatusCategory {
  @SerialName("initial")
  INITIAL,
  @SerialName("intermediate")
  INTERMEDIATE,
  @SerialName("completion")
  COMPLETION,
}

@Serializable
@Immutable
public data class TaskStatusInput(
  @SerialName("name")
  public val name: String,
  @SerialName("category")
  public val category: TaskStatusCategory,
)

@Serializable
@Immutable
public data class TaskStatusList(
  @SerialName("items")
  public val items: List<TaskStatus> = emptyList(),
)

@Serializable
@Immutable
public data class TaskStatusReplace(
  @SerialName("items")
  public val items: List<TaskStatusInput> = emptyList(),
)

@Serializable
@Immutable
public data class TaskUpdate(
  /**
   * New title.
   */
  @SerialName("title")
  public val title: String? = null,
  /**
   * New description.
   */
  @SerialName("description")
  public val description: String? = null,
  /**
   * New status name.
   */
  @SerialName("status")
  public val status: String? = null,
  /**
   * New effort level name.
   */
  @SerialName("effort")
  public val effort: String? = null,
  /**
   * New priority level name.
   */
  @SerialName("priority")
  public val priority: String? = null,
  /**
   * New recurrence type.
   */
  @SerialName("recurrenceType")
  public val recurrenceType: TaskRecurrenceType? = null,
  /**
   * New recurrence rule (RRULE string, max 500 chars).
   */
  @SerialName("recurrenceRule")
  public val recurrenceRule: String? = null,
  @SerialName("assigneeIds")
  public val assigneeIds: List<String>? = null,
  @SerialName("rotationPool")
  public val rotationPool: List<String>? = null,
  @SerialName("tags")
  public val tags: List<String>? = null,
  @SerialName("due")
  public val due: TaskDue? = null,
  @SerialName("overdueActionRule")
  public val overdueActionRule: TaskOverdueActionRule? = null,
)

@Serializable
@Immutable
public data class User(
  @SerialName("id")
  public val id: String,
  @SerialName("email")
  public val email: String,
  @SerialName("name")
  public val name: String,
  @SerialName("isOwner")
  public val isOwner: Boolean,
  @SerialName("hasPassword")
  public val hasPassword: Boolean,
  @SerialName("createdAt")
  public val createdAt: Instant,
  @SerialName("updatedAt")
  public val updatedAt: Instant,
)

@Serializable
@Immutable
public data class UserCreate(
  @SerialName("name")
  public val name: String,
  @SerialName("email")
  public val email: String,
  @SerialName("isOwner")
  public val isOwner: Boolean? = null,
  @SerialName("password")
  public val password: String? = null,
)

@Serializable
@Immutable
public data class UserList(
  @SerialName("items")
  public val items: List<User> = emptyList(),
)

@Serializable
@Immutable
public data class UserUpdate(
  @SerialName("name")
  public val name: String? = null,
  @SerialName("email")
  public val email: String? = null,
  @SerialName("isOwner")
  public val isOwner: Boolean? = null,
  @SerialName("setPassword")
  public val setPassword: String? = null,
  @SerialName("clearPassword")
  public val clearPassword: Boolean? = null,
)
