/* 
 * NOTE: This file is auto generated. Do not edit the file manually!
 * 
 * Horologia API
 * Version 0.0.0
 * 
 * Generated reproducibly; timestamp omitted.
 * OpenAPI KMP Gen (version 1.3.0) by kroegerama
 */
@file:Suppress("ArrayInDataClass", "RedundantVisibilityModifier", "unused", "ConstPropertyName")

package dev.horologia.mobile.generated.api

import arrow.core.Either
import com.kroegerama.openapi.kmp.gen.`companion`.AuthPlugin.Plugin.authKeys
import com.kroegerama.openapi.kmp.gen.`companion`.CallException
import com.kroegerama.openapi.kmp.gen.`companion`.HttpCallResponse
import com.kroegerama.openapi.kmp.gen.`companion`.appendSerializedQueryParameter
import com.kroegerama.openapi.kmp.gen.`companion`.createSerializedPathSegment
import com.kroegerama.openapi.kmp.gen.`companion`.eitherRequest
import dev.horologia.mobile.generated.Api
import dev.horologia.mobile.generated.Auth
import dev.horologia.mobile.generated.models.ActivityLogPage
import dev.horologia.mobile.generated.models.AuthTokenCreate
import dev.horologia.mobile.generated.models.AuthTokenCreateResponse
import dev.horologia.mobile.generated.models.AuthTokenList
import dev.horologia.mobile.generated.models.Space
import dev.horologia.mobile.generated.models.SpaceCreate
import dev.horologia.mobile.generated.models.SpaceList
import dev.horologia.mobile.generated.models.SpaceMember
import dev.horologia.mobile.generated.models.SpaceMemberCreate
import dev.horologia.mobile.generated.models.SpaceMemberList
import dev.horologia.mobile.generated.models.SpaceMemberUpdate
import dev.horologia.mobile.generated.models.SpaceUpdate
import dev.horologia.mobile.generated.models.Tag
import dev.horologia.mobile.generated.models.TagCreate
import dev.horologia.mobile.generated.models.TagList
import dev.horologia.mobile.generated.models.TagUpdate
import dev.horologia.mobile.generated.models.Task
import dev.horologia.mobile.generated.models.TaskCreate
import dev.horologia.mobile.generated.models.TaskEffortLevelList
import dev.horologia.mobile.generated.models.TaskEffortLevelReplace
import dev.horologia.mobile.generated.models.TaskPage
import dev.horologia.mobile.generated.models.TaskPriorityLevelList
import dev.horologia.mobile.generated.models.TaskPriorityLevelReplace
import dev.horologia.mobile.generated.models.TaskRelation
import dev.horologia.mobile.generated.models.TaskRelationCreate
import dev.horologia.mobile.generated.models.TaskRelationKind
import dev.horologia.mobile.generated.models.TaskSearchResultList
import dev.horologia.mobile.generated.models.TaskStatusList
import dev.horologia.mobile.generated.models.TaskStatusReplace
import dev.horologia.mobile.generated.models.TaskUpdate
import dev.horologia.mobile.generated.models.User
import dev.horologia.mobile.generated.models.UserCreate
import dev.horologia.mobile.generated.models.UserList
import dev.horologia.mobile.generated.models.UserUpdate
import io.ktor.client.request.HttpRequestBuilder
import io.ktor.client.request.setBody
import io.ktor.http.ContentType
import io.ktor.http.HttpMethod
import io.ktor.http.appendPathSegments
import io.ktor.http.contentType
import kotlin.Int
import kotlin.String
import kotlin.Suppress
import kotlin.Unit

public object AuthApi {
  /**
   * `GET /auth/tokens`
   *
   * @return The request has succeeded.
   */
  public suspend fun authListTokens(decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<AuthTokenList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "auth",
      "tokens",
    )
    decorator()
  }

  /**
   * `POST /auth/tokens`
   *
   * @return The request has succeeded and a new resource has been created as a result.
   */
  public suspend fun authCreateToken(body: AuthTokenCreate, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<AuthTokenCreateResponse>> = Api.client.eitherRequest {
    method = HttpMethod.parse("POST")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "auth",
      "tokens",
    )
    setBody(body)
    decorator()
  }

  /**
   * `DELETE /auth/tokens/{tokenId}`
   *
   * @param tokenId Token ID.
   * @return There is no content to send for this request, but the headers may be useful. 
   */
  public suspend fun authDeleteToken(tokenId: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<Unit>> = Api.client.eitherRequest {
    method = HttpMethod.parse("DELETE")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "auth",
      "tokens",
      createSerializedPathSegment(value = tokenId, explode = false, json = Api.json),
    )
    decorator()
  }
}

