package libs

func InArr(element string, array []string) string {
	for _, value := range array {
		if len(element) < len(value) {
			if element == value[:len(element)] {
				return value
			}
		} else {
			if element == value {
				return value
			}
		}
	}
	return ""
}
