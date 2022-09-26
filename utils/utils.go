package utils


func FindIndex(slice []string, searchValue string) int {
	for index, el := range slice {
		if searchValue == el {
			return index
		}

	}
	return -1
}

func RemoveElementByIndex(slice []string, index int) []string  {
	sliceLen := len(slice)
	sliceLastIndex := sliceLen - 1

	if index != sliceLastIndex {
		slice[index] = slice[sliceLastIndex]
	}

	return slice[:sliceLastIndex]
}

type Set map[string]struct{}

// Adds an  to the set
func (s Set) Add(value string) {
	s[value] = struct{}{}
}

// Removes an  from the set
func (s Set) Remove(value string) {
	delete(s, value)
}

func (s Set) Has(value string) bool {
	_, ok := s[value]
	return ok
}