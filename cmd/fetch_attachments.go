package cmd

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	attachmentsApiToken string
	forceRedownload     bool
)

var fetchAttachmentsCmd = &cobra.Command{
	Use:   "fetch-attachments",
	Short: "Fetch all file attachments and add them to the output archive",
	RunE:  fetchAttachments,
}

func init() {
	fetchAttachmentsCmd.PersistentFlags().StringVar(&attachmentsApiToken, "api-token", "", "Slack API token. Can be obtained here: https://api.slack.com/docs/oauth-test-tokens")
	fetchAttachmentsCmd.PersistentFlags().BoolVar(&forceRedownload, "force", false, "Force re-download all files, deleting any existing staging directory")
}

// getStagingDir returns the staging directory path for the given output archive
func getStagingDir(outputArchive string) string {
	return outputArchive + ".staging"
}

// getDownloadedFileIDsFromStaging scans the staging directory and returns a map of already downloaded file IDs
func getDownloadedFileIDsFromStaging(stagingDir string) (map[string]struct{}, error) {
	downloadedIDs := make(map[string]struct{})

	uploadsDir := filepath.Join(stagingDir, "__uploads")
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		return downloadedIDs, nil
	}

	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		return downloadedIDs, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// The directory name is the file ID
			downloadedIDs[entry.Name()] = struct{}{}
		}
	}

	return downloadedIDs, nil
}

// downloadFileToStaging downloads a single file to the staging directory
func downloadFileToStaging(stagingDir string, file *SlackFile, token string) error {
	// Figure out the download URL to use
	var downloadUrl string
	if len(file.UrlPrivateDownload) > 0 {
		downloadUrl = file.UrlPrivateDownload
	} else {
		downloadUrl = file.UrlPrivate
	}

	// Build the output file path in staging directory
	outputDir := filepath.Join(stagingDir, "__uploads", file.Id)
	outputPath := filepath.Join(outputDir, file.Name)

	// Create the directory structure
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer outFile.Close()

	// Fetch the file
	client := &http.Client{}
	req, err := http.NewRequest("GET", downloadUrl, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", downloadUrl, err)
	}
	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", downloadUrl, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: HTTP %d", downloadUrl, response.StatusCode)
	}

	// Save the file
	_, err = io.Copy(outFile, response.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	return nil
}

