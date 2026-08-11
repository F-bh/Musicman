package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	exePath := "yt-dlp.exe"
	outputTemplate := "%(title)s.%(ext)s"
	playlistFile := filepath.Join("./", "playlists.txt")
	netRcLocation := filepath.Join("./", "config.netrc")

	file, err := os.Open(playlistFile)
	if err != nil {
		fmt.Println("Failed to open playlists.txt:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		temp := strings.SplitN(line, " ", 2)
		url := strings.TrimSpace(temp[0])
		playListTitle := strings.TrimSpace(temp[1])
		if url == "" || playListTitle == "" {
			println("Invalid line in playlists.txt", line)
			continue
		}

		// Create directory if it doesn't exist
		err := os.MkdirAll("./"+playListTitle, 0755)
		if err != nil {
			fmt.Println("Failed to create directory:", err)
			return
		}

		archive := fmt.Sprintf("./%v/archive.txt", playListTitle)

		args := []string{
			"-P", "./" + playListTitle,
			"-f", "ba/b",
			"-x", "--audio-format", "mp3",
			"-S", "acodec:mp3",
			"--embed-metadata",
			"--download-archive", archive,
			"--add-metadata",
			"--postprocessor-args", "ffmpeg:-metadata album= -metadata comment= -metadata album_artist=",
			"--cookies-from-browser", "firefox",
			"--embed-thumbnail",
			"-o", outputTemplate,
			"--no-abort-on-error",
			url,
		}
		
		if  _, err := os.Stat(netRcLocation); err == nil {
			args = append(args, "--netrc", "--netrc-location", netRcLocation)
		}

		cmd := exec.Command(exePath, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

		// Capture stdout and stderr so we can scan output line-by-line and log per-item errors
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Println("Failed to get stdout pipe:", err)
			continue
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			fmt.Println("Failed to get stderr pipe:", err)
			continue
		}

		if err := cmd.Start(); err != nil {
			errorMsg := fmt.Sprintf("yt-dlp failed to start for: %s - %v", line, err)
			fmt.Println(errorMsg)
			logErrorToFile(errorMsg)
			continue
		}

		// Scan stdout and stderr concurrently. Forward output to the program's stdout/stderr
		go func() {
			s := bufio.NewScanner(stdoutPipe)
			for s.Scan() {
				text := s.Text()
				fmt.Fprintln(os.Stdout, text)
				if strings.Contains(text, "ERROR:") {
					logErrorToFile(fmt.Sprintf("yt-dlp error for playlist '%s': %s", playListTitle, text))
				}
			}
			// ignore scanner error; if needed, could log it
		}()

		go func() {
			s := bufio.NewScanner(stderrPipe)
			for s.Scan() {
				text := s.Text()
				fmt.Fprintln(os.Stderr, text)
				if strings.Contains(text, "ERROR:") {
					logErrorToFile(fmt.Sprintf("yt-dlp error for playlist '%s': %s", playListTitle, text))
				}
			}
		}()

		if err := cmd.Wait(); err != nil {
			errorMsg := fmt.Sprintf("yt-dlp finished with error for: %s - %v", line, err)
			fmt.Println(errorMsg)
			// Still log this overall failure, but per-item ERROR: lines will have been logged already
			logErrorToFile(errorMsg)
		}
	}
}

func logErrorToFile(errorMsg string) {
	// Create log filename with current date
	logFileName := fmt.Sprintf("errors.log")
	logFilePath := filepath.Join("./", logFileName)

	// Open log file in append mode, create if it doesn't exist
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		return
	}
	defer logFile.Close()

	// Write error message with timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, errorMsg)
	if _, err := logFile.WriteString(logEntry); err != nil {
		fmt.Println("Failed to write to log file:", err)
	}
}
