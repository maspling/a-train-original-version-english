package main

import (
	"a1patch/graphics"
	"a1patch/pe"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	gameDir, outDir, exePath := setupEnv()

	if err := pe.Patch(exePath, outDir); err != nil {
		log.Fatalf("Failed to patch EXE: %v\n", err)
	}

	if err := graphics.PatchDDT(gameDir, outDir); err != nil {
		log.Fatalf("Failed to patch DDT: %v\n", err)
	}

	if err := graphics.PackEmsg(gameDir, outDir); err != nil {
		log.Fatalf("Failed to patch Emsg: %v\n", err)
	}
	log.SetPrefix("")
	fmt.Println("\nTranslation complete! Check translated_files for the output.")
}

func setupEnv() (string, string, string) {
	gameDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}
	fmt.Printf("Working directory: %s\n", gameDir)

	exePath := filepath.Join(gameDir, "A1Win.exe")
	fmt.Printf("Working directory: %s\n", exePath)
	if _, err := os.Stat(exePath); err != nil {
		log.Fatalf("Cannot find A1Win.exe in %s, make sure this is run from the game directory", gameDir)
	}

	fmt.Printf("Starting Patcher!")
	if err := os.Mkdir("translated_files", 0o755); err != nil && !os.IsExist(err) {
		log.Fatalf("Failed to create folder translated_files: %v", err)
	}

	outDir := filepath.Join(gameDir, "translated_files")
	return gameDir, outDir, exePath
}
