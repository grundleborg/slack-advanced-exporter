package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var fetchProfilePicturesCmd = &cobra.Command{
	Use:   "fetch-profile-pictures",
	Short: "Download profile pictures and link them to user profiles",
	RunE:  fetchProfilePics,
}

func init() {
	fetchProfilePicturesCmd.PersistentFlags()
}

func fetchProfilePics(cmd *cobra.Command, args []string) error {
	// Open the input archive.
	r, err := zip.OpenReader(inputArchive)
	if err != nil {
		fmt.Printf("Could not open input archive for reading: %s\n", inputArchive)
		os.Exit(1)
	}
	defer r.Close()

	// Open the output archive.
	f, err := os.Create(outputArchive)
	if err != nil {
		fmt.Printf("Could not open the output archive for writing: %s\n\n%s", outputArchive, err)
		os.Exit(1)
	}
	defer f.Close()

	// Create a zip writer on the output archive.
	w := zip.NewWriter(f)

	// Run through all the files in the input archive.
	for _, file := range r.File {
		verbosePrintln(fmt.Sprintf("Processing file: %s\n", file.Name))

		// Open the file from the input archive.
		inReader, err := file.Open()
		if err != nil {
			fmt.Printf("Failed to open file in input archive: %s\n\n%s", file.Name, err)
			os.Exit(1)
		}

		if file.Name == "users.json" {
			err = downloadPictures(inReader, w)
			if err != nil {
				fmt.Printf("Failed to fetch users' emails.\n\n%s", err)
				os.Exit(1)
			}
		} else {
			// Copy, because CreateHeader modifies it.
			header := file.FileHeader
			outFile, err := w.CreateHeader(&header)
			if err != nil {
				fmt.Printf("Failed to create file in output archive: %s\n\n%s", file.Name, err)
				os.Exit(1)
			}

			_, err = io.Copy(outFile, inReader)
			if err != nil {
				fmt.Printf("Failed to copy file to output archive: %s\n\n%s", file.Name, err)
				os.Exit(1)
			}
		}
	}

	// Close the output zip writer.
	err = w.Close()
	if err != nil {
		fmt.Printf("Failed to close the output archive.\n\n%s", err)
	}

	return nil
}

func downloadPictures(input io.Reader, w *zip.Writer) error {
	verbosePrintln("Found users.json file.")

	// We want to preserve all existing fields in JSON.
	// By using interface{} (instead of struct), we can avoid describing all
	// the fields (new ones might be added by Slack devs in the future!) at the cost of
	// slight inconvenience of type assertions and working with maps.
	var data []map[string]interface{}
	err := json.NewDecoder(input).Decode(&data)
	if err != nil {
		return err
	}

	verbosePrintln("Updating users.json contents with fetched pictures.")

	for _, user := range data {
		// These 'ok's only check for type assertion success.
		// Map access would return untyped nil,
		// which is fine, as untyped nil would fail both these type assertions.
		name, _ := user["name"].(string)

		if userid, ok := user["id"].(string); ok {
			if profile, ok := user["profile"].(map[string]interface{}); ok {
				if image_url, ok := profile["image_original"].(string); ok &&
					strings.HasPrefix(image_url, "https://avatars.slack-edge.com/") &&
					(strings.HasSuffix(image_url, ".jpg") || strings.HasSuffix(image_url, ".png")) {
					parts := strings.Split(image_url, ".")
					extension := "." + parts[len(parts)-1]

					req, err := http.NewRequest("GET", image_url, nil)
					if err != nil {
						return fmt.Errorf("Got error %s when building the request", err.Error())
					}
					log.Printf("Downloading profile picture for %q", name)

					response, err := httpClient.Do(req)
					if err != nil {
						log.Printf("Failed to download profile picture for user %q from %s", userid, image_url)
					}
					defer response.Body.Close()

					picFileName := "profile_pictures/" + userid + extension
					profile["image_path"] = picFileName

					// Save the file to the output zip file.
					outFile, err := w.Create(picFileName)
					if err != nil {
						return fmt.Errorf("Failed to write profile picture to zip file for %q from %s", userid, image_url)
					}
					_, err = io.Copy(outFile, response.Body)
					if err != nil {
						log.Print("++++++ Failed to write the downloaded file to the output archive: " + image_url + "\n\n" + err.Error() + "\n")
					}

				} else {
					log.Printf("Skipping %q, no suitable profile picture found", userid)
				}

			} else {
				log.Printf("User %q doesn't have 'profile' in JSON file (unexpected error!)", userid)
			}
		} else {
			log.Print("Some user array entry doesn't have id, skipping")
		}
	}

	file, err := w.Create("users.json")
	if err != nil {
		return fmt.Errorf("Failed to write users.json back to archive")
	}
	enc := json.NewEncoder(file)
	// The same indent level as export zip uses.
	enc.SetIndent("", "    ")
	return enc.Encode(&data)
}
