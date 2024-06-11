package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

//go:embed assets/nancy.mp3
var defaultMusic embed.FS

func playMusic(wg *sync.WaitGroup, stopChan chan struct{}) {
	defer wg.Done()

	tmpFile, err := os.CreateTemp("", "nancy*.mp3")
	if err != nil {
		fmt.Println("Failed to create temp file:", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	// Copy the embedded file to the temp file
	file, err := defaultMusic.Open("assets/nancy.mp3")
	if err != nil {
		fmt.Println("Failed to open embedded music file:", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		fmt.Println("Failed to copy embedded music file:", err)
		return
	}

	// Close the temp file to ensure it's written
	if err := tmpFile.Close(); err != nil {
		fmt.Println("Failed to close temp file:", err)
		return
	}

	// Play the file using mpg123
	cmd := exec.Command("mpg123", "-q", "--loop", "-1", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Println("Failed to start mpg123:", err)
		return
	}

	<-stopChan
	cmd.Process.Kill()
}

func runCommand(command string, args []string, wg *sync.WaitGroup, stopChan chan struct{}) {
	defer wg.Done()

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("Command failed:", err)
	}

	// Signal the music to stop
	close(stopChan)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: sahh-music-cli <command> [args...]")
		os.Exit(1)
	}

	command := os.Args[1]
	commandArgs := os.Args[2:]

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	wg.Add(2)
	go playMusic(&wg, stopChan)
	go runCommand(command, commandArgs, &wg, stopChan)

	wg.Wait()
}