public object SpacesApi {
  /**
   * `GET /spaces`
   *
   * @return The request has succeeded.
   */
  public suspend fun spacesList(decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<SpaceList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
    )
    decorator()
  }

  /**
   * `POST /spaces`
   *
   * @return The request has succeeded and a new resource has been created as a result.
   */
  public suspend fun spacesCreate(body: SpaceCreate, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<Space>> = Api.client.eitherRequest {
    method = HttpMethod.parse("POST")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
    )
    setBody(body)
    decorator()
  }

  /**
   * `GET /spaces/{spaceSlug}`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spacesRead(spaceSlug: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<Space>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
    )
    decorator()
  }

  /**
   * `DELETE /spaces/{spaceSlug}`
   *
   * @param spaceSlug Slug of the space.
   * @return There is no content to send for this request, but the headers may be useful. 
   */
  public suspend fun spacesDelete(spaceSlug: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<Unit>> = Api.client.eitherRequest {
    method = HttpMethod.parse("DELETE")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
    )
    decorator()
  }

  /**
   * `PATCH /spaces/{spaceSlug}`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spacesUpdate(
    spaceSlug: String,
    body: SpaceUpdate,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Space>> = Api.client.eitherRequest {
    method = HttpMethod.parse("PATCH")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
    )
    setBody(body)
    decorator()
  }
}

public object ActivityApi {
  /**
   * `GET /spaces/{spaceSlug}/activity`
   *
   * @param spaceSlug Slug of the space.
   * @param cursor Pagination cursor from a previous response.
   * @param limit Maximum number of items to return (1–100).
   * @return The request has succeeded.
   */
  public suspend fun spaceActivityList(
    spaceSlug: String,
    cursor: String? = null,
    limit: Int? = null,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<ActivityLogPage>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "activity",
    )
    appendSerializedQueryParameter(name = "cursor", value = cursor, explode = false, json = Api.json)
    appendSerializedQueryParameter(name = "limit", value = limit, explode = false, json = Api.json)
    decorator()
  }

  /**
   * `GET /spaces/{spaceSlug}/tasks/{taskId}/activity`
   *
   * @param spaceSlug Slug of the space.
   * @param taskId Task ID.
   * @param cursor Pagination cursor from a previous response.
   * @param limit Maximum number of items to return (1–100).
   * @return The request has succeeded.
   */
  public suspend fun spaceTaskActivityList(
    spaceSlug: String,
    taskId: String,
    cursor: String? = null,
    limit: Int? = null,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<ActivityLogPage>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tasks",
      createSerializedPathSegment(value = taskId, explode = false, json = Api.json),
      "activity",
    )
    appendSerializedQueryParameter(name = "cursor", value = cursor, explode = false, json = Api.json)
    appendSerializedQueryParameter(name = "limit", value = limit, explode = false, json = Api.json)
    decorator()
  }

  /**
   * `GET /users/{userId}/activity`
   *
   * @param userId User ID.
   * @param cursor Pagination cursor from a previous response.
   * @param limit Maximum number of items to return (1–100).
   * @return The request has succeeded.
   */
  public suspend fun userActivityList(
    userId: String,
    cursor: String? = null,
    limit: Int? = null,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<ActivityLogPage>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "users",
      createSerializedPathSegment(value = userId, explode = false, json = Api.json),
      "activity",
    )
    appendSerializedQueryParameter(name = "cursor", value = cursor, explode = false, json = Api.json)
    appendSerializedQueryParameter(name = "limit", value = limit, explode = false, json = Api.json)
    decorator()
  }
}

public object SpaceMembersApi {
  /**
   * `GET /spaces/{spaceSlug}/members`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spaceMembersList(spaceSlug: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<SpaceMemberList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "members",
    )
    decorator()
  }

  /**
   * `POST /spaces/{spaceSlug}/members`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded and a new resource has been created as a result.
   */
  public suspend fun spaceMembersCreate(
    spaceSlug: String,
    body: SpaceMemberCreate,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<SpaceMember>> = Api.client.eitherRequest {
    method = HttpMethod.parse("POST")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "members",
    )
    setBody(body)
    decorator()
  }

