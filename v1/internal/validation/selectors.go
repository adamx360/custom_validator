package validation

type SliceSelector struct {
	CollectionPath string
	KeyField       string
}

func (v *Validator) RegisterSliceSelector(collectionPath, keyField string) {
	v.sliceSelectors[normalizePath(collectionPath)] = SliceSelector{
		CollectionPath: normalizePath(collectionPath),
		KeyField:       keyField,
	}
}
