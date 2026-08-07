package main

// writeDesktopFiles creates TeamID.txt and its shortcut, as well as links
// to the ScoringReport, ReadMe, and other needed files. Identical to the
// Linux behavior -- FreeBSD desktop environments (installed via pkg) honor
// the same freedesktop.org .desktop entry format.
func writeDesktopFiles() {
	info("Creating or emptying TeamID.txt...")
	shellCommand("echo 'YOUR-TEAMID-HERE' > " + dirPath + "TeamID.txt")
	shellCommand("chmod 666 " + dirPath + "TeamID.txt")
	shellCommand("chown " + conf.User + ":" + conf.User + " " + dirPath + "TeamID.txt")
	info("Writing shortcuts to Desktop...")
	shellCommand("mkdir -p /home/" + conf.User + "/Desktop/")
	shellCommand("cp " + dirPath + "misc/desktop/*.desktop /home/" + conf.User + "/Desktop/")
	shellCommand("chmod +x /home/" + conf.User + "/Desktop/*.desktop")
	shellCommand("chown " + conf.User + ":" + conf.User + " /home/" + conf.User + "/Desktop/*")
}

// configureAutologin configures the auto-login capability for LightDM and
// GDM, both of which are installed under /usr/local on FreeBSD. If neither
// display manager is present, it falls back to FreeBSD's native console
// autologin mechanism (a login.conf class + getty flag on ttyv0).
func configureAutologin() {
	lightdm, _ := cond{Path: "/usr/local/etc/lightdm"}.PathExists()
	gdm, _ := cond{Path: "/usr/local/etc/gdm"}.PathExists()
	if lightdm {
		info("LightDM detected for autologin.")
		shellCommand("mkdir -p /usr/local/etc/lightdm/lightdm.conf.d")
		shellCommand(`printf '[Seat:*]\nautologin-user=` + conf.User + `\n' >> /usr/local/etc/lightdm/lightdm.conf.d/50-freebsd.conf`)
	} else if gdm {
		info("GDM detected for autologin.")
		shellCommand(`printf '[daemon]\nAutomaticLoginEnable=true\nAutomaticLogin=` + conf.User + `\n' >> /usr/local/etc/gdm/custom.conf`)
	} else {
		info("No supported display manager detected; falling back to console autologin.")
		configureConsoleAutologin()
	}
}

// configureConsoleAutologin sets up automatic login on the primary virtual
// console (ttyv0) using a dedicated login class, which is FreeBSD's native
// mechanism for autologin when there's no graphical display manager
// installed. This is a best-effort convenience -- please verify it actually
// took effect (reboot and check) before relying on it for a competition.
func configureConsoleAutologin() {
	className := "al_" + conf.User
	script := `
set -e
if ! grep -q '^` + className + `|' /etc/login.conf 2>/dev/null; then
	printf '` + className + `|Auto-login class for ` + conf.User + `:\n\t:tc=default:\n' >> /etc/login.conf
	cap_mkdb /etc/login.conf
fi
sed -i '' -E 's|^ttyv0[[:space:]].*|ttyv0\t"/usr/libexec/getty al.` + conf.User + `"\txterm\ton\tsecure|' /etc/ttys
`
	if err := shellCommand(script); err != nil {
		fail("Unable to configure autologin! Please do so manually.")
	}
}

// installFont is skipped for FreeBSD, same as Linux.
func installFont() {
	info("Skipping font install for FreeBSD...")
}

