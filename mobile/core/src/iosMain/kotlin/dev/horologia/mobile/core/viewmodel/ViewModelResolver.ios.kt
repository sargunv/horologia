package dev.horologia.mobile.core.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.ViewModelStore
import androidx.lifecycle.viewmodel.CreationExtras
import kotlin.experimental.ExperimentalObjCName
import kotlin.reflect.KClass
import kotlinx.cinterop.BetaInteropApi
import kotlinx.cinterop.ObjCClass
import kotlinx.cinterop.getOriginalKotlinClass

@OptIn(BetaInteropApi::class, ExperimentalObjCName::class)
@Throws(IllegalArgumentException::class)
fun ViewModelStore.resolveViewModel(
  modelClass: ObjCClass,
  factory: ViewModelProvider.Factory,
  key: String?,
  extras: CreationExtras? = null,
): ViewModel {
  @Suppress("UNCHECKED_CAST") val vmClass = getOriginalKotlinClass(modelClass) as? KClass<ViewModel>
  require(vmClass != null) { "The modelClass parameter must be a ViewModel type." }

  val provider = ViewModelProvider.create(this, factory, extras ?: CreationExtras.Empty)
  return key?.let { provider[key, vmClass] } ?: provider[vmClass]
}
