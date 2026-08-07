# aeacus

## Checks

This is a list of vulnerability checks that can be used in the configuration for `aeacus`. The notes on this page
contain a lot of important information, please be sure to read them.

> **Note**: Each of the commands here can check for the opposite by appending 'Not' to the check type. For
> example, `PathExistsNot` to pass if a file does not exist.

> **Note**: If a check has negative points assigned to it, it automatically becomes a penalty.

> **Note**: Each of these check types can be used for `Pass`, `PassOverride` or `Fail` conditions, and there can be
> multiple conditions per check. See [configuration](config.md) for more details.

> **Note**: Regex is officially supported for `CommandContainsRegex`, `DirContainsRegex`, and `FileContainsRegex`. Read
> more about regex [here](regex.md).

**CommandContains**: pass if command output contains string. If executing the command fails (the check returns an
error), check never passes. Use of this check is discouraged.

```
type = 'CommandContains'
cmd = 'ufw status'
value = 'Status: active'
```

> **Note**: `Command*` checks are prone to interception, modification, and tomfoolery. Your scoring configuration will
> be much more robust if you rely on checks using native mechanisms rather than shell commands (for
> example, `PathExists`
> instead of ls).

> **Note**: If any check returns an error (e.g., something that it was not expecting), it will _never_ pass, even if
> it's a `Not` condition. This varies by check, but for example, if you try to check the content of a file that doesn't
> exist, it will return an error and not succeed-- even if you were doing `FileContainsNot`.

**CommandOutput**: pass if command output matches string exactly. If it returns an error, check never passes. Use of
this check is discouraged.

```
type = 'CommandOutput'
cmd = '(Get-NetFirewallProfile -Name Domain).Enabled'
value = 'True'
```

**DirContains**: pass if directory contains a string value

```
type = 'DirContains'
path = '/etc/sudoers.d/'
value = 'NOPASSWD'
```

> `DirContains` is recursive! This means it checks every folder and subfolder. It currently is capped at 10,000 files,
> so you should begin your search at the deepest folder possible.


> **Note**: You don't have to escape any characters because we're using single quotes, which are literal strings in
> TOML. If you need use single quotes, use a TOML multi-line string
> literal `''' like this! that's neat! C:\path\here '''`), or just normal quotes (but you'll have to escape characters
> with those).

**FileContains**: pass if file contains a value

> **Note**: `FileContains` will never pass if file does not exist! Add an additional PassOverride check for
> PathExistsNot, if you want to score that a file does not contain a line, OR it doesn't exist.

```
type = 'FileContains'
path = 'C:\Users\coolUser\Desktop\Forensic Question 1.txt'
value = 'ANSWER:\s*Cool[a-zA-Z]+VariedAnswer'
```

**FileEquals**: pass if file equals sha256 hash

```
type = 'FileEquals'
path = '/etc/sysctl.conf'
value = 'e61ff3fb83b51fe9f2cd03cc0408afa15d4e8e69b8488b4ed1ecb854ae25da9b'
```

**FileOwner**: pass if specified user owns a given file

```
type = 'FileOwner'
path = 'C:\test.txt'
name = 'BUILTIN\Administrators'
```

```
type = 'FileOwner'
path = '/etc/passwd'
name = 'root'
```

> Get owner of the file in both Windows and Linux. You can see the owner of a file on Windows using
> PowerShell: `(Get-Acl [FILENAME]).Owner`. For Linux, use `ls -la FILENAME`.


**FirewallUp**: pass if firewall is active

```
type = 'FirewallUp'
```

> **Note**: On Linux, `ufw` (checks `/etc/ufw/ufw.conf`) and `firewalld` are supported. If the `ufw` config does not
> exist, the engine checks if `firewalld` is running. On Window, this passes if all three Windows Firewall profiles are
> active.

**PasswordChanged**: pass if user password has changed

For Linux, check if user's password hash is not next to their username in `/etc/shadow`. If you don't use the whole
hash, make sure you start it from the beginning (typically `$X$...` where X is a number).

```
type = 'PasswordChanged'
user = 'bob'
value = '$6$BgBsRlajjwVOoQCY$rw5WBSha4nkpynzfCzc3yYkV1OyDhr.ELoJOPpidwZoygUzRFBFSrtE3fyP0ITubCwN9Bb9DUqVV3mzTHL8sw/'
```

> This check will never pass if the user does not exist, so don't use this with users that should be removed.

For Windows, check if password was changed after the specified date:

```
type = 'PasswordChanged'
user = 'username'
after = 'Monday, January 02, 2006 3:04:05 PM'
```

> You should take the value from `(Get-LocalUser <USERNAME>).PasswordLastSet` and use it as `after`. This check will
> never pass if the user does not exist, so don't use this with users that should be removed.

**PathExists**: pass if specified path exists. This works for both files AND folders (directories).

```
type = 'PathExists'
path = '/var/www/backup.zip'
```

```
type = 'PathExists'
path = 'C:\importantfolder\'
```

**PermissionIs**: pass if specified user has specified permission on a given file

For Linux, use the standard octal `rwx` format (`ls -la yourfile` will show them). Use question marks to omit bits you
don't care about.

```
type = 'PermissionIs'
path = '/etc/shadow
value = 'rw-rw----'
```

For example, this one checks that /bin/bash is not SUID and world writable at the same time:

```
type = 'PermissionIsNot'
path = '/bin/bash'
value = '???s????w?'
```

So if `/bin/bash` is no longer world writable OR no longer SUID, the check will pass. If you want to ensure both
attributes are removed, you should use two conditions in the same check (`pass` for writeable bit, in addition to `pass`
for SUID bit).

For Windows, get a users permission of the file using `(Get-Acl [FILENAME]).Access`.

