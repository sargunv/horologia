import java.io.ByteArrayOutputStream
import org.gradle.api.DefaultTask
import org.gradle.api.file.RegularFileProperty
import org.gradle.api.tasks.CacheableTask
import org.gradle.api.tasks.InputFile
import org.gradle.api.tasks.OutputFile
import org.gradle.api.tasks.PathSensitive
import org.gradle.api.tasks.PathSensitivity
import org.gradle.api.tasks.TaskAction
import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.plugin.mpp.apple.XCFramework

plugins {
  alias(libs.plugins.android.kmp.library)
  alias(libs.plugins.kotlin.multiplatform)
  alias(libs.plugins.kotlin.serialization)
  alias(libs.plugins.skie)
}

@CacheableTask
abstract class PrepareHorologiaOpenApiTask : DefaultTask() {
  @get:InputFile
  @get:PathSensitive(PathSensitivity.RELATIVE)
  abstract val inputSpec: RegularFileProperty

  @get:OutputFile abstract val outputSpec: RegularFileProperty

  @TaskAction
  fun inlineRefs() {
    val inputText = inputSpec.get().asFile.readText()
    val componentParameters = parseComponentParameters(inputText)

    val withInlinedParameters =
      inputText.replace(Regex("""(?m)^(\s*)- \${'$'}ref: '#/components/parameters/([^']+)'$""")) {
        match ->
        val indent = match.groupValues[1]
        val parameterName = match.groupValues[2]
        val parameterLines =
          componentParameters[parameterName]
            ?: error(
              "OpenAPI component parameter $parameterName not found in ${inputSpec.get().asFile}"
            )

        parameterLines
          .mapIndexed { index, line -> if (index == 0) "${indent}- $line" else "$indent  $line" }
          .joinToString("\n")
      }
    val withNormalizedNullableAnyOf = normalizeSimpleNullableAnyOf(withInlinedParameters)
    val resolvedText = withNormalizedNullableAnyOf.replace("scheme: Bearer", "scheme: bearer")

    val outputFile = outputSpec.get().asFile
    outputFile.parentFile.mkdirs()
    outputFile.writeText(resolvedText)
  }

  private fun parseComponentParameters(spec: String): Map<String, List<String>> {
    val componentParameters = linkedMapOf<String, MutableList<String>>()
    var inParametersSection = false
    var currentParameter: String? = null

    spec.lineSequence().forEach { line ->
      when {
        !inParametersSection && line == "  parameters:" -> {
          inParametersSection = true
        }

        inParametersSection && line == "  schemas:" -> {
          return componentParameters
        }

        inParametersSection &&
          line.startsWith("    ") &&
          !line.startsWith("      ") &&
          line.endsWith(":") -> {
          currentParameter = line.trim().removeSuffix(":")
          componentParameters[currentParameter!!] = mutableListOf()
        }

        inParametersSection && currentParameter != null && line.startsWith("      ") -> {
          componentParameters.getValue(currentParameter!!).add(line.removePrefix("      "))
        }
      }
    }

    return componentParameters
  }

  private fun normalizeSimpleNullableAnyOf(spec: String): String {
    val lines = spec.lines()
    val rewritten = mutableListOf<String>()
    var index = 0

    while (index < lines.size) {
      val line = lines[index]
      if (line.trim() != "anyOf:") {
        rewritten += line
        index += 1
        continue
      }

      val indent = line.indexOfFirst { !it.isWhitespace() }.coerceAtLeast(0)
      val itemPrefix = " ".repeat(indent + 2) + "- "
      val nestedPrefix = " ".repeat(4)
      val firstAlternative = lines.getOrNull(index + 1)

      if (firstAlternative == null || !firstAlternative.startsWith(itemPrefix)) {
        rewritten += line
        index += 1
        continue
      }

      var secondAlternativeIndex = index + 2
      while (
        secondAlternativeIndex < lines.size &&
          !(lines[secondAlternativeIndex].startsWith(itemPrefix) &&
            lines[secondAlternativeIndex].countLeadingWhitespace() == indent + 2)
      ) {
        if (lines[secondAlternativeIndex].countLeadingWhitespace() <= indent) {
          break
        }
        secondAlternativeIndex += 1
      }

      val secondAlternative = lines.getOrNull(secondAlternativeIndex)
      if (secondAlternative == null) {
        rewritten += line
        index += 1
        continue
      }

      val blockEnd = findIndentedBlockEnd(lines, secondAlternativeIndex, indent)
      val hasThirdAlternative =
        (secondAlternativeIndex + 1 until blockEnd).any {
          lines[it].startsWith(itemPrefix) && lines[it].countLeadingWhitespace() == indent + 2
        }

      if (secondAlternative.trim() != "- type: 'null'" || hasThirdAlternative) {
        rewritten += line
        index += 1
        continue
      }

      rewritten += " ".repeat(indent) + firstAlternative.removePrefix(itemPrefix)
      for (nestedIndex in index + 2 until secondAlternativeIndex) {
        rewritten += lines[nestedIndex].removePrefix(nestedPrefix)
      }
      rewritten += " ".repeat(indent) + "nullable: true"
      index = blockEnd
    }

    return rewritten.joinToString("\n")
  }