// installService for FreeBSD installs and starts the aeacus client (phocus)
// as an rc.d service, managed via daemon(8) for pidfile tracking/respawn
// protection -- the closest FreeBSD-native analog to Linux's
// start-stop-daemon-based init.d script.
func installService() {
	info("Installing service...")
	rcScript := `#!/bin/sh
#
# PROVIDE: aeacus
# REQUIRE: NETWORKING DAEMON
# KEYWORD: shutdown

. /etc/rc.subr

name="aeacus"
rcvar="aeacus_enable"
pidfile="/var/run/${name}.pid"
command="/usr/sbin/daemon"
command_args="-P ${pidfile} -r -f ` + dirPath + `phocus"

load_rc_config $name
: ${aeacus_enable:="NO"}

run_rc_command "$1"
`
	writeFile("/usr/local/etc/rc.d/aeacus", rcScript)
	shellCommand("chmod +x /usr/local/etc/rc.d/aeacus")
	shellCommand(`sysrc aeacus_enable="YES"`)
	shellCommand("service aeacus start")
}

// cleanUp for FreeBSD is primarily focused on removing cached files,
// history, and other pieces of forensic evidence. It also removes the
// non-required files in the aeacus directory.
func cleanUp() {
	findPaths := "/bin /etc /home /root /sbin /usr /var"

	info("Changing perms to 755 in " + dirPath + "...")
	shellCommand("chmod 755 -R " + dirPath)

	info("Removing aeacus binary...")
	shellCommand("rm " + dirPath + "aeacus")

	info("Removing scoring.conf...")
	shellCommand("rm " + dirPath + "scoring.conf*")

	info("Removing other setup files...")
	shellCommand("rm -rf " + dirPath + "misc/")
	shellCommand("find " + dirPath + " -name '[R|r]*.conf' -type f -delete")
	shellCommand("rm -rf " + dirPath + "README.md")
	shellCommand("rm -rf " + dirPath + ".git")
	shellCommand("rm -rf " + dirPath + ".github")
	shellCommand("rm -rf " + dirPath + "*.go")
	shellCommand("rm -rf " + dirPath + "Makefile")
	shellCommand("rm -rf " + dirPath + "go.*")
	shellCommand("rm -rf " + dirPath + "docs")

	if !ask("Do you want to remove cache and log files, overwrite timestamps, and remove other forensic data from this machine? This may impact data used for your forensic questions!") {
		return
	}

	info("Removing .viminfo and .swp files...")
	shellCommand("find " + findPaths + " -iname '*.viminfo*' -delete -iname '*.swp' -delete")

	info("Symlinking .bash_history and .zsh_history to /dev/null...")
	shellCommand(`find ` + findPaths + ` -iname '*.bash_history' -exec ln -sf /dev/null {} \;`)
	shellCommand(`find ` + findPaths + ` -name '.zsh_history' -exec ln -sf /dev/null {} \;`)

	info("Symlinking .history (csh/tcsh, FreeBSD's default root shell) to /dev/null...")
	shellCommand(`find ` + findPaths + ` -name '.history' -exec ln -sf /dev/null {} \;`)

	info("Removing .mysql_history...")
	shellCommand(`find ` + findPaths + ` -name '.mysql_history' -exec rm {} \;`)

	info("Removing .local files...")
	shellCommand("rm -rf /root/.local /home/*/.local/")

	info("Removing cache...")
	shellCommand("rm -rf /root/.cache /home/*/.cache/")

	info("Removing temp root and Desktop files...")
	shellCommand("rm -rf /root/*~ /home/*/Desktop/*~")

	info("Removing crash dumps...")
	shellCommand("rm -f /var/crash/*")

	info("Clearing pkg cache...")
	shellCommand("pkg clean -ay")

	info("Removing logs (auth, messages, security)...")
	shellCommand("rm -f /var/log/auth.log* /var/log/messages* /var/log/security*")

	info("Installing BleachBit...")
	shellCommand("pkg install -y bleachbit")

	info("Clearing Firefox cache and browsing history...")
	shellCommand("bleachbit --clean firefox.url_history; bleachbit --clean firefox.cache")

	info("Overwriting timestamps to obfuscate changes...")
	shellCommand(`find /etc /home /var -exec touch -h -t 201212121212 {} \; 2>/dev/null`)
}
