import org.gradle.api.tasks.Delete
import org.gradle.api.tasks.Sync
import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.openapitools.generator.gradle.plugin.tasks.GenerateTask

plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.android.kmp.library)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.openapi.generator)
}

val kmpOpenApiSpec = rootProject.layout.projectDirectory.file("../../api/gen/kmp/openapi.yaml")
val openApiStagingOutput = layout.buildDirectory.dir("generated/openapi-staging")
val generatedCommonMainKotlin = layout.buildDirectory.dir("generated/openapi-kotlin")
val generatedCommonMainKotlinSources =
    objects.fileCollection().from(generatedCommonMainKotlin)

kotlin {
    android {
        namespace = "dev.horologia.mobile.api"
        compileSdk = libs.versions.compileSdk.get().toInt()
        minSdk = libs.versions.minSdk.get().toInt()

        compilerOptions {
            jvmTarget = JvmTarget.JVM_17
        }
    }
    iosArm64()
    iosSimulatorArm64()
    jvmToolchain(libs.versions.java.get().toInt())

    sourceSets {
        commonMain {
            kotlin.srcDir(generatedCommonMainKotlinSources)
            dependencies {
                implementation(libs.ktor.client.core)
                implementation(libs.ktor.client.content.negotiation)
                implementation(libs.ktor.serialization.kotlinx.json)
                implementation(libs.kotlinx.serialization.core)
                implementation(libs.kotlinx.serialization.json)
                implementation(libs.kotlinx.datetime)
            }
        }
        androidMain {
            dependencies {
                implementation(libs.ktor.client.okhttp)
            }
        }
        iosMain {
            dependencies {
                implementation(libs.ktor.client.darwin)
            }
        }
    }
}


val cleanKmpApi = tasks.register<Delete>("cleanOpenApiOutput") {
    delete(openApiStagingOutput)
}

val generateKmpApi = tasks.named<GenerateTask>("openApiGenerate") {
    dependsOn(cleanKmpApi)
    group = "code generation"
    description = "Generate the Kotlin Multiplatform API client from the derived OpenAPI 3.0 document"

    generatorName.set("kotlin")
    library.set("multiplatform")
    inputSpec.set(kmpOpenApiSpec.asFile.absolutePath)
    outputDir.set(openApiStagingOutput.get().asFile.absolutePath)
    configFile.set(layout.projectDirectory.file("openapi-generator-config.yaml").asFile.absolutePath)
    schemaMappings.set(
        mapOf(
            "RecipeUpdate" to "RecipeUpdateWire",
            "TaskUpdate" to "TaskUpdateWire",
        ),
    )
    importMappings.set(
        mapOf(
            "RecipeUpdateWire" to "dev.horologia.mobile.api.models.RecipeUpdateWire",
            "TaskUpdateWire" to "dev.horologia.mobile.api.models.TaskUpdateWire",
        ),
    )
    globalProperties.set(
        mapOf(
            "apiDocs" to "false",
            "apiTests" to "false",
            "modelDocs" to "false",
            "modelTests" to "false",
        ),
    )

    inputs.file(kmpOpenApiSpec)
    inputs.file(layout.projectDirectory.file("openapi-generator-config.yaml"))
    outputs.dir(openApiStagingOutput)

}

val syncOpenApiGeneratedSources = tasks.register<Sync>("syncOpenApiGeneratedSources") {
    dependsOn(generateKmpApi)
    group = "code generation"
    description = "Copy generated common Kotlin API sources into the compilation source directory"
    from(openApiStagingOutput.map { it.dir("src/commonMain/kotlin") })
    into(generatedCommonMainKotlin)
}

generatedCommonMainKotlinSources.builtBy(syncOpenApiGeneratedSources)

tasks.withType<org.jetbrains.kotlin.gradle.tasks.KotlinCompilationTask<*>>().configureEach {
    dependsOn(syncOpenApiGeneratedSources)
}