  private fun findIndentedBlockEnd(lines: List<String>, startIndex: Int, parentIndent: Int): Int {
    var index = startIndex + 1
    while (index < lines.size && lines[index].countLeadingWhitespace() > parentIndent) {
      index += 1
    }
    return index
  }

  private fun String.countLeadingWhitespace(): Int {
    val index = indexOfFirst { !it.isWhitespace() }
    return if (index == -1) length else index
  }
}

val openApiSpec = rootProject.layout.projectDirectory.file("../api/gen/openapi.yaml")
val resolvedOpenApiSpec =
  layout.buildDirectory.file("generated/openapi/horologia-openapi-resolved.yaml")
val generatedSourceDir = layout.buildDirectory.dir("generated/openapi-kmp/src/commonMain/kotlin")

val openapiKmpGenCli by configurations.creating

dependencies { openapiKmpGenCli(libs.openapi.kmpgen.cli) }

kotlin {
  compilerOptions { freeCompilerArgs.add("-Xexpect-actual-classes") }

  androidLibrary {
    namespace = "dev.horologia.mobile.core"
    compileSdk = libs.versions.android.compileSdk.get().toInt()
    minSdk = libs.versions.android.minSdk.get().toInt()
  }

  jvm("desktop") { compilerOptions { jvmTarget = JvmTarget.JVM_17 } }

  val xcf = XCFramework("HorologiaCore")
  listOf(iosArm64(), iosSimulatorArm64(), iosX64()).forEach { target ->
    target.binaries.framework {
      baseName = "HorologiaCore"
      isStatic = true
      xcf.add(this)
      export(libs.androidx.lifecycle.viewmodel)
    }
  }

  sourceSets {
    commonMain {
      kotlin.srcDir(generatedSourceDir)
      dependencies {
        implementation(libs.androidx.compose.runtime.annotation)
        implementation(libs.kotlinx.coroutines.core)
        implementation(libs.kotlinx.serialization.json)
        implementation(libs.ktor.client.content.negotiation)
        implementation(libs.ktor.client.core)
        implementation(libs.ktor.serialization.kotlinx.json)
        implementation(libs.openapi.kmpgen.companion)
        api(libs.androidx.lifecycle.viewmodel)
      }
    }
    commonTest.dependencies {
      implementation(kotlin("test"))
      implementation(libs.kotlinx.coroutines.test)
      implementation(libs.ktor.client.mock)
    }
    androidMain.dependencies {
      implementation(libs.androidx.security.crypto)
      implementation(libs.ktor.client.okhttp)
    }
    val desktopMain by getting {
      dependencies {
        implementation(libs.kotlinx.coroutines.swing)
        implementation(libs.ktor.client.okhttp)
      }
    }
    iosMain.dependencies { implementation(libs.ktor.client.darwin) }
  }
}

val prepareHorologiaOpenApi by
  tasks.registering(PrepareHorologiaOpenApiTask::class) {
    group = "code generation"
    description =
      "Rewrite Horologia's emitted OpenAPI into the subset openapi-kmp-gen handles today"

    inputSpec.set(openApiSpec)
    outputSpec.set(resolvedOpenApiSpec)
  }

val generateRawHorologiaApi by
  tasks.registering(JavaExec::class) {
    var capturedStdout: ByteArrayOutputStream? = null
    var capturedStderr: ByteArrayOutputStream? = null

    group = "code generation"
    description = "Generate the committed Horologia KMP client from the emitted OpenAPI spec"

    dependsOn(prepareHorologiaOpenApi)
    inputs.file(resolvedOpenApiSpec)
    outputs.dir(generatedSourceDir)

    classpath = openapiKmpGenCli
    mainClass.set("com.kroegerama.openapi.kmp.gen.cli.CommandLineKt")
    isIgnoreExitValue = true

    doFirst {
      capturedStdout = ByteArrayOutputStream()
      capturedStderr = ByteArrayOutputStream()
      standardOutput = capturedStdout!!
      errorOutput = capturedStderr!!
    }

    args(
      "generate",
      "-p",
      "dev.horologia.mobile.generated",
      "-o",
      generatedSourceDir.get().asFile.path,
      "-s",
      "-a",
      resolvedOpenApiSpec.get().asFile.path,
    )

    doLast {
      val stdoutText = capturedStdout?.toString(Charsets.UTF_8) ?: ""
      stdoutText
        .lineSequence()
        .filter { line -> line.startsWith("selected options:") || line.startsWith("writing ") }
        .forEach(::println)

      val stderrText = capturedStderr?.toString(Charsets.UTF_8) ?: ""
      if (stderrText.isNotBlank()) {
        System.err.print(stderrText)
      }

      executionResult.get().assertNormalExitValue()
    }
  }

tasks.register("generateHorologiaApi") {
  group = "code generation"
  description = "Generate the Horologia KMP client from the emitted OpenAPI spec"

  dependsOn(generateRawHorologiaApi)
}

tasks
  .matching { task ->
    task.name.startsWith("compile") ||
      task.name.startsWith("sources") ||
      task.name == "assemble" ||
      task.name == "build"
  }
  .configureEach { dependsOn("generateHorologiaApi") }
