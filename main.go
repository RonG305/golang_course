package main
import "fmt"

func main() {
	var name string = "My Todo List App"
	var todos []string = []string{
		"Learn Go",
		"Build a web app",
		"Write tests",
	}
	fmt.Println(name)
	fmt.Println("***********************************************************")
	fmt.Println(todos)
	for i, todos:=range todos {
		fmt.Println(i, todos)
	}

	fmt.Println("*****************************************************")
	fmt.Println("Variable Declaration Examples:")
	fmt.Println("*****************************************************")
	// Go variables
	var firstName string = "Ronald"
	var lastName string = "Mutia"
	var age int = 25
	var isMarried bool = false

	fmt.Println("First Name:", firstName)
	fmt.Println("Last Name:", lastName)
	fmt.Println("Age:", age)
	fmt.Println("Is Married:", isMarried)
}