  /**
   * `DELETE /spaces/{spaceSlug}/members/{userId}`
   *
   * @param spaceSlug Slug of the space.
   * @param userId User ID of the member.
   * @return There is no content to send for this request, but the headers may be useful. 
   */
  public suspend fun spaceMembersDelete(
    spaceSlug: String,
    userId: String,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Unit>> = Api.client.eitherRequest {
    method = HttpMethod.parse("DELETE")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "members",
      createSerializedPathSegment(value = userId, explode = false, json = Api.json),
    )
    decorator()
  }

  /**
   * `PATCH /spaces/{spaceSlug}/members/{userId}`
   *
   * @param spaceSlug Slug of the space.
   * @param userId User ID of the member.
   * @return The request has succeeded.
   */
  public suspend fun spaceMembersUpdate(
    spaceSlug: String,
    userId: String,
    body: SpaceMemberUpdate,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<SpaceMember>> = Api.client.eitherRequest {
    method = HttpMethod.parse("PATCH")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "members",
      createSerializedPathSegment(value = userId, explode = false, json = Api.json),
    )
    setBody(body)
    decorator()
  }
}

public object TagsApi {
  /**
   * `GET /spaces/{spaceSlug}/tags`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spaceTagsList(spaceSlug: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<TagList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tags",
    )
    decorator()
  }

  /**
   * `POST /spaces/{spaceSlug}/tags`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded and a new resource has been created as a result.
   */
  public suspend fun spaceTagsCreate(
    spaceSlug: String,
    body: TagCreate,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Tag>> = Api.client.eitherRequest {
    method = HttpMethod.parse("POST")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tags",
    )
    setBody(body)
    decorator()
  }

  /**
   * `DELETE /spaces/{spaceSlug}/tags/{tagName}`
   *
   * @param spaceSlug Slug of the space.
   * @param tagName Tag name to delete.
   * @return There is no content to send for this request, but the headers may be useful. 
   */
  public suspend fun spaceTagsDelete(
    spaceSlug: String,
    tagName: String,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Unit>> = Api.client.eitherRequest {
    method = HttpMethod.parse("DELETE")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tags",
      createSerializedPathSegment(value = tagName, explode = false, json = Api.json),
    )
    decorator()
  }

  /**
   * `PATCH /spaces/{spaceSlug}/tags/{tagName}`
   *
   * @param spaceSlug Slug of the space.
   * @param tagName Current tag name.
   * @return The request has succeeded.
   */
  public suspend fun spaceTagsUpdate(
    spaceSlug: String,
    tagName: String,
    body: TagUpdate,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Tag>> = Api.client.eitherRequest {
    method = HttpMethod.parse("PATCH")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tags",
      createSerializedPathSegment(value = tagName, explode = false, json = Api.json),
    )
    setBody(body)
    decorator()
  }
}

public object TaskEffortLevelsApi {
  /**
   * `GET /spaces/{spaceSlug}/task-effort-levels`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spaceTaskEffortLevelsList(spaceSlug: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<TaskEffortLevelList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "task-effort-levels",
    )
    decorator()
  }

  /**
   * `PUT /spaces/{spaceSlug}/task-effort-levels`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spaceTaskEffortLevelsReplace(
    spaceSlug: String,
    body: TaskEffortLevelReplace,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<TaskEffortLevelList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("PUT")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "task-effort-levels",
    )
    setBody(body)
    decorator()
  }
}

public object TaskPriorityLevelsApi {
  /**
   * `GET /spaces/{spaceSlug}/task-priority-levels`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spaceTaskPriorityLevelsList(spaceSlug: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<TaskPriorityLevelList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "task-priority-levels",
    )
    decorator()
  }

  /**
   * `PUT /spaces/{spaceSlug}/task-priority-levels`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spaceTaskPriorityLevelsReplace(
    spaceSlug: String,
    body: TaskPriorityLevelReplace,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<TaskPriorityLevelList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("PUT")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "task-priority-levels",
    )
    setBody(body)
    decorator()
  }
}

public object TaskStatusesApi {
  /**
   * `GET /spaces/{spaceSlug}/task-statuses`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spaceTaskStatusesList(spaceSlug: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<TaskStatusList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "task-statuses",
    )
    decorator()
  }

  /**
   * `PUT /spaces/{spaceSlug}/task-statuses`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded.
   */
  public suspend fun spaceTaskStatusesReplace(
    spaceSlug: String,
    body: TaskStatusReplace,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<TaskStatusList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("PUT")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "task-statuses",
    )
    setBody(body)
    decorator()
  }
}

