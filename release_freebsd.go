package main

import "fmt"

type releaseCommand struct {
	description string
	command     string
}

func runReleaseCommands(commands ...releaseCommand) error {
	for _, command := range commands {
		info(command.description)
		if err := shellCommand(command.command); err != nil {
			return fmt.Errorf("%s: %w", command.description, err)
		}
	}
	return nil
}

// writeDesktopFiles creates TeamID.txt and its shortcut, as well as links
// to the ScoringReport, ReadMe, and other needed files. Identical to the
// Linux behavior -- FreeBSD desktop environments (installed via pkg) honor
// the same freedesktop.org .desktop entry format.
func writeDesktopFiles() error {
	return runReleaseCommands(
		releaseCommand{"Creating or emptying TeamID.txt...", "echo 'YOUR-TEAMID-HERE' > " + dirPath + "TeamID.txt"},
		releaseCommand{"Setting TeamID.txt permissions...", "chmod 666 " + dirPath + "TeamID.txt"},
		releaseCommand{"Setting TeamID.txt ownership...", "chown " + conf.User + ":" + conf.User + " " + dirPath + "TeamID.txt"},
		releaseCommand{"Creating the Desktop directory...", "mkdir -p /home/" + conf.User + "/Desktop/"},
		releaseCommand{"Writing shortcuts to Desktop...", "cp " + dirPath + "misc/desktop/*.desktop /home/" + conf.User + "/Desktop/"},
		releaseCommand{"Making Desktop shortcuts executable...", "chmod +x /home/" + conf.User + "/Desktop/*.desktop"},
		releaseCommand{"Setting Desktop shortcut ownership...", "chown " + conf.User + ":" + conf.User + " /home/" + conf.User + "/Desktop/*"},
	)
}

// configureAutologin configures the auto-login capability for LightDM and
// GDM, both of which are installed under /usr/local on FreeBSD. If neither
// display manager is present, it falls back to FreeBSD's native console
// autologin mechanism (a login.conf class + getty flag on ttyv0).
func configureAutologin() error {
	lightdm, err := cond{Path: "/usr/local/etc/lightdm"}.PathExists()
	if err != nil {
		return fmt.Errorf("detect LightDM: %w", err)
	}
	if lightdm {
		return runReleaseCommands(
			releaseCommand{"Creating LightDM configuration directory...", "mkdir -p /usr/local/etc/lightdm/lightdm.conf.d"},
			releaseCommand{"Configuring LightDM autologin...", `printf '[Seat:*]\nautologin-user=` + conf.User + `\n' >> /usr/local/etc/lightdm/lightdm.conf.d/50-freebsd.conf`},
		)
	}

	gdm, err := cond{Path: "/usr/local/etc/gdm"}.PathExists()
	if err != nil {
		return fmt.Errorf("detect GDM: %w", err)
	}
	if gdm {
		return runReleaseCommands(
			releaseCommand{"Configuring GDM autologin...", `printf '[daemon]\nAutomaticLoginEnable=true\nAutomaticLogin=` + conf.User + `\n' >> /usr/local/etc/gdm/custom.conf`},
		)
	}

	info("No supported display manager detected; falling back to console autologin.")
	return configureConsoleAutologin()
}

// configureConsoleAutologin sets up automatic login on the primary virtual
// console (ttyv0) using a dedicated login class, which is FreeBSD's native
// mechanism for autologin when there's no graphical display manager
// installed. This is a best-effort convenience -- please verify it actually
// took effect (reboot and check) before relying on it for a competition.
func configureConsoleAutologin() error {
	className := "al_" + conf.User
	script := `
set -e
if ! grep -q '^` + className + `|' /etc/login.conf 2>/dev/null; then
	printf '` + className + `|Auto-login class for ` + conf.User + `:\n\t:tc=default:\n' >> /etc/login.conf
	cap_mkdb /etc/login.conf
fi
sed -i '' -E 's|^ttyv0[[:space:]].*|ttyv0\t"/usr/libexec/getty al.` + conf.User + `"\txterm\ton\tsecure|' /etc/ttys
`
	return runReleaseCommands(releaseCommand{"Configuring console autologin...", script})
}

