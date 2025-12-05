package main

import "fmt"

func main() {
	// Slices
	colorsSlice := []string{"Red", "Green", "Blue", "Yellow"}
	colorsSlice = append(colorsSlice, "purple")
	colorsSlice = append(colorsSlice[:0], colorsSlice[1:]...)
	println("************************Slice of colors ***************")
	for _, color := range colorsSlice {
		fmt.Println(color)
	}

	// Arrays
	colorsArray := [9]string{ "Red", "Green", "Blue", "Yellow" }
	tempSlice := append(colorsArray[:0], colorsArray[1:]...)
	copy(colorsArray[:], tempSlice)
	println("************************Array of colors ***************")
	for _, color := range colorsArray {
		fmt.Println(color)
	}

	// Maps
	colorsMap := map[string]string{
		"R": "Red",
		"G": "Green",
		"B": "Blue",
		"Y": "Yellow",
	}
	delete(colorsMap, "R")
	println("************************Map of colors ***************")
	for key, color:= range colorsMap {
		fmt.Printf("%s: %s\n", key, color)
	}


	// Another Map
	println("************************Map of countries ***************")
	countriesMap := map[string] string {
		"US": "United States",
		"CA": "Canada",
		"MX": "Mexico",
		"FR": "France",
		"DE": "Germany",
		"JP": "Japan",
		"CN": "China",
		"IN": "India",
		"BR": "Brazil",
		"ZA": "South Africa",
	}
	delete(countriesMap, "ZA")

	findCountry, found := countriesMap["MX"]
	if found {
		fmt.Println("Country for code 'MX':", findCountry)
	} else {
		fmt.Println("Country code 'MX' does not exist.")
	}

	fmt.Println(countriesMap["US"])
	country, exists := countriesMap["KE"]
	if exists {
		fmt.Println("Country for code 'KE':", country)
	} else {
		fmt.Println("Country code 'FR' does not exist.")
	}

	for code, country:= range countriesMap {
		fmt.Println(code, ":", country)
	}



}


