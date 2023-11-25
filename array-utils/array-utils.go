package arrayUtils

func Every[T any](arr []T, validator func(T) bool) bool {
	for _, el := range arr {
		if !validator(el) {
			return false
		}
	}

	return true
}

func Some[T any](arr []T, validator func(T) bool) bool {
	for _, el := range arr {
		if validator(el) {
			return true
		}
	}

	return false
}

func Includes[T comparable](arr []T, item T) bool {
	return Some(arr, func(i T) bool {
		return i == item
	})
}

func FilterEmptyStrings(arr []string) []string {
	output := make([]string, 0, len(arr))

	for _, item := range arr {
		if item != "" {
			output = append(output, item)
		}
	}

	return output
}
