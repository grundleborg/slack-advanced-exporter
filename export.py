#!/usr/bin/env python3
# 
# Slack Advanced Exporter.
#
# This tool provides a means of exporting data from Slack that the official
# export download doesn't export, inluding attachments, private groups and
# direct messages.
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

from bs4 import BeautifulSoup

import argparse
import getpass
import json
import os
import requests
import shutil
import sys
import urllib
import zipfile

VERSION = "0.1.0"

class SlackSession:
    def __init__(self):
        self.logged_in = False

    def from_cookie(self, cookie, team):
        # Set up the team URL and requests session.
        self.teamUrl = "https://{}/".format(team)
        self.session = requests.Session()

        # Parse the cookie.
        cookie_dict = dict(p.split('=') for p in cookie.split('; '))
        
        for key, value in cookie_dict.items():
            self.session.cookies.set(key, urllib.parse.unquote(value))

        # Check the cookie.
        t = self.session.get(self.teamUrl + "messages")
        t.raise_for_status()

        self.logged_in = True

    def log_in(self, team, email, password):
        # Set up the team URL and requests session.
        self.teamUrl = "https://{}/".format(team)
        self.session = requests.Session()

        # Get the signin form.
        r = self.session.get(self.teamUrl)
        r.raise_for_status()

        # Extract the crumb.
        soup = BeautifulSoup(r.text, "html.parser")
        crumb = soup.find("input", attrs={"name": "crumb"})["value"]

        # Assemble the sign-in form submission data.
        data = {
                "signin": 1,
                "redir": None,
                "email": email,
                "password": password,
                "remember": "on",
                "crumb": crumb,
        }

        # Sign in.
        s = self.session.post(self.teamUrl, data=data)
        s.raise_for_status()

        # Check the cookie.
        t = self.session.get("https://slack.com/checkcookie", params={"redir": self.teamUrl})
        t.raise_for_status()

        self.logged_in = True


class AttachmentsExporter:
    def __init__(self, session):
        self.session = session

    def set_files_path(self, export_file, attachments_path):
        self.export_file = export_file
        self.attachments_path = attachments_path

        # Extract the export zip file.
        if os.path.exists("export"):
            shutil.rmtree("export")
        with zipfile.ZipFile(export_file, "r") as zip_ref:
            zip_ref.extractall("export/")

        self.export_path = "export"

    def run(self):
        print("Building list of channels.")
        channels = self.identify_channels()
        print()

        print("Processing each channel.")
        for channel in channels:
            files = self.build_file_list(channel)

            for f in files:
                print(" - Processing file: {}".format(f))
                self.process_file(f)

    def identify_channels(self):
        """ Build a list of channels based on the contents of the export. """

        channels = []
        for x in os.walk(self.export_path):
            # Skip the export directory itself.
            if x[0] == self.export_path:
                continue

            # Add to list.
            channels.append(x[0])
            print(" - Found Channel: {}".format(x[0]))

        return channels

    def build_file_list(self, path):
        """ Build a list of export files within a given channel. """
        files = [os.path.join(path, f) for f in os.listdir(path) if os.path.isfile(os.path.join(path, f))]
        return files

    def process_file(self, path):
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

                    if not self.check_if_already_retrieved(id, name):
                        print("   + Retrieving File: {}".format(name))
                        self.retrieve_upload(id, name, url)
                    else:
                        print("   + Already retrieved file: {}".format(name))

    def check_if_already_retrieved(self, id, name):
        """ Check if a file has already been retrieved before. """
        if os.path.isfile(os.path.join(self.attachments_path, id, name)):
            return True
        return False

    def retrieve_upload(self, id, name, url):
        """ Retrievve an uploaded file from Slack's servers and save it locally. """

        r = self.session.session.get(url)
        if r.status_code != 200:
            print("ERROR {}".format(r.status_code))
            raise Exception("Received non-200 status code: {}".format(r.status_code))

        # Make the folder and write out the file.
        if not os.path.exists(os.path.join(self.attachments_path, id)):
            os.makedirs(os.path.join(self.attachments_path, id))
        
        with open(os.path.join(self.attachments_path, id, name), "wb") as f:
            f.write(r.content)


