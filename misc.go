package main

//Find string in slice and return index
func findStringInSlice(slice []string, str string) int {
	for i, v := range slice {
		if v == str {
			return i
		}
	}
	return -1
}