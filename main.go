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
	for i, todos := range todos {
		fmt.Printf("%d. %s\n", i+1, todos)
	}
}