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
}
