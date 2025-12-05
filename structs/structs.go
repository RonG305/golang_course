package main
import "fmt"

func main() {
		
	// Structs let you create your own types (like classes without methods).
	// Structs
	println("************************Struct of person ***************")
	type Person struct {
		FirstName string
		LastName  string
		email	 string
		Password   string
		Age       int
		City      string
		State    string
	}

	person := Person {
		FirstName: "John",
		LastName:  "Doe",
		email:     "john.doe@example.com",
		Password:  "password123",
		Age:       30,
		City:      "New York",
		State:     "NY",
	}

	fmt.Printf("Person: %+v\n", person)
	person.Age = 34
	fmt.Println(person.FirstName, person.LastName, person.email, person.Password, person.Age, person.City, person.State)
}