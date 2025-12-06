package functions

import "fmt"

// simple function
func sayHello() {
    fmt.Println("Hello!")
}

// function with parameter
func greet(name string) {
    fmt.Println("Hello", name)
}

// function with one return value
func add(a int, b int) int {
    return a + b
}

// function with multiple returns
func divide(a, b float64) (float64, string) {
    if b == 0 {
        return 0, "cannot divide by zero"
    }
    return a / b, ""
}

func RunFunctions() {
    sayHello()
    greet("Ronald")

    sum := add(4, 5)
    fmt.Println("Sum:", sum)

    result, err := divide(10, 0)
    fmt.Println("Divide:", result, "Error:", err)
}
