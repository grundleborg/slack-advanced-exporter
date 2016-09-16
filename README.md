Slack Advanced Exporter
=======================

The Slack Advanced Exporter is a tool for supplementing official data exports from Slack with the
other bits and pieces that these don't include.

If you want a local copy of *all* your Slack data for offline use or just for peace of mind, this is
the tool for you.

*This tool was built for my own use to keep an offline copy of Slack data for me to use. It's
completely unofficial, and is not affiliated with Slack in any way. I make no guarantee that it'll
work (or keep on working). Use it at your own risk.*

Current Features
----------------

This tool can supplement an official Slack team export by adding the following to it:

* File Uploads

Under Development
-----------------

Exporting of the following items from Slack is still under development:

* Private group messages (only the ones you are a member of).
* Private messages (only those you are a party to).

Installation
------------

    git clone https://github.com/grundleborg/slack-advanced-exporter.git
    cd slack-advanced-exporter
    pip install requirements.txt

(You might like to use a Python virtualenvironment so you don't need to install the tool's
requirements system wide).

Usage
-----

First, run a full export of your Slack team, and have the produced zip file handy.

Interactive Mode
++++++++++++++++

The simplest way to use this tool is in interactive mode, where you will be prompted to go through
the process entirely.

Just run the command below, and follow the prompts provided:

    ./export.py -i


If your Slack team uses an authentication scheme other than simple email/password (or uses 2FA),
interactive mode won't be able to log you in. Instead you should extract your Slack cookies from
a logged-in session in a web browser, and put them in a file. You can then run interactive mode
as follows:

    ./export.py -c COOKIE_FILE -t yourteam.slack.com -i

Non-Interactive Mode
++++++++++++++++++++

If you don't want to use interactive mode, you will need to provide a Slack session cookie
as above. You can then run commands using flags to ```export.py```. For example, to download all
file uploads to go with a team export, use the following command line:

    ./export.py -c COOKIE_FILE -t yourteam.slack.com -e path/to/export.zip --download-attachments

For all available command line flags, run:

    ./export.py --help

Problems
--------

If you encounter a bug in this tool, please file a bug report on its Github issue tracker.

Contributing
------------

Pull requests are welcome. Please ensure that you comply with the requirements below:

* Only data that is not included in the official team export feature should be exported by this
  tool - if it's already in the official export, there's no need for it here.
* Ensure you only request what is needed, to avoid unnecessary load being placed on servers.


