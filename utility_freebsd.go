package main

import (
	"crypto/md5"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

// readFile (FreeBSD) wraps ioutil's ReadFile function.
func readFile(fileName string) (string, error) {
	fileContent, err := ioutil.ReadFile(fileName)
	return string(fileContent), err
}

// decodeString (FreeBSD) strictly does nothing, however it's here for
// compatibility with Windows ANSI/UNICODE/etc.
func decodeString(fileContent string) (string, error) {
	return fileContent, nil
}

// sendNotification sends a notification to the end user. Unlike Linux
// (where systemd-logind reliably parks the user's D-Bus session socket at
// /run/user/$uid/bus), FreeBSD has no single standard location for it, since
// it depends on how the session was started (dbus-launch, slim, lightdm,
// etc). This looks up the address from a running dbus-daemon's environment
// via procstat(1), falling back to the classic ~/.dbus/session-bus file.
func sendNotification(messageString string) {
	if conf.User == "" {
		fail("User not specified in configuration, can't send notification.")
		return
	}
	err := shellCommand(`
		user="` + conf.User + `"
		dbuspid="$(pgrep -u "$user" -x dbus-daemon | head -n1)"
		display=""
		if [ -n "$dbuspid" ]; then
			display="$(procstat -e "$dbuspid" 2>/dev/null | awk -F'DBUS_SESSION_BUS_ADDRESS=' 'NF>1 {print $2; exit}')"
		fi
		if [ -z "$display" ]; then
			busfile="$(ls /home/$user/.dbus/session-bus/* 2>/dev/null | head -n1)"
			if [ -n "$busfile" ]; then
				display="$(grep -m1 DBUS_SESSION_BUS_ADDRESS "$busfile" | cut -d= -f2-)"
			fi
		fi
		su -m "$user" -c "env DISPLAY=:0 DBUS_SESSION_BUS_ADDRESS=\"$display\" notify-send -i ` + dirPath + `assets/img/logo.png \"Aeacus SE\" \"` + messageString + `\""
	`)
	if err != nil {
		fail("Sending notification failed. Is the user in the configuration correct, and are they logged in to a desktop environment?")
	}
}

// checkTrace protects the scoring engine from being attached to by a
// debugger. FreeBSD has no /proc/self/status TracerPid field to read (even
// with procfs mounted), so instead it uses the standard self-ptrace trick:
// PT_TRACE_ME succeeds unless the calling process is already being traced,
// in which case it fails with EBUSY.
func checkTrace() {
	const (
		sysPtrace = syscall.SYS_PTRACE
		ptTraceMe = 0
	)
	_, _, errno := syscall.Syscall6(uintptr(sysPtrace), uintptr(ptTraceMe), 0, 0, 0, 0, 0)
	if errno != 0 {
		fail("Try harder instead of ptracing the engine, please.")
		os.Exit(1)
	}
}

// CreateFQs is a quality of life function that creates Forensic Question
// files on the Desktop, pre-populated with a template.
func CreateFQs(numFqs int) {
	for i := 1; i <= numFqs; i++ {
		fileName := "'Forensic Question " + strconv.Itoa(i) + ".txt'"
		shellCommand("echo 'QUESTION:' > /home/" + conf.User + "/Desktop/" + fileName)
		shellCommand("echo 'ANSWER:' >> /home/" + conf.User + "/Desktop/" + fileName)
		info("Wrote " + fileName + " to Desktop")
	}
}

// rawCmd returns an exec.Command object for FreeBSD shell commands. /bin/sh
// is guaranteed to exist in the base system, unlike bash.
func rawCmd(commandGiven string) *exec.Cmd {
	return exec.Command("/bin/sh", "-c", commandGiven)
}

// playAudio plays a .wav file with the given path. FreeBSD's base system has
// no bundled audio player (there's no ALSA/aplay); this uses sox's `play`
// utility, which is a common, lightweight choice available via
// `pkg install sox`.
func playAudio(wavPath string) {
	info("Playing audio:", wavPath)
	commandText := "play -q " + wavPath
	shellCommand(commandText)
}

// hashFileMD5 generates the MD5 Hash of a file with the given path.
func hashFileMD5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	return hexEncode(string(hashInBytes)), err
}

func adminCheck() bool {
	currentUser, err := user.Current()
	uid, _ := strconv.Atoi(currentUser.Uid)
	if err != nil {
		fail("Error for checking if running as root: " + err.Error())
		return false
	} else if uid != 0 {
		return false
	}
	return true
}

func getInfo(infoType string) {
	warn("Info gathering is not supported for FreeBSD-- there's always a better, easier command-line tool.")
}
