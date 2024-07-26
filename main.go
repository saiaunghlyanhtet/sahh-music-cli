package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

//go:embed assets/nancy.mp3
var defaultMusic embed.FS

func playMusic(wg *sync.WaitGroup, stopChan chan struct{}, musicPath string) {
	defer wg.Done()

	var musicFilePath string

	if musicPath != "" {
		musicFilePath = musicPath
	} else {
		tmpFile, err := os.CreateTemp("", "nancy.mp3")
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

		musicFilePath = tmpFile.Name()
	}

	// Play the file using mpg123
	cmd := exec.Command("mpg123", "-q", "--loop", "-1", musicFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Println("Failed to start mpg123:", err)
		return
	}

	go func() {
		<-stopChan
		cmd.Process.Kill()
	}()

	cmd.Wait()
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
	musicPath := flag.String("music", "", "Path to custom MP3 file")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: sahh [--music <path_to_mp3>] <command> [args...]")
		os.Exit(1)
	}

	command := flag.Args()[0]
	commandArgs := flag.Args()[1:]

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		close(stopChan)
	}()

	wg.Add(2)
	go playMusic(&wg, stopChan, *musicPath)
	go runCommand(command, commandArgs, &wg, stopChan)

	wg.Wait()
}
