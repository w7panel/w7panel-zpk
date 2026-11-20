package entity

func (library DocumentLibrary) Reference() string {
	return library.Category + "/" + library.Identifie
}
