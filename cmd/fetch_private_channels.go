package cmd

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	privateChannelsApiToken string
)

var fetchPrivateChannelsCmd = &cobra.Command{
	Use:   "fetch-private-channels",
	Short: "Fetch all private channels accessible to the user",
	RunE:  fetchPrivateChannels,
}

func init() {
	fetchPrivateChannelsCmd.PersistentFlags().StringVar(&privateChannelsApiToken, "api-token", "", "Slack API token. Can be obtained here: https://api.slack.com/docs/oauth-test-tokens")
	fetchPrivateChannelsCmd.MarkPersistentFlagRequired("api-token")
}

func fetchPrivateChannels(cmd *cobra.Command, args []string) error {
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

	groupsFound := false
	// Run through all the files in the input archive.
	for _, file := range r.File {
		verbosePrintln(fmt.Sprintf("Processing file: %s\n", file.Name))

		// Open the file from the input archive.
		inReader, err := file.Open()
		if err != nil {
			fmt.Printf("Failed to open file in input archive: %s\n\n%s", file.Name, err)
			os.Exit(1)
		}

		// Copy, because CreateHeader modifies it.
		header := file.FileHeader

		outFile, err := w.CreateHeader(&header)
		if err != nil {
			fmt.Printf("Failed to create file in output archive: %s\n\n%s", file.Name, err)
			os.Exit(1)
		}

		if file.Name == "groups.json" {
			groupsFound = true
			verbosePrintln("The file groups.json is already present in the dump, we don't fetch it again")
		}
		_, err = io.Copy(outFile, inReader)
		if err != nil {
			fmt.Printf("Failed to copy file to output archive: %s\n\n%s", file.Name, err)
			os.Exit(1)
		}
	}

	outFile, err := w.Create("groups.json")
	if err != nil {
		return err
	}
	if !groupsFound {
		err = createGroupsJson(outFile, privateChannelsApiToken, w)
		if err != nil {
			fmt.Printf("Failed to fetch private channels.\n\n%s\n", err)
			os.Exit(1)
		}
	}

	// Close the output zip writer.
	err = w.Close()
	if err != nil {
		fmt.Printf("Failed to close the output archive.\n\n%s", err)
	}

	return nil
}

func createGroupsJson(output io.Writer, slackApiToken string, w *zip.Writer) error {

	verbosePrintln("Creating groups.json by fetching private channels.")

	privateChannels, err := fetchPrivateChannelsList(slackApiToken)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(output)
	// The same indent level as export zip uses.
	enc.SetIndent("", "    ")
	var err2 = enc.Encode(&privateChannels)
	if err2 != nil {
		return err2
	}

	verbosePrintln("Fetching the contents of private channels")
	for _, channel := range privateChannels {
		var channelId = channel["id"].(string)
		var channelName = channel["name"].(string)
		verbosePrintln("Fetching the replies of private channel " + channelName)

		var messageFileName = channelName + "/messages.json"

		outFile, err := w.Create(messageFileName)
		if err != nil {
			return err
		}
		ts_ids, err := fetchChannelHistory(outFile, slackApiToken, channelId)
		if err != nil {
			return err
		}

		outFileReplies, err := w.Create(channelName + "/replies.json")
		if err != nil {
			return err
		}
		fetchChannelReplies(outFileReplies, slackApiToken, channelId, ts_ids)

		verbosePrintln("Done with replies of private channel " + channelName)

	}
	return nil
}

func fetchChannelHistory(output io.Writer, token string, channelId string) ([]string, error) {
	client := &http.Client{}
	res := make([]map[string]interface{}, 0)
	ts_ids := make([]string, 0)
	url := "https://slack.com/api/conversations.history"

	cursor := ""

	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("got error %s when building the request", err)
		}

		query := req.URL.Query()
		query.Add("limit", "200")
		query.Add("channel", channelId)
		if cursor != "" {
			query.Add("cursor", cursor)
		}
		req.URL.RawQuery = query.Encode()

		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Slack API returned HTTP code %d", resp.StatusCode)
		}

		var data struct {
			Ok               bool                     `json:"ok"`
			Messages         []map[string]interface{} `json:"messages"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			return nil, err
		}

		if !data.Ok {
			return nil, errors.New("unexpected lack of ok=true in Slack API response. Is access token correct?")
		}

		res = append(res, data.Messages...)
		for _, message := range data.Messages {
			reply_count, has_reply_count := message["reply_count"].(float64)
			if has_reply_count && reply_count > 0 {
				id, present := message["ts"].(string)
				if present {
					ts_ids = append(ts_ids, id)
				}
			}
		}

		cursor = data.ResponseMetadata.NextCursor
		verbosePrintln("Processed a batch of messages.")

		if cursor == "" {
			break // Exit the loop if there's no next cursor
		}
	}
	enc := json.NewEncoder(output)
	// The same indent level as export zip uses.
	enc.SetIndent("", "    ")
	return ts_ids, enc.Encode(&res)
}

func fetchChannelReplies(output io.Writer, token string, channelId string, tsIds []string) error {
	client := &http.Client{}
	res := make([]map[string]interface{}, 0)
	url := "https://slack.com/api/conversations.replies"

	cursor := ""

	for _, tsId := range tsIds {
		for {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("got error %s when building the request", err)
			}

			query := req.URL.Query()
			query.Add("limit", "200")
			query.Add("channel", channelId)
			query.Add("ts", tsId)
			if cursor != "" {
				query.Add("cursor", cursor)
			}
			req.URL.RawQuery = query.Encode()

			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("Slack API returned HTTP code %d", resp.StatusCode)
			}

			var data struct {
				Ok               bool                     `json:"ok"`
				Messages         []map[string]interface{} `json:"messages"`
				ResponseMetadata struct {
					NextCursor string `json:"next_cursor"`
				} `json:"response_metadata"`
			}
			err = json.NewDecoder(resp.Body).Decode(&data)
			if err != nil {
				return err
			}

			if !data.Ok {
				return errors.New("unexpected lack of ok=true in Slack API response. Is access token correct?")
			}

			res = append(res, data.Messages...)

			cursor = data.ResponseMetadata.NextCursor
			verbosePrintln("Processed a batch of replies.")

			if cursor == "" {
				break // Exit the loop if there's no next cursor
			}
		}
	}
	enc := json.NewEncoder(output)
	// The same indent level as export zip uses.
	enc.SetIndent("", "    ")
	return enc.Encode(&res)
}

func fetchPrivateChannelsList(token string) ([]map[string]interface{}, error) {
	verbosePrintln("Fetching private channels from Slack API")

	client := &http.Client{}
	res := make([]map[string]interface{}, 0)
	url := "https://slack.com/api/conversations.list"

	cursor := ""

	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("got error %s when building the request", err)
		}

		query := req.URL.Query()
		query.Add("limit", "1000")
		query.Add("types", "private_channel")
		if cursor != "" {
			query.Add("cursor", cursor)
		}
		req.URL.RawQuery = query.Encode()

		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Slack API returned HTTP code %d", resp.StatusCode)
		}

		var data struct {
			Ok               bool                     `json:"ok"`
			Channels         []map[string]interface{} `json:"channels"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			return nil, err
		}

		if !data.Ok {
			return nil, errors.New("unexpected lack of ok=true in Slack API response. Is access token correct?")
		}

		res = append(res, data.Channels...)

		cursor = data.ResponseMetadata.NextCursor
		verbosePrintln("Processed a batch of channels.")

		if cursor == "" {
			break // Exit the loop if there's no next cursor
		}
	}

	verbosePrintln("Fetched all private channels from Slack API.")
	return res, nil
}
