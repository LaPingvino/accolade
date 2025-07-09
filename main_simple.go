package main

import (
	"fmt"
	"log"
	"os"
)

const (
	AppID = "org.codeberg.lapingvino.Accolade"
)

func main() {
	fmt.Println("Accolade - Fountain Screenplay Editor")
	fmt.Println("=====================================")
	
	// Check for command line arguments
	if len(os.Args) > 1 {
		for i, arg := range os.Args[1:] {
			fmt.Printf("File %d: %s\n", i+1, arg)
		}
	} else {
		fmt.Println("No files specified")
	}
	
	// Check for lexington availability
	if executableExists("lexington") {
		fmt.Println("✓ Lexington converter available")
	} else {
		fmt.Println("✗ Lexington converter not found")
	}
	
	fmt.Println("\nThis is a minimal stub. Full GTK4 UI coming soon...")
	
	log.Println("Accolade stub started successfully")
}

// Helper function to check if executable exists
func executableExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}