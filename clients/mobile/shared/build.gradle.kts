import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.android.kmp.library)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.sqldelight)
}

kotlin {
    jvmToolchain(libs.versions.java.get().toInt())

    android {
        namespace = "dev.horologia.mobile.shared"
        compileSdk = libs.versions.compileSdk.get().toInt()
        minSdk = libs.versions.minSdk.get().toInt()
        withHostTestBuilder {}.configure {}

        compilerOptions {
            jvmTarget = JvmTarget.JVM_17
        }
    }

    listOf(
        iosArm64(),
        iosSimulatorArm64(),
    ).forEach { target ->
        target.binaries.framework {
            baseName = "HorologiaShared"
            isStatic = true
        }
    }

    sourceSets {
        commonMain.dependencies {
            implementation(project(":api-generated"))
            implementation(libs.androidx.lifecycle.viewmodel)
            implementation(libs.ktor.client.core)
            implementation(libs.ktor.client.content.negotiation)
            implementation(libs.ktor.serialization.kotlinx.json)
            implementation(libs.kotlinx.coroutines.core)
            implementation(libs.kotlinx.datetime)
            implementation(libs.kotlinx.serialization.core)
            implementation(libs.kotlinx.serialization.json)
            implementation(libs.sqldelight.runtime)
        }

        androidMain.dependencies {
            implementation(libs.androidx.browser)
            implementation(libs.ktor.client.okhttp)
            implementation(libs.sqldelight.android.driver)
        }

        iosMain.dependencies {
            implementation(libs.ktor.client.darwin)
            implementation(libs.sqldelight.native.driver)
        }

        commonTest.dependencies {
            implementation(kotlin("test"))
        }
    }
}

sqldelight {
    databases {
        create("HorologiaDatabase") {
            packageName.set("dev.horologia.mobile.persistence")
            srcDirs.setFrom("src/commonMain/sqldelight")
        }
    }
}
