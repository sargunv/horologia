package dev.horologia.mobile.compose.nav

import kotlinx.serialization.Serializable

/**
 * Type-safe navigation routes for the Compose app. Each destination is a `@Serializable` class so
 * nav-compose can encode it into the back-stack's saved state; args-carrying destinations will
 * appear here as `data class` entries when future screens need them.
 */
@Serializable data object LoginRoute

@Serializable data object ProfileRoute

@Serializable data object SpacesRoute
