package controlflow
import "fmt"

func ControlFlow() {
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

	grade := 90
	if grade < 55 {
		fmt.Println("Failed")
	} else if grade >= 55 && grade < 70 {
		fmt.Println("Good Job")
	} else if grade >= 70 && grade < 86 {
		fmt.Println("Great work")
	} else {
		fmt.Println("Excellent!")
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


