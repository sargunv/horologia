@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package dev.horologia.mobile.api.models

import kotlinx.serialization.EncodeDefault
import kotlinx.serialization.KSerializer
import kotlinx.serialization.Serializable
import kotlinx.serialization.descriptors.nullable
import kotlinx.serialization.encoding.Decoder
import kotlinx.serialization.encoding.Encoder

/**
 * A PATCH member that distinguishes omission from an explicit JSON null.
 *
 * [Absent] is the default and is never encoded. [Null] clears a nullable server field, while
 * [Value] replaces it.
 */
@Serializable(with = PatchFieldSerializer::class)
sealed interface PatchField<out T> {
    data object Absent : PatchField<Nothing>

    data object Null : PatchField<Nothing>

    data class Value<T>(val value: T) : PatchField<T>
}

class PatchFieldSerializer<T>(
    private val valueSerializer: KSerializer<T>,
) : KSerializer<PatchField<T>> {
    override val descriptor = valueSerializer.descriptor.nullable

    override fun serialize(encoder: Encoder, value: PatchField<T>) {
        when (value) {
            PatchField.Absent -> error("PatchField.Absent must be omitted by its containing wire model")
            PatchField.Null -> encoder.encodeNull()
            is PatchField.Value -> encoder.encodeSerializableValue(valueSerializer, value.value)
        }
    }

    override fun deserialize(decoder: Decoder): PatchField<T> =
        if (decoder.decodeNotNullMark()) {
            PatchField.Value(decoder.decodeSerializableValue(valueSerializer))
        } else {
            decoder.decodeNull()
            PatchField.Null
        }
}

@Serializable
data class TaskUpdateWire(
    val title: String? = null,
    val description: String? = null,
    val status: String? = null,
    @EncodeDefault(EncodeDefault.Mode.NEVER)
    val effort: PatchField<String> = PatchField.Absent,
    @EncodeDefault(EncodeDefault.Mode.NEVER)
    val priority: PatchField<String> = PatchField.Absent,
    val recurrenceType: TaskRecurrenceType? = null,
    @EncodeDefault(EncodeDefault.Mode.NEVER)
    val recurrenceRule: PatchField<String> = PatchField.Absent,
    val assigneeIds: List<String>? = null,
    val rotationPool: List<String>? = null,
    val tags: List<String>? = null,
    @EncodeDefault(EncodeDefault.Mode.NEVER)
    val due: PatchField<TaskDue> = PatchField.Absent,
    @EncodeDefault(EncodeDefault.Mode.NEVER)
    val overdueActionRule: PatchField<TaskOverdueActionRule> = PatchField.Absent,
)

@Serializable
data class RecipeUpdateWire(
    val name: String? = null,
    val description: String? = null,
    @EncodeDefault(EncodeDefault.Mode.NEVER)
    val yield: PatchField<RecipeYield> = PatchField.Absent,
    @EncodeDefault(EncodeDefault.Mode.NEVER)
    val prepMinutes: PatchField<Int> = PatchField.Absent,
    @EncodeDefault(EncodeDefault.Mode.NEVER)
    val cookMinutes: PatchField<Int> = PatchField.Absent,
    val tags: List<String>? = null,
    val ingredientSections: List<RecipeIngredientSectionInput>? = null,
    val instructionSections: List<RecipeInstructionSectionInput>? = null,
)