```
type = 'PermissionIs'
path = 'C:\test.txt'
name = 'BUILTIN\Administrators'
value = 'FullControl'
```

> **Note**: Use absolute paths when possible (rather than relative) for more reliable scoring.

**ProgramInstalled**: pass if program is installed. On Linux, will use `dpkg` (or `rpm` for RHEL-based systems), and on
Windows, checks if any installed programs contain your program string.

```
type = 'ProgramInstalled'
name = 'Mozilla Firefox 75 (x64 en-US)'
```

**ProgramVersion**: pass if a program meets the version requirements

For Linux, get version from `dpkg -s programnamehere`.

```
type = 'ProgramVersion'
name = 'Firefox'
value = '88.0.1+build1-0ubuntu0.20.04.2'
```

> Only works for `dpkg` based distributions (such as Debian and Ubuntu).

For Windows, get versions from `.\aeacus.exe info programs`.

```
# Checks version on first matching substring. E.g., for program name 'Ace',
# it may match on 'Ace Of Spades' rather than 'Ace Ventura'. Make your program
# name as detailed as possible.
type = 'ProgramVersion'
name = 'Firefox'
value = '95.0.1'
```

> **Note**: We recommend you use the `Not` version of this check to score a program's version being different from its
> version at the beginning of the image. You can't guarantee that the latest version of the program you're scoring will
> be
> the same once your round is released, and it's unlikely that a competitor will intentionally downgrade a package.

> For packages, Linux uses `dpkg`, Windows uses the Windows API

**ServiceUp**: pass if service is running

For Linux, use the `systemd` service name.

```
type = 'ServiceUp'
name = 'sshd'
```

For Windows: check the service 'Properties' to find the real service name

```
type = 'ServiceUp'
name = 'tapisrv' # this is telephony

```

> For services, Linux uses `systemctl`, Windows uses `Get-Service`. If you are using a different init system on Linux,
> you can use a `Command` check.

**UserExists**: pass if user exists on system

```
type = 'UserExists'
user = 'ballen'
```

**UserInGroup**: pass if specified user is in specified group

```
type = 'UserInGroupNot'
user = 'HackerUser'
group = 'Administrators'
```

> Linux reads `/etc/group` and Windows uses the Windows API.

<hr>


### FreeBSD-Specific Checks

`aeacus` on FreeBSD supports the same check set as Linux (with the underlying mechanisms remapped to FreeBSD's
tooling -- `pkg` instead of `dpkg`/`rpm`, `pf`/`ipfw` instead of `ufw`/`firewalld`, `service` instead of `systemctl`,
etc), plus a handful of checks unique to FreeBSD.

**AutoCheckUpdatesEnabled**: pass if the base system is configured to automatically check for security updates, via
either an active `freebsd-update cron` entry in `/etc/crontab`, or the `periodic(8)` daily security job

```
type = 'AutoCheckUpdatesEnabled'
```

**Command**: pass if command succeeds (command is executed, and has a return code of zero). Use of this check is
discouraged. This check will NOT return an error if the command is not found

```
type = 'Command'
cmd = 'cat coolfile.txt'
```

**GuestDisabledLDM**: pass if guest is disabled (for LightDM, installed under `/usr/local/etc`)

```
type = 'GuestDisabledLDM'
```

**KernelVersion**: pass if kernel version is equal to specified. Reads the `kern.osrelease` sysctl node, since
FreeBSD doesn't expose a Linux-style `uname(2)` syscall to Go

```
type = 'KernelVersion'
value = '14.1-RELEASE-p3'
```

> Tip: Check your `KernelVersion` with `uname -r`, or `sysctl kern.osrelease`.

**PasswordChanged**: pass if user password has changed, reading `/etc/master.passwd` (FreeBSD's shadow-equivalent)
instead of `/etc/shadow`

```
type = 'PasswordChanged'
user = 'coolUser'
value = '$6$SomeKnownDefaultHash'
```

**FileOwner**: pass if specified user owns a given file

```
type = 'FileOwner'
path = '/etc/rc.conf'
name = 'root'
```

**PermissionIs**: pass if specified path has the given permission string (9 or 10 characters, `?` as wildcard)

```
type = 'PermissionIs'
path = '/etc/master.passwd'
value = '-rw-------'
```

**ProgramInstalled**: pass if a package is installed, via `pkg info -e`

```
type = 'ProgramInstalled'
name = 'nmap'
```

**ProgramVersion**: pass if a package meets the version requirement, via `pkg query %v`

```
type = 'ProgramVersion'
name = 'openssh-portable'
value = '9.7.p1'
```

**ServiceUp**: pass if service is running, via `service <name> onestatus`

```
type = 'ServiceUp'
name = 'sshd'
```

**UserExists**: pass if user exists on system

```
type = 'UserExists'
user = 'coolUser'
```

**UserInGroup**: pass if specified user is in specified group

```
type = 'UserInGroup'
user = 'coolUser'
group = 'wheel'
```

Uhhhh from here I just started experimenting and ended up making these checks. I make no guarantee that they will work, have fun!

**SysctlValue**: pass if the given sysctl MIB's value matches the value provided

```
type = 'SysctlValue'
key = 'security.bsd.see_other_uids'
value = '0'
```

**PkgAuditClean**: pass if `pkg audit` reports no known vulnerabilities in currently installed packages

```
type = 'PkgAuditClean'
```

**RcVarEnabled**: pass if the given `rc.conf(5)` variable is set to `YES`/`true`, checking both `/etc/rc.conf` and
`/etc/rc.conf.d/<name>` overrides

```
type = 'RcVarEnabled'
name = 'sendmail_enable'
```

**JailRunning**: pass if a jail with the given name is currently running (via `jls`)

```
type = 'JailRunning'
name = 'webjail'
```

<hr>