public object TasksApi {
  /**
   * `GET /spaces/{spaceSlug}/tasks`
   *
   * @param spaceSlug Slug of the space.
   * @param cursor Pagination cursor from a previous response.
   * @param limit Maximum number of items to return (1–100).
   * @return The request has succeeded.
   */
  public suspend fun spaceTasksList(
    spaceSlug: String,
    cursor: String? = null,
    limit: Int? = null,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<TaskPage>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tasks",
    )
    appendSerializedQueryParameter(name = "cursor", value = cursor, explode = false, json = Api.json)
    appendSerializedQueryParameter(name = "limit", value = limit, explode = false, json = Api.json)
    decorator()
  }

  /**
   * `POST /spaces/{spaceSlug}/tasks`
   *
   * @param spaceSlug Slug of the space.
   * @return The request has succeeded and a new resource has been created as a result.
   */
  public suspend fun spaceTasksCreate(
    spaceSlug: String,
    body: TaskCreate,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Task>> = Api.client.eitherRequest {
    method = HttpMethod.parse("POST")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tasks",
    )
    setBody(body)
    decorator()
  }

  /**
   * `GET /spaces/{spaceSlug}/tasks/{taskId}`
   *
   * @param spaceSlug Slug of the space.
   * @param taskId Task ID.
   * @return The request has succeeded.
   */
  public suspend fun spaceTasksRead(
    spaceSlug: String,
    taskId: String,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Task>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tasks",
      createSerializedPathSegment(value = taskId, explode = false, json = Api.json),
    )
    decorator()
  }

  /**
   * `DELETE /spaces/{spaceSlug}/tasks/{taskId}`
   *
   * @param spaceSlug Slug of the space.
   * @param taskId Task ID.
   * @return There is no content to send for this request, but the headers may be useful. 
   */
  public suspend fun spaceTasksDelete(
    spaceSlug: String,
    taskId: String,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Unit>> = Api.client.eitherRequest {
    method = HttpMethod.parse("DELETE")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tasks",
      createSerializedPathSegment(value = taskId, explode = false, json = Api.json),
    )
    decorator()
  }

  /**
   * `PATCH /spaces/{spaceSlug}/tasks/{taskId}`
   *
   * @param spaceSlug Slug of the space.
   * @param taskId Task ID.
   * @return The request has succeeded.
   */
  public suspend fun spaceTasksUpdate(
    spaceSlug: String,
    taskId: String,
    body: TaskUpdate,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Task>> = Api.client.eitherRequest {
    method = HttpMethod.parse("PATCH")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tasks",
      createSerializedPathSegment(value = taskId, explode = false, json = Api.json),
    )
    setBody(body)
    decorator()
  }

  /**
   * `POST /spaces/{spaceSlug}/tasks/{taskId}/relations`
   *
   * @param spaceSlug Slug of the space.
   * @param taskId Task ID of the source task.
   * @return The request has succeeded and a new resource has been created as a result.
   */
  public suspend fun spaceTaskRelationsCreate(
    spaceSlug: String,
    taskId: String,
    body: TaskRelationCreate,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<TaskRelation>> = Api.client.eitherRequest {
    method = HttpMethod.parse("POST")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tasks",
      createSerializedPathSegment(value = taskId, explode = false, json = Api.json),
      "relations",
    )
    setBody(body)
    decorator()
  }

  /**
   * `DELETE /spaces/{spaceSlug}/tasks/{taskId}/relations/{kind}/{relatedTaskId}`
   *
   * @param spaceSlug Slug of the space.
   * @param taskId Task ID of the source task.
   * @param kind Relation kind.
   * @param relatedTaskId Task ID of the related task.
   * @return There is no content to send for this request, but the headers may be useful. 
   */
  public suspend fun spaceTaskRelationsDelete(
    spaceSlug: String,
    taskId: String,
    kind: TaskRelationKind,
    relatedTaskId: String,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<Unit>> = Api.client.eitherRequest {
    method = HttpMethod.parse("DELETE")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "spaces",
      createSerializedPathSegment(value = spaceSlug, explode = false, json = Api.json),
      "tasks",
      createSerializedPathSegment(value = taskId, explode = false, json = Api.json),
      "relations",
      createSerializedPathSegment(value = kind, explode = false, json = Api.json),
      createSerializedPathSegment(value = relatedTaskId, explode = false, json = Api.json),
    )
    decorator()
  }

  /**
   * `GET /tasks/search`
   *
   * @param q Search query.
   * @param spaceSlug Optional space slug to restrict results to a single space.
   * @param excludeTaskId Optional task ID to exclude from results.
   * @param limit Maximum number of items to return (1–100).
   * @return The request has succeeded.
   */
  public suspend fun tasksSearch(
    q: String,
    spaceSlug: String? = null,
    excludeTaskId: String? = null,
    limit: Int? = null,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<TaskSearchResultList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "tasks",
      "search",
    )
    appendSerializedQueryParameter(name = "q", value = q, explode = false, json = Api.json)
    appendSerializedQueryParameter(name = "spaceSlug", value = spaceSlug, explode = false, json = Api.json)
    appendSerializedQueryParameter(name = "excludeTaskId", value = excludeTaskId, explode = false, json = Api.json)
    appendSerializedQueryParameter(name = "limit", value = limit, explode = false, json = Api.json)
    decorator()
  }
}

