package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"time"
)

func downloadFile(url string, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		fmt.Printf("File exists, skipping download: %s\n", dest)
		return nil
	}

	fmt.Printf("Downloading: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func getRouteViewsLatestURL(baseURL string) (string, error) {
	now := time.Now().UTC()
	path := fmt.Sprintf("%04d.%02d/RIBS/", now.Year(), now.Month())
	fullURL := baseURL + path

	resp, err := http.Get(fullURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	re := regexp.MustCompile(`rib\.\d{8}\.\d{4}\.bz2`)
	matches := re.FindAllString(string(body), -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no rib files found at %s", fullURL)
	}

	sort.Strings(matches)
	return fullURL + matches[len(matches)-1], nil
}
