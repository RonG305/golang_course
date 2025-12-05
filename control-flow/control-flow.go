package main
import "fmt"

func main() {
//  If statements 
	println("************************If Statements ***************")
	age := 67
	if age  < 18 {
		fmt.Println("You are a minor.")
	} else if age >= 18 && age < 65 {
		fmt.Println("You are an adult.")
	} else {
		fmt.Println("You are a senior.")
	}


	// Swicth statements
	println("************************Switch Statements ***************")
	day := "Tuesday"
	switch day {
	case "Monday":
		fmt.Println("It's", day, "the start of the week.")
	case "Wednesday":
		fmt.Println("It's", day, "we're halfway through the week.")
	case "Friday":
		fmt.Println("It's", day, "the weekend is near!")
	default:
		fmt.Println("It's", day, "just another day.")
	}

	// Switch with fallthrough
	// Fallthrough allows the execution to continue to the next case
	println("************************Switch with fallthrough ***************")
	grade := "A"
	switch grade {
	case "A":
		fmt.Println("Excellent!")
		fallthrough
	case "B":
		fmt.Println("Very Good!")
		fallthrough
	case "C":
		fmt.Println("Good!")
	case "D":
		fmt.Println("You passed.")
	case "F":
		fmt.Println("Better luck next time.")
	default:
		fmt.Println("Invalid grade.")
	}

	// Type Switch
	println("************************Type Switch ***************")
	var value interface{} = 42
	switch v := value.(type) {
	case int:
		fmt.Println("Integer:", v)
	case string:
		fmt.Println("String:", v)
	case bool:
		fmt.Println("Boolean:", v)
	default:
		fmt.Println("Unknown type")
	}
}


