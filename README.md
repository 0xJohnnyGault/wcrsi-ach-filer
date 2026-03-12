# Setup

Open Powershell as Admin

`winget install -e --id GoLang.Go`
`winget install -e --id OpenJS.NodeJS.LTS`

`go install github.com/wailsapp/wails/v2/cmd/wails@latest`

`wails doctor`

Create a Github.com account

Install Github Desktop
https://central.github.com/deployments/desktop/desktop/latest/win32

Download this repo https://github.com/0xJohnnyGault/wcrsi-ach-filer using Github Desktop

# Prompt (paste into Claude)

Build a multi-platform app in the current directory. The app should be called ACHFiler and use Wails and React that will allow the user to select a folder on their computer and designate it as the Destination Folder, then the user can select another folder and designate it as Source Folder. These choices should be saved in a config file.

The program should follow these steps:

- Open the first XLS file found in the Source Directory
- Find the "Account" column in the XLS file (in Row 2), and loop through every Account Number, and for each account number:
  - FOUND_DIR_NAME = Search the Destination Folder for a sub-directory name that contains the Account Number
  - If no FOUND_DIR_NAME is found, output an error log for the user
  - Copy the contents of the Source Directory to the FOUND_DIR_NAME/Payments folder (create the Payments subdirectory if it does not exist)

# Usage

Once Claude builds the app, you can run these commands from a Powershell

`wails build` This will build the EXE and put it in the `build/bin` directory.
