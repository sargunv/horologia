// Values must be non-null; null is reserved as the internal "none queued/in-flight" sentinel.
export class QueuedAutosave<T extends NonNullable<unknown>> {
  #persistedValue: T;
  #localValue: T;
  #inFlightValue: T | null = null;
  #queuedValue: T | null = null;
  #acceptedExternalValue: T;
  #onChange: (() => void) | undefined;
  readonly #save: (value: T) => Promise<T | undefined>;
  readonly #equals: (a: T, b: T) => boolean;

  constructor(
    initialValue: T,
    save: (value: T) => Promise<T | undefined>,
    equals: (a: T, b: T) => boolean = Object.is,
  ) {
    this.#save = save;
    this.#equals = equals;
    this.#persistedValue = initialValue;
    this.#localValue = initialValue;
    this.#acceptedExternalValue = initialValue;
  }

  get localValue() {
    return this.#localValue;
  }

  get persistedValue() {
    return this.#persistedValue;
  }

  setChangeListener(onChange: (() => void) | undefined) {
    this.#onChange = onChange;
  }

  setLocalValue(value: T) {
    if (this.#equals(value, this.#localValue)) return;
    this.#localValue = value;
    this.#notify();
  }

  receiveExternalValue(value: T) {
    if (this.#equals(value, this.#acceptedExternalValue)) return;

    this.#acceptedExternalValue = value;
    if (this.isBlocked()) {
      this.#notify();
      return;
    }

    this.#persistedValue = value;
    this.#localValue = value;
    this.#notify();
  }

  canSyncExternalValue(value: T) {
    return (
      !this.isBlocked() &&
      this.#equals(value, this.#acceptedExternalValue) &&
      this.#equals(value, this.#persistedValue) &&
      this.#equals(value, this.#localValue)
    );
  }

  isBlocked() {
    return (
      this.#inFlightValue !== null ||
      this.#queuedValue !== null ||
      !this.#equals(this.#localValue, this.#persistedValue)
    );
  }

  requestSave(value = this.#localValue) {
    this.#localValue = value;

    if (this.#equals(value, this.#persistedValue)) {
      this.#queuedValue = null;
      this.#notify();
      return;
    }

    if (this.#inFlightValue === null) {
      this.#startSave(value);
      return;
    }

    if (this.#equals(value, this.#inFlightValue)) {
      this.#queuedValue = null;
    } else {
      this.#queuedValue = value;
    }
    this.#notify();
  }

  #startSave(value: T) {
    if (this.#equals(value, this.#persistedValue)) {
      this.#notify();
      return;
    }

    this.#inFlightValue = value;
    if (this.#queuedValue !== null && this.#equals(this.#queuedValue, value)) {
      this.#queuedValue = null;
    }
    this.#notify();

    void this.#save(value).then(
      (savedValue) => {
        this.#persistedValue = savedValue ?? value;
        this.#inFlightValue = null;
        this.#queuedValue = null;

        const nextValue = this.#localValue;

        if (!this.#equals(nextValue, this.#persistedValue)) {
          this.#startSave(nextValue);
          return;
        }

        this.#notify();
      },
      () => {
        this.#inFlightValue = null;
        this.#queuedValue = null;
        this.#notify();
      },
    );
  }

  #notify() {
    this.#onChange?.();
  }
}
