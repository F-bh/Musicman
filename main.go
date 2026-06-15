package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Println("yt-dlp failed to download:", line,  err)
		}
	}
}