// installFont is skipped for FreeBSD, same as Linux.
func installFont() error {
	info("Skipping font install for FreeBSD...")
	return nil
}

// installService for FreeBSD installs and starts the aeacus client (phocus)
// as an rc.d service, managed via daemon(8) for pidfile tracking/respawn
// protection -- the closest FreeBSD-native analog to Linux's
// start-stop-daemon-based init.d script.
func installService() error {
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
	if err := writeFileResult("/usr/local/etc/rc.d/aeacus", rcScript); err != nil {
		return fmt.Errorf("write aeacus rc.d service: %w", err)
	}
	return runReleaseCommands(
		releaseCommand{"Making the aeacus service executable...", "/bin/chmod 0555 /usr/local/etc/rc.d/aeacus"},
		releaseCommand{"Enabling the aeacus service...", `/usr/sbin/sysrc aeacus_enable="YES"`},
		releaseCommand{"Starting the aeacus service...", "/usr/sbin/service aeacus start"},
	)
}

// cleanUp for FreeBSD is primarily focused on removing cached files,
// history, and other pieces of forensic evidence. It also removes the
// non-required files in the aeacus directory.
func cleanUp() error {
	findPaths := "/bin /etc /home /root /sbin /usr /var"

	if err := runReleaseCommands(
		releaseCommand{"Changing perms to 755 in " + dirPath + "...", "chmod 755 -R " + dirPath},
		releaseCommand{"Removing aeacus binary...", "rm " + dirPath + "aeacus"},
		releaseCommand{"Removing scoring.conf...", "rm " + dirPath + "scoring.conf*"},
		releaseCommand{"Removing misc files...", "rm -rf " + dirPath + "misc/"},
		releaseCommand{"Removing ReadMe configuration...", "find " + dirPath + " -name '[R|r]*.conf' -type f -delete"},
		releaseCommand{"Removing README.md...", "rm -rf " + dirPath + "README.md"},
		releaseCommand{"Removing Git metadata...", "rm -rf " + dirPath + ".git " + dirPath + ".github"},
		releaseCommand{"Removing Go source...", "rm -rf " + dirPath + "*.go " + dirPath + "go.*"},
		releaseCommand{"Removing Makefile...", "rm -rf " + dirPath + "Makefile"},
		releaseCommand{"Removing documentation...", "rm -rf " + dirPath + "docs"},
	); err != nil {
		return err
	}

	if !ask("Do you want to remove cache and log files, overwrite timestamps, and remove other forensic data from this machine? This may impact data used for your forensic questions!") {
		return nil
	}

	return runReleaseCommands(
		releaseCommand{"Removing .viminfo and .swp files...", "find " + findPaths + " -iname '*.viminfo*' -delete -iname '*.swp' -delete"},
		releaseCommand{"Symlinking .bash_history to /dev/null...", `find ` + findPaths + ` -iname '*.bash_history' -exec ln -sf /dev/null {} \;`},
		releaseCommand{"Symlinking .zsh_history to /dev/null...", `find ` + findPaths + ` -name '.zsh_history' -exec ln -sf /dev/null {} \;`},
		releaseCommand{"Symlinking .history to /dev/null...", `find ` + findPaths + ` -name '.history' -exec ln -sf /dev/null {} \;`},
		releaseCommand{"Removing .mysql_history...", `find ` + findPaths + ` -name '.mysql_history' -exec rm {} \;`},
		releaseCommand{"Removing .local files...", "rm -rf /root/.local /home/*/.local/"},
		releaseCommand{"Removing cache...", "rm -rf /root/.cache /home/*/.cache/"},
		releaseCommand{"Removing temp root and Desktop files...", "rm -rf /root/*~ /home/*/Desktop/*~"},
		releaseCommand{"Removing crash dumps...", "rm -f /var/crash/*"},
		releaseCommand{"Clearing pkg cache...", "pkg clean -ay"},
		releaseCommand{"Removing logs...", "rm -f /var/log/auth.log* /var/log/messages* /var/log/security*"},
		releaseCommand{"Overwriting timestamps...", `find /etc /home /var -exec touch -h -t 201212121212 {} \; 2>/dev/null`},
	)
}
