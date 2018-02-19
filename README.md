Slack Advanced Exporter
=======================

The Slack Advanced Exporter is a tool for supplementing official data exports from Slack with the
other bits and pieces that these don't include.

Current Features
----------------

This tool can supplement an official Slack team export by adding the following to it:

* Users' e-mail addresses
* File Uploads

Installation
------------

Release binaries can be downloaded from release tags on Github
[here](https://github.com/grundleborg/slack-advanced-exporter/releases).

Usage
-----

First, run a full export of your Slack team, and have the produced zip file handy.

Due to `archive/zip` limitations, these actions cannot modify archive in place.
It's preferable to fetch e-mails first to avoid copying large attachments around.

### Add users' e-mails to your export.
To fetch all users' e-mail addresses and add them to the archive,
user this command:

    ./slack-advanced-exporter --input-archive your-slack-team-export.zip --output-archive export-with-emails.zip fetch-emails --api-token xoxp-123...

You'll need to obtain an API token [here](https://api.slack.com/docs/oauth-test-tokens).

### Add all the File Attachments to your export.

To fetch all the file attachments referenced in your Slack team export andd add them to the archive,
use this command:

    ./slack-advanced-exporter --input-archive your-slack-team-export.zip --output-archive export-with-attachments.zip fetch-attachments

Problems
--------

If you encounter a bug in this tool, please file a bug report on its Github issue tracker.

Development Plans
-----------------

I'm interested in adding features to this application to export anything else that it is possible to
export but which isn't included in the existing Slack export archives. PRs and Issues welcome.

Contributing
------------

Pull requests are welcome. Please ensure that you comply with the requirements below:

* Only data that is not included in the official team export feature should be exported by this
  tool - if it's already in the official export, there's no need for it here.
* Ensure you only request what is needed, to avoid unnecessary load being placed on servers.


