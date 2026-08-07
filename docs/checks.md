# aeacus (Modified)

I have included just the BSD specific checks, please refer to the aeacus docs for the normal checks if you are not familiar with them.



### FreeBSD Specific Checks

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