class Application:
    def __init__(self):
        self.session = SlackSession()

    def parse_args(self):
        parser = argparse.ArgumentParser()
        parser.add_argument("-i", "--interactive", help="Run the script interactively.", action="store_true")
        parser.add_argument("-c", "--cookie", help="Path to a file containing the Slack cookies.", type=str)
        parser.add_argument("-t", "--team", help="The Slack team URL in the form: team.slack.com", type=str)
        parser.add_argument("-e", "--export-file", help="The path to the Slack export file (in .zip format).", type=str)
        parser.add_argument("-o", "--output-file", help="The path to the zip file this script should produce containing the original Slack export contents and any additional data exported by this script.", type=str)
        parser.add_argument("--download-attachments", help="Download all attachments for the accompanying export file.", action="store_true")
        return parser.parse_args()

    def session_from_cookie(self, cookie, team):
        print("Checking cookie for session")
        with open(cookie, "r") as f:
            cookie_content = f.read().rstrip("\n").rstrip("\r")
            self.session.from_cookie(cookie_content, team)

    def interactive(self):
        # Get the Slack Team, Email and Password:
        print("")
        print("Slack Advanced Exporter v{}".format(VERSION))
        print("==============================")
        print("")
        print("This script provides a means of exporting data from Slack that the official export")
        print("download doesn't export, including attachments, private groups and direct messages.")
        print("")
        print("For usage instructions, please see the included README.md")
        print("")
        print("---------------------------------------------------------------------------------------")
        print("")
        
        if self.session.logged_in:
            print("Using session from cookie, so no need to log in.")
        else:
            print("Please enter your Slack team address, in the form teamname.slack.com")
            team = input("Team Name: ")

            print("")
            print("Please enter your Slack email address and password below to sign in.")

            email = input("Email Address: ")
            password = getpass.getpass()

            print("")
            print("Signing in to Slack...")
            print("")

            # Attempt to sign in to Slack.
            self.session.log_in(team, email, password)

            print("Signed in successfully.")

        while True:
            print("")
            print("---------------------------------------------------------------------------------------")
            print("")
            print("Please select from the following options by keying in the number then hitting return.")
            #print(" 1. Export my direct message history.")
            #print(" 2. Export my private groups.")
            print(" 3. Download attachments to go with an existing Slack export.")
            print(" 4. Save the original export and the data exported by this script to a file.")
            print(" 5. Exit Slack Advanced Exporter.")
            print("")
            choice = input("Your Choice: ")

            if choice == "1":
                print()
                print()
                print(" **** Sorry this is not yet implemented. **** ")
                print()

            elif choice == "2":
                print(" **** Sorry this is not yet implemented. **** ")

            elif choice == "3":
                print("")
                export_file = input("Path to Slack Export zip file [export.zip]: ")
                if export_file == "":
                    export_file = "export.zip"
                self.download_attachments(export_file)

            elif choice == "4":
                print("")
                output_file = input("Path to the new export zip file [output.zip]: ")
                if output_file == "":
                    output_file = "output.zip"
                self.save_output(output_file)

            elif choice == "5":
                print("")
                print("Exiting Slack Advanced Exporter.")
                break

            else:
                print("")
                print("Unrecognised Choice selected. Please try again.")

    def download_attachments(self, export_file):
        a = AttachmentsExporter(self.session)
        a.set_files_path(export_file, "./files")

        print("")
        print("Extracting attachments. This will take a *long* time...")
        a.run()

        print("Extracting Attachments Complete")

    def save_output(self, output_file):
        print("")
        print("Exporting original archive plus extra exported data to {}.".format(output_file))
        # Copy the files downloaded to the export directory.
        if os.path.exists("export/__uploads"):
            shutil.rmtree("export/__uploads")
        shutil.copytree("files", "export/__uploads")
        zipf = zipfile.ZipFile(output_file, 'w', zipfile.ZIP_DEFLATED)
        for root, dirs, files in os.walk("export"):
            for f in files:
                filePath = os.path.join(root, f)
                inZipPath = filePath.replace("export", "", 1).lstrip("\\/")
                zipf.write(filePath, inZipPath)
        zipf.close()

if __name__ == "__main__":
    app = Application()

    args = app.parse_args()

    if args.cookie is not None and args.team is not None:
        app.session_from_cookie(args.cookie, args.team)

    if args.interactive:
        app.interactive()
    else:
        if args.download_attachments and args.export_file is not None:
            app.download_attachments(args.export_file)
        else:
            print("Unrecognised command combinations. Please see included README.md for how to use")
            print("this script.")

        if args.output_file is not None:
            app.save_output(args.output_file)


