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
		fmt.Println(i+1,".", todos)
	}

	fmt.Println("*****************************************************")
	fmt.Println("Variable Declaration Examples:")
	fmt.Println("*****************************************************")
	// Go variables
	var firstName string = "Ronald"
	var lastName string = "Mutia"
	var age int = 25
	var isMarried bool = false

	// short variable declaration
	city:= "Nairobi"
	country:= "Kenya"
	height:= 5.9
	isAdmin:= true

	fmt.Println("Is Admin:", isAdmin)
	fmt.Println("City:", city)
	fmt.Println("Country:", country)
	fmt.Println("Height:", height)


	fmt.Println("First Name:", firstName)
	fmt.Println("Last Name:", lastName)
	fmt.Println("Age:", age)
	fmt.Println("Is Married:", isMarried)

	fmt.Println("My name is", firstName + " " + lastName, "I am" + " ", age, "Years old.")

	// Multiple variable declaration
	company, role, experience := "Tech Corp", "Software Engineer", 3
	fmt.Println("Company:", company)
	fmt.Println("Role:", role)
	fmt.Println("Experience:", experience, "years")

	// Constant declaration
	const pi float64 = 3.14
	const appName string = "Todo List App"
	fmt.Println("App Name:", appName)
	fmt.Println("Value of Pi:", pi)
}
