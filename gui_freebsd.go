package main

// launchIDPrompt launches an ID Prompt on FreeBSD using zenity, same as on
// Linux. Install it with `pkg install zenity` if it isn't already present.
func launchIDPrompt() {
	err := shellCommand(`zenity --title "Team ID Prompt" --text "Enter your Team ID below!" --entry > ` + dirPath + "TeamID.txt")
	if err != nil {
		fail("Error running ID prompt command: " + err.Error())
	}
}
