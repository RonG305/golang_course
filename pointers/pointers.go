package pointers
import "fmt"

func RunPointers() {
// Pointers
// Pointers hold the memory address of a value.
	println("************************Pointers ***************")
	var x int = 42
	var p *int = &x // p holds the address of x

	fmt.Println("Value of x:", x)		   // 42
	fmt.Println("Address of x:", &x)        // memory address of x
	fmt.Println("Value of p (address of x):", p) // same as address of x
	fmt.Println("Value at address p:", *p)  // get value at address p
}   