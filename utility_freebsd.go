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

// sendNotification sends a notification to the end user, on any desktop
// environment (GNOME, KDE, XFCE, MATE, Cinnamon, LXQt, etc), whatever the
// DISPLAY value or session type (X11 or Wayland).
//
// notify-send only needs to reach the user's D-Bus *session bus*, so the
// real job is finding that bus address. FreeBSD has no single standard
// location for it (unlike systemd-logind on Linux), since it depends on how
// the session was started (dbus-launch, dbus-run-session, gdm, sddm, slim,
// lightdm, etc). Instead of trusting one method, this gathers every
// candidate address it can find, then actually tries notify-send against
// each one until a delivery succeeds:
//
//  1. DBUS_SESSION_BUS_ADDRESS from the environment of any of the user's
//     running processes, read via procstat(1). This is the authoritative
//     source on GNOME and most other full sessions. Note procstat can come
//     up empty for big environments, because FreeBSD caps how many bytes
//     of argv+environ the kernel keeps for it (kern.ps_arg_cache_limit),
//     which is why this is not the only method.
//  2. The usual runtime-directory sockets: $XDG_RUNTIME_DIR/bus,
//     /var/run/user/$uid/bus, /run/user/$uid/bus, /tmp/runtime-$user/bus.
//  3. The session bus's own socket file under /tmp (dbus-launch's default
//     location on FreeBSD).
//  4. The --address= argument of the user's running dbus-daemon, read via
//     ps(1) with unlimited width.
//  5. The classic ~/.dbus/session-bus/* files.
//
// DISPLAY, WAYLAND_DISPLAY and XDG_RUNTIME_DIR are likewise taken from the
// user's real session when it can be found, falling back to scanning
// /tmp/.X11-unix for the live X display (rather than assuming :0).
//
// The message, user and icon path are handed to the script through its
// environment rather than spliced into the script text, so nothing
// configurable can break out of shell quoting.
//
// If every candidate fails, this surfaces the actual error output rather
// than a generic message, since "failed" alone doesn't distinguish between
// (for example) no user session existing at all versus a notification
// daemon simply not being registered to receive the message.
func sendNotification(messageString string) {
	if conf.User == "" {
		fail("User not specified in configuration, can't send notification.")
		return
	}
	script := `
		PATH="/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/sbin:/usr/local/bin:$PATH"
		export PATH

		user="$AEACUS_USER"
		uid="$(id -u "$user" 2>/dev/null)"
		if [ -z "$uid" ]; then
			echo "user $user does not exist"
			exit 1
		fi
		home="$(pw usershow "$user" 2>/dev/null | cut -d: -f9)"
		if [ -z "$home" ]; then
			home="/home/$user"
		fi

		# Snapshot the environments of the user's running processes once.
		envdump="$(for pid in $(pgrep -u "$user" 2>/dev/null | head -n 200); do procstat -e "$pid" 2>/dev/null; done)"

		getenvvar() {
			printf '%s\n' "$envdump" | awk -v key="$1" '{ for (i = 1; i <= NF; i++) if (index($i, key "=") == 1) { print substr($i, length(key) + 2); exit } }'
		}

		# Runtime directory.
		xdg="$(getenvvar XDG_RUNTIME_DIR)"
		if [ -z "$xdg" ]; then
			for d in "/var/run/user/$uid" "/run/user/$uid" "/tmp/runtime-$user"; do
				if [ -d "$d" ]; then
					xdg="$d"
					break
				fi
			done
		fi

		# X11 display: session's own value, else whatever X socket is live.
		disp="$(getenvvar DISPLAY)"
		if [ -z "$disp" ]; then
			xsock="$(ls /tmp/.X11-unix 2>/dev/null | grep '^X[0-9]' | head -n1)"
			if [ -n "$xsock" ]; then
				disp=":${xsock#X}"
			else
				disp=":0"
			fi
		fi

		# Wayland display, if any.
		wl="$(getenvvar WAYLAND_DISPLAY)"
		if [ -z "$wl" ] && [ -n "$xdg" ]; then
			wl="$(ls "$xdg" 2>/dev/null | grep '^wayland-[0-9]*$' | head -n1)"
		fi

		export HOME="$home"
		export DISPLAY="$disp"
		if [ -n "$xdg" ]; then
			export XDG_RUNTIME_DIR="$xdg"
		fi
		if [ -n "$wl" ]; then
			export WAYLAND_DISPLAY="$wl"
		fi

		# Gather candidate session bus addresses, one per line.
		candidates=""
		add() {
			if [ -n "$1" ]; then
				candidates="$candidates
$1"
			fi
		}

		# 1. From the user's running processes.
		add "$(getenvvar DBUS_SESSION_BUS_ADDRESS)"

		# 2. Runtime directory sockets.
		for sock in "$xdg/bus" "/var/run/user/$uid/bus" "/run/user/$uid/bus" "/tmp/runtime-$user/bus"; do
			if [ -S "$sock" ]; then
				add "unix:path=$sock"
			fi
		done

		# 3. dbus-launch sockets under /tmp.
		for sock in $(find /tmp -maxdepth 2 -type s -name 'dbus-*' -user "$user" 2>/dev/null); do
			add "unix:path=$sock"
		done

		# 4. The running dbus-daemon's own --address argument.
		for a in $(ps -U "$user" -ww -o args= 2>/dev/null | grep 'dbus-daemon' | grep -o -e '--address=[^ ]*' | sed 's/^--address=//'); do
			add "$a"
		done

		# 5. Classic ~/.dbus/session-bus files.
		for f in "$home"/.dbus/session-bus/*; do
			if [ -f "$f" ]; then
				add "$(sed -n 's/^DBUS_SESSION_BUS_ADDRESS=//p' "$f" | head -n1 | tr -d "'\"")"
			fi
		done

		if [ -z "$(printf '%s' "$candidates" | tr -d '[:space:]')" ]; then
			echo "no D-Bus session address found for user $user (checked running processes, runtime-dir sockets, /tmp sockets, dbus-daemon arguments, and ~/.dbus/session-bus)"
			exit 1
		fi

		# Try each candidate until one delivers.
		tried=""
		lasterr=""
		oldifs="$IFS"
		IFS='
'
		for addr in $candidates; do
			IFS="$oldifs"
			if [ -z "$addr" ]; then
				continue
			fi
			case "$tried" in
				*"|$addr|"*) continue ;;
			esac
			tried="$tried|$addr|"
			out="$(DBUS_SESSION_BUS_ADDRESS="$addr" timeout 10 su -m "$user" -c 'notify-send -i "$AEACUS_ICON" "Aeacus SE" "$AEACUS_MSG"' 2>&1)"
			if [ $? -eq 0 ]; then
				exit 0
			fi
			lasterr="[$addr] $out"
		done

		echo "notify-send failed on every D-Bus address found; last attempt: $lasterr"
		exit 1
	`
	cmd := rawCmd(script)
	cmd.Env = append(os.Environ(),
		"AEACUS_USER="+conf.User,
		"AEACUS_ICON="+dirPath+"assets/img/logo.png",
		"AEACUS_MSG="+messageString,
	)
	// CombinedOutput (rather than shellCommandOutput/Output) is used
	// deliberately here: Output() discards stdout on a nonzero exit, which
	// would throw away exactly the diagnostic text this script produces
	// when it fails, leaving us back at an undiagnosable generic error.
	out, err := cmd.CombinedOutput()
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