// createFinalZip creates the final ZIP archive from the input archive and staging directory
func createFinalZip(inputArchive, outputArchive, stagingDir string) error {
	// Open the input archive
	r, err := zip.OpenReader(inputArchive)
	if err != nil {
		return fmt.Errorf("could not open input archive: %w", err)
	}
	defer r.Close()

	// Create the output archive
	f, err := os.Create(outputArchive)
	if err != nil {
		return fmt.Errorf("could not create output archive: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// Copy all files from input archive to output
	for _, file := range r.File {
		verbosePrintln(fmt.Sprintf("Copying from input archive: %s", file.Name))

		inReader, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in input archive %s: %w", file.Name, err)
		}

		outFile, err := w.Create(file.Name)
		if err != nil {
			inReader.Close()
			return fmt.Errorf("failed to create file in output archive %s: %w", file.Name, err)
		}

		_, err = io.Copy(outFile, inReader)
		inReader.Close()
		if err != nil {
			return fmt.Errorf("failed to copy file to output archive %s: %w", file.Name, err)
		}
	}

	// Add all files from staging directory
	uploadsDir := filepath.Join(stagingDir, "__uploads")
	if _, err := os.Stat(uploadsDir); err == nil {
		err = filepath.Walk(uploadsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Calculate the relative path for the ZIP entry
			relPath, err := filepath.Rel(stagingDir, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path for %s: %w", path, err)
			}
			// Normalize to forward slashes for ZIP
			zipPath := filepath.ToSlash(relPath)

			verbosePrintln(fmt.Sprintf("Adding to output archive: %s", zipPath))

			// Open the source file
			srcFile, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open staged file %s: %w", path, err)
			}
			defer srcFile.Close()

			// Create the file in the ZIP
			outFile, err := w.Create(zipPath)
			if err != nil {
				return fmt.Errorf("failed to create file in output archive %s: %w", zipPath, err)
			}

			// Copy the content
			_, err = io.Copy(outFile, srcFile)
			if err != nil {
				return fmt.Errorf("failed to write file to output archive %s: %w", zipPath, err)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk staging directory: %w", err)
		}
	}

	return nil
}

func fetchAttachments(cmd *cobra.Command, args []string) error {
	stagingDir := getStagingDir(outputArchive)

	// Handle --force: clean slate
	if forceRedownload {
		verbosePrintln("Force flag set, removing existing staging directory...")
		os.RemoveAll(stagingDir)
	}

	// Create staging directory if needed
	uploadsDir := filepath.Join(stagingDir, "__uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	verbosePrintln(fmt.Sprintf("Staging directory: %s", stagingDir))

	// Get already-downloaded file IDs from staging
	downloadedFileIDs, err := getDownloadedFileIDsFromStaging(stagingDir)
	if err != nil {
		log.Printf("Warning: could not scan staging directory: %v", err)
		downloadedFileIDs = make(map[string]struct{})
	}
	verbosePrintln(fmt.Sprintf("Found %d files already in staging", len(downloadedFileIDs)))

	// Open input archive
	r, err := zip.OpenReader(inputArchive)
	if err != nil {
		return fmt.Errorf("could not open input archive for reading: %s: %w", inputArchive, err)
	}
	defer r.Close()

	// Counters for summary
	var downloadedCount, skippedCount int

	// Process channel files and download to staging
	for _, file := range r.File {
		verbosePrintln(fmt.Sprintf("Processing file: %s", file.Name))

		// Check if this is a channel file
		if isChannelFile(file.Name) {
			downloaded, skipped, err := processChannelFile(file, stagingDir, attachmentsApiToken, downloadedFileIDs)
			if err != nil {
				return fmt.Errorf("error processing %s: %w", file.Name, err)
			}
			downloadedCount += downloaded
			skippedCount += skipped
		}
	}

	r.Close()

	// All downloads complete - create final ZIP
	fmt.Println("All downloads complete. Creating final archive...")
	err = createFinalZip(inputArchive, outputArchive, stagingDir)
	if err != nil {
		return fmt.Errorf("failed to create final archive: %w", err)
	}

	// Clean up staging directory
	verbosePrintln("Cleaning up staging directory...")
	os.RemoveAll(stagingDir)

	fmt.Printf("Completed: Downloaded %d new files, skipped %d already present.\n", downloadedCount, skippedCount)
	return nil
}

func processChannelFile(file *zip.File, stagingDir string, token string, downloadedFileIDs map[string]struct{}) (downloaded int, skipped int, err error) {
	verbosePrintln("This is a 'channels' file. Examining its contents for attachments.")

	// Open the file from the archive
	inReader, err := file.Open()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer inReader.Close()

	// Read the file content
	inBuf, err := io.ReadAll(inReader)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the JSON
	var posts []SlackPost
	if err := json.Unmarshal(inBuf, &posts); err != nil {
		return 0, 0, errors.New("couldn't parse the JSON file: " + file.Name + "\n\n" + err.Error())
	}

	// Loop through all the posts
	for _, post := range posts {
		// Support for legacy file_share posts
		if post.Subtype == "file_share" {
			if post.File == nil {
				log.Print("++++++ file_share post has no File property: " + post.Ts + "\n")
				continue
			}
			post.Files = []*SlackFile{post.File}
		}

		// If the post doesn't contain any files, move on
		if post.Files == nil {
			continue
		}

		// Loop through all the files
		for _, slackFile := range post.Files {
			// Check there's an Id, Name and either UrlPrivateDownload or UrlPrivate property
			if len(slackFile.Id) < 1 || len(slackFile.Name) < 1 || !(len(slackFile.UrlPrivate) > 0 || len(slackFile.UrlPrivateDownload) > 0) {
				log.Print("++++++ file_share post has missing properties on its File object: " + post.Ts + "\n")
				continue
			}

			// Check if already downloaded
			if _, exists := downloadedFileIDs[slackFile.Id]; exists {
				verbosePrintln(fmt.Sprintf("Skipping already downloaded file: %s (%s)", slackFile.Id, slackFile.Name))
				skipped++
				continue
			}

			// Download the file to staging
			verbosePrintln(fmt.Sprintf("Downloading file %s (%s) to staging...", slackFile.Id, slackFile.Name))
			err := downloadFileToStaging(stagingDir, slackFile, token)
			if err != nil {
				log.Printf("++++++ Failed to download file %s: %v", slackFile.Id, err)
				continue
			}

			// Mark as downloaded for subsequent iterations
			downloadedFileIDs[slackFile.Id] = struct{}{}
			downloaded++
			fmt.Printf("Downloaded: %s (%s)\n", slackFile.Id, slackFile.Name)
		}
	}

	return downloaded, skipped, nil
}
