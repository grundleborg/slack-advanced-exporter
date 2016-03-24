#!/usr/bin/env python3
# 
# Slack Upload Exporter.
#
# This script parses a Slack team data export and downloads all the uploaded
# files that are linked to in messages in the data export, allowing you to
# import them elsewhere when you need them.
#
# Please see README.md for how to use this script.
#
# Copyright (c) 2016 George Goldberg
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#
#

import json
import os
import requests

################################ CONFIGURATION VARS ################################################

# The path to the directory where your slack export (unzipped) can be found.
SLACK_EXPORT_PATH='export/'

# The path to the directory where the exported files should be put.
FILES_PATH='files/'

# The path to the file containing your Slack signin cookie data.
COOKIE_FILE='COOKIE'

#################################### THE SCRIPT ####################################################

def identify_channels(path):
    """ Build a list of channels based on the contents of the export. """

    channels = []
    for x in os.walk(path):
        # Skip the export directory itself.
        if x[0] == path:
            continue

        # Add to list.
        channels.append(x[0])
        print(" - Found Channel: {}".format(x[0]))

    return channels


def get_file_list(path):
    """ Build a list of export files within a given channel. """
    files = [os.path.join(path, f) for f in os.listdir(path) if os.path.isfile(os.path.join(path, f))]
    return files


def process_file(path, file_path, cookie):
    """ Parse the file for a given day/channel combination and identify all the uploads. """
    with open(path, "r", encoding="utf-8") as f:
        j = json.load(f)

        for i in j:
            if i.get("subtype") == "file_share":
                if "url_private_download" in i["file"]:
                    url = i["file"]["url_private_download"]
                elif "url_private" in i["file"]:
                    url = i["file"]["url_private"]
                else:
                    raise Exception("Couldn't figure out what file to download.")

                id = i["file"]["id"]
                name = i["file"]["name"]

                if not check_if_retrieved(file_path, id, name):
                    print("   + Retrieving File: {}".format(name))
                    retrieve_upload(id, name, url, file_path, cookie)
                else:
                    print("   + Already retrieved file: {}".format(name))


def check_if_retrieved(file_path, id, name):
    """ Check if a file has already been retrieved before. """
    if os.path.isfile(os.path.join(file_path, id, name)):
        return True
    return False


def retrieve_upload(id, name, url, file_path, cookie):
    """ Retrievve an uploaded file from Slack's servers and save it locally. """

    r = requests.get(url, headers={ "Cookie": cookie})
    if r.status_code != 200:
        print("ERROR {}".format(r.status_code))
        raise Exception("Received non-200 status code: {}".format(r.status_code))

    # Make the folder and write out the file.
    if not os.path.exists(os.path.join(file_path, id)):
        os.makedirs(os.path.join(file_path, id))
    
    with open(os.path.join(file_path, id, name), "wb") as f:
        f.write(r.content)


def main(export_path, files_path, cookie):
    """ The script main function. """

    with open(cookie) as f:
        c = f.read()
    
    print("Building list of channels.")
    channels = identify_channels(export_path)
    print()

    print("Processing each channel.")
    for channel in channels:
        files = get_file_list(channel)

        for f in files:
            print(" - Processing file: {}".format(f))
            process_file(f, files_path, c)
    print()


if __name__ == "__main__":
    print("Launching script.")
    main(SLACK_EXPORT_PATH, FILES_PATH, COOKIE_FILE)
    print("Done.")


