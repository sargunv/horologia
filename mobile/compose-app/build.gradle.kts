import org.jetbrains.compose.desktop.application.dsl.TargetFormat
import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
  alias(libs.plugins.kotlin.multiplatform)
  alias(libs.plugins.android.application)
  alias(libs.plugins.kotlin.compose)
  alias(libs.plugins.compose)
}

android {
  namespace = "dev.horologia.mobile.compose"
  compileSdk = libs.versions.android.compileSdk.get().toInt()
  buildToolsVersion = "36.0.0"

  defaultConfig {
    applicationId = "dev.horologia.mobile.compose"
    minSdk = libs.versions.android.minSdk.get().toInt()
    targetSdk = libs.versions.android.targetSdk.get().toInt()
    versionCode = 1
    versionName = "0.0.0"
  }

  buildFeatures { compose = true }

  compileOptions {
    isCoreLibraryDesugaringEnabled = true
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
  }

  packaging { resources { excludes += "/META-INF/{AL2.0,LGPL2.1}" } }
}

kotlin {
  androidTarget { compilerOptions { jvmTarget = JvmTarget.JVM_17 } }
  jvm("desktop") { compilerOptions { jvmTarget = JvmTarget.JVM_17 } }

  sourceSets {
    val commonMain by getting {
      dependencies {
        implementation(projects.core)
        implementation(libs.compose.foundation)
        implementation(libs.compose.material3)
        implementation(libs.compose.runtime)
        implementation(libs.compose.ui)
      }
    }

    val androidMain by getting {
      dependencies {
        implementation(libs.androidx.activity.compose)
        implementation(libs.androidx.core.ktx)
      }
    }

    val desktopMain by getting { dependencies { implementation(compose.desktop.currentOs) } }
  }
}

compose.desktop {
  application {
    mainClass = "dev.horologia.mobile.compose.MainKt"

    nativeDistributions {
      targetFormats(TargetFormat.Dmg, TargetFormat.Msi, TargetFormat.Deb)
      packageName = "Horologia"
      packageVersion = "1.0.0"
    }
  }
}

dependencies { coreLibraryDesugaring(libs.android.desugar.jdk.libs) }