public object UsersApi {
  /**
   * `GET /users`
   *
   * @return The request has succeeded.
   */
  public suspend fun usersList(decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<UserList>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "users",
    )
    decorator()
  }

  /**
   * `POST /users`
   *
   * @return The request has succeeded and a new resource has been created as a result.
   */
  public suspend fun usersCreate(body: UserCreate, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<User>> = Api.client.eitherRequest {
    method = HttpMethod.parse("POST")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "users",
    )
    setBody(body)
    decorator()
  }

  /**
   * `GET /users/me`
   *
   * @return The request has succeeded.
   */
  public suspend fun usersMe(decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<User>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "users",
      "me",
    )
    decorator()
  }

  /**
   * `GET /users/{userId}`
   *
   * @param userId User ID.
   * @return The request has succeeded.
   */
  public suspend fun usersGet(userId: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<User>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "users",
      createSerializedPathSegment(value = userId, explode = false, json = Api.json),
    )
    decorator()
  }

  /**
   * `DELETE /users/{userId}`
   *
   * @param userId User ID.
   * @return There is no content to send for this request, but the headers may be useful. 
   */
  public suspend fun usersDelete(userId: String, decorator: HttpRequestBuilder.() -> Unit = {}): Either<CallException, HttpCallResponse<Unit>> = Api.client.eitherRequest {
    method = HttpMethod.parse("DELETE")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "users",
      createSerializedPathSegment(value = userId, explode = false, json = Api.json),
    )
    decorator()
  }

  /**
   * `PATCH /users/{userId}`
   *
   * @param userId User ID.
   * @return The request has succeeded.
   */
  public suspend fun usersUpdate(
    userId: String,
    body: UserUpdate,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<User>> = Api.client.eitherRequest {
    method = HttpMethod.parse("PATCH")
    contentType(ContentType.Application.Json)
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "users",
      createSerializedPathSegment(value = userId, explode = false, json = Api.json),
    )
    setBody(body)
    decorator()
  }

  /**
   * `GET /users/{userId}/tasks`
   *
   * @param userId User ID.
   * @param cursor Pagination cursor from a previous response.
   * @param limit Maximum number of items to return (1–100).
   * @return The request has succeeded.
   */
  public suspend fun userTasksList(
    userId: String,
    cursor: String? = null,
    limit: Int? = null,
    decorator: HttpRequestBuilder.() -> Unit = {},
  ): Either<CallException, HttpCallResponse<TaskPage>> = Api.client.eitherRequest {
    method = HttpMethod.parse("GET")
    authKeys(
      Auth.BearerAuth.ID,
    )
    url.appendPathSegments(
      "users",
      createSerializedPathSegment(value = userId, explode = false, json = Api.json),
      "tasks",
    )
    appendSerializedQueryParameter(name = "cursor", value = cursor, explode = false, json = Api.json)
    appendSerializedQueryParameter(name = "limit", value = limit, explode = false, json = Api.json)
    decorator()
  }
}
