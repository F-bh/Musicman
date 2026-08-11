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

		// Request yt-dlp to print a single unambiguous line per video (title|id)
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
			"--print", "%(title)s|%(id)s",
			"--newline",
			url,
		}

		if _, err := os.Stat(netRcLocation); err == nil {
			args = append(args, "--netrc", "--netrc-location", netRcLocation)
		}

		cmd := exec.Command(exePath, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

		// Combine stderr into stdout so a single scanner can read both streams.
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Println("Failed to get stdout pipe:", err)
			continue
		}
		// merge stderr into stdout
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			errorMsg := fmt.Sprintf("yt-dlp failed to start for: %s - %v", line, err)
			fmt.Println(errorMsg)
			logErrorToFile(errorMsg)
			continue
		}

		// Single scanner reading combined stdout+stderr in the main goroutine.
		var currentItem string
		scannerOut := bufio.NewScanner(stdoutPipe)
		for scannerOut.Scan() {
			text := scannerOut.Text()
			fmt.Println(text)

			// parse our explicit per-video print line: "Title|ID"
			if strings.Contains(text, "|") {
				parts := strings.SplitN(text, "|", 2)
				if len(parts) == 2 {
					currentItem = strings.TrimSpace(parts[0])
				}
			}

			// detect error lines and log them with the current item
			if strings.Contains(text, "ERROR:") {
				item := currentItem
				if item == "" {
					item = "unknown"
				}
				logErrorToFile(fmt.Sprintf("yt-dlp error for playlist '%s' item '%s': %s", playListTitle, item, text))
			}
		}
		if err := scannerOut.Err(); err != nil {
			fmt.Println("scanner error:", err)
		}

		if err := cmd.Wait(); err != nil {
			item := currentItem
			if item == "" {
				item = "unknown"
			}
			errorMsg := fmt.Sprintf("yt-dlp finished with error for: %s (last item: %s) - %v", line, item, err)
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
