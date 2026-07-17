export interface KeyedItem {
  key: string;
}

export interface KeyedCollection<T extends KeyedItem> {
  key: string;
  title: string;
  items: T[];
}

export function moveKeyed<T extends KeyedItem>(
  values: T[],
  activeKey: string,
  targetKey: string,
): T[] {
  const sourceIndex = values.findIndex((value) => value.key === activeKey);
  const targetIndex = values.findIndex((value) => value.key === targetKey);
  if (sourceIndex < 0 || targetIndex < 0 || sourceIndex === targetIndex) return values;

  const next = [...values];
  const [moved] = next.splice(sourceIndex, 1);
  if (!moved) return values;
  next.splice(targetIndex, 0, moved);
  return next;
}

export function moveKeyedCollectionItem<T extends KeyedItem>(
  collections: KeyedCollection<T>[],
  activeKey: string,
  sourceCollectionKey: string,
  targetCollectionKey: string,
  targetKey?: string,
): KeyedCollection<T>[] {
  const sourceIndex = collections.findIndex((collection) => collection.key === sourceCollectionKey);
  const targetIndex = collections.findIndex((collection) => collection.key === targetCollectionKey);
  if (sourceIndex < 0 || targetIndex < 0) return collections;

  const sourceItemIndex = collections[sourceIndex]!.items.findIndex(
    (item) => item.key === activeKey,
  );
  if (sourceItemIndex < 0) return collections;

  const targetItemIndex = targetKey
    ? collections[targetIndex]!.items.findIndex((item) => item.key === targetKey)
    : collections[targetIndex]!.items.length;
  if (targetItemIndex < 0) return collections;

  if (sourceIndex === targetIndex) {
    const currentItems = collections[sourceIndex]!.items;
    const items = targetKey
      ? moveKeyed(currentItems, activeKey, targetKey)
      : sourceItemIndex === currentItems.length - 1
        ? currentItems
        : [
            ...currentItems.filter((item) => item.key !== activeKey),
            currentItems[sourceItemIndex]!,
          ];
    if (items === collections[sourceIndex]!.items) return collections;
    return collections.map((collection, index) =>
      index === sourceIndex ? { ...collection, items } : collection,
    );
  }

  const moved = collections[sourceIndex]!.items[sourceItemIndex]!;
  return collections
    .map((collection, index) => {
      if (index === sourceIndex) {
        return {
          ...collection,
          items: collection.items.filter((item) => item.key !== activeKey),
        };
      }
      if (index === targetIndex) {
        const items = [...collection.items];
        items.splice(targetItemIndex, 0, moved);
        return { ...collection, items };
      }
      return collection;
    })
    .filter((collection) => collection.title.trim() || collection.items.length > 0);
}
