package main

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"
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

// shQuote wraps s in single quotes for safe embedding in a POSIX /bin/sh
// command line, escaping any single quotes it already contains. This is
// needed anywhere a value we don't fully control (configuration data, a
// check message, etc.) gets interpolated into a shell string, so that
// characters like ", a backtick, $, or a stray ' can't break out of their
// intended context.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// sendNotification sends a notification to the end user. Unlike Linux
// (where systemd-logind reliably parks the user's D-Bus session socket at
// /run/user/$uid/bus), FreeBSD has no single standard location for it, since
// it depends on how the session was started (dbus-launch, slim, lightdm,
// etc). This tries three ways to find it, in order of reliability:
//
//  1. The session bus's own socket file under /tmp (dbus-launch's default
//     location on FreeBSD). This is the most reliable option because it
//     doesn't depend on reading another process's environment at all.
//  2. A running dbus-daemon's environment, read via procstat(1). Note this
//     can silently come up empty on a real desktop session: FreeBSD caps
//     how many bytes of a process's argv+environ the kernel keeps around
//     for tools like procstat to read (kern.ps_arg_cache_limit, often 256
//     bytes by default), and a full DE session environment can easily
//     exceed that, truncating or dropping DBUS_SESSION_BUS_ADDRESS
//     entirely before procstat ever sees it.
//  3. The classic ~/.dbus/session-bus file.
//
// If the underlying command fails, this surfaces its actual stderr rather
// than a generic message, since "failed" alone doesn't distinguish between
// (for example) no user session existing at all versus a notification
// daemon simply not being registered to receive the message.
func sendNotification(messageString string) {
	if conf.User == "" {
		fail("User not specified in configuration, can't send notification.")
		return
	}
	user := shQuote(conf.User)
	script := `
		user=` + user + `
		display=""

		busfile="$(find /tmp -maxdepth 1 -type s -name 'dbus-*' -user "$user" 2>/dev/null | head -n1)"
		if [ -n "$busfile" ]; then
			display="unix:path=$busfile"
		fi

		if [ -z "$display" ]; then
			dbuspid="$(pgrep -u "$user" -x dbus-daemon | head -n1)"
			if [ -n "$dbuspid" ]; then
				display="$(procstat -e "$dbuspid" 2>/dev/null | awk -F'DBUS_SESSION_BUS_ADDRESS=' 'NF>1 {print $2; exit}')"
			fi
		fi

		if [ -z "$display" ]; then
			busaddrfile="$(ls /home/$user/.dbus/session-bus/* 2>/dev/null | head -n1)"
			if [ -n "$busaddrfile" ]; then
				display="$(grep -m1 DBUS_SESSION_BUS_ADDRESS "$busaddrfile" | cut -d= -f2-)"
			fi
		fi

		if [ -z "$display" ]; then
			echo "no D-Bus session address found for user $user (no /tmp socket, no dbus-daemon process, no ~/.dbus/session-bus file)"
			exit 1
		fi

		su -m "$user" -c "env DISPLAY=:0 DBUS_SESSION_BUS_ADDRESS=\"$display\" notify-send -i ` + shQuote(dirPath+"assets/img/logo.png") + ` \"Aeacus SE\" ` + shQuote(messageString) + `" 2>&1
	`
	// CombinedOutput (rather than shellCommandOutput/Output) is used
	// deliberately here: Output() discards stdout on a nonzero exit, which
	// would throw away exactly the diagnostic text this script produces
	// when it fails, leaving us back at an undiagnosable generic error.
	out, err := rawCmd(script).CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail == "" {
			detail = err.Error()
		}
		fail("Sending notification failed: " + detail)
	}
}

// pTraced is P_TRACED from <sys/proc.h> -- the bit the kernel sets in a
// process's own flags for as long as something is attached to it via
// ptrace(2). It has been stable at this value across every FreeBSD release
// that still receives support.
const pTraced = 0x00000800

// checkTrace protects the scoring engine from being attached to by a
// debugger or tracer.
//
// FreeBSD has no /proc/self/status TracerPid field to read (even with
// procfs mounted). The previous implementation worked around that by
// calling ptrace(PT_TRACE_ME) on itself and treating failure as "already
// traced". That is not an observational check: a successful PT_TRACE_ME
// call *changes* the calling process's tracing state by declaring the
// caller's parent as its tracer, so every call after the first one runs
// against a process that is now already in a tracing relationship and can
// fail (or behave inconsistently) for reasons that have nothing to do with
// a debugger being attached. Since Phocus calls this once at startup and
// then again on every scoring pass, that bug could crash a clean image.
//
// This version reads the same fact -- "is anything currently ptrace-ing
// this process?" -- straight from the kernel's own bookkeeping, without
// touching it. FreeBSD exposes that via the P_TRACED bit in the process's
// kinfo_proc flags (KERN_PROC_PID sysctl). Rather than hand-decode that
// struct ourselves (its layout has shifted across major FreeBSD releases,
// e.g. the FreeBSD 12 ino64 transition, and differs enough between
// architectures that a hardcoded offset table is an easy way to silently
// read the wrong field), we ask the base-system `ps(1)` utility for it via
// `ps -o flags=`. ps is built and shipped as part of the same release as
// the kernel it runs against, so it always decodes the struct using the
// layout that release actually uses -- we don't have to track that
// ourselves, and we can't get it wrong in a way that quietly reads garbage.
//
// Because this never mutates any tracing state, it is safe to call from
// both engine startup and every subsequent scoring pass.
//
// If the query itself fails (ps missing, unexpected output, etc.) this
// fails closed: we can't prove the process isn't traced, so an
// unverifiable result is treated the same as a positive detection.
func checkTrace() {
	traced, err := isBeingTraced()
	if err != nil {
		fail("Unable to verify engine process state: " + err.Error())
		os.Exit(1)
	}
	if traced {
		fail("Try harder instead of ptracing the engine, please.")
		os.Exit(1)
	}
}

// isBeingTraced reports whether the current process currently has the
// P_TRACED flag set, without altering any tracing state.
func isBeingTraced() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pid := strconv.Itoa(os.Getpid())
	// Fixed absolute path and argument vector -- no shell, no PATH lookup,
	// nothing derived from configuration or check data.
	cmd := exec.CommandContext(ctx, "/bin/ps", "-p", pid, "-o", "flags=")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("querying process flags via ps(1): %w", err)
	}
	return parseTracedFlag(string(out))
}

// parseTracedFlag interprets the hexadecimal process-flags field printed by
// `ps -o flags=` (documented in ps(1) as the raw kinfo_proc/p_flag value
// from <sys/proc.h>) and reports whether P_TRACED is set within it.
func parseTracedFlag(psOutput string) (bool, error) {
	field := strings.TrimSpace(psOutput)
	field = strings.TrimPrefix(field, "0x")
	field = strings.TrimPrefix(field, "0X")
	if field == "" {
		return false, errors.New("empty process flags field from ps(1)")
	}
	flags, err := strconv.ParseUint(field, 16, 64)
	if err != nil {
		return false, fmt.Errorf("unparseable process flags field %q from ps(1): %w", field, err)
	}
	return flags&pTraced != 0, nil
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
