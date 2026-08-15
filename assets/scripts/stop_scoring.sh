#!/usr/bin/env sh

if /usr/local/bin/zenity --question \
	--text="Would you like to stop scoring for this image?" \
	--title="Aeacus SE"; then
	/usr/local/bin/notify-send -i /opt/aeacus/assets/img/logo.png "Aeacus SE" "Stopping scoring, and shutting down."
	if ! /usr/sbin/service aeacus stop && /usr/sbin/service aeacus onestatus >/dev/null 2>&1; then
		/usr/local/bin/notify-send -i /opt/aeacus/assets/img/logo.png "Aeacus SE" "Unable to stop the aeacus service."
		exit 1
	fi
	/bin/rm -f /opt/aeacus/phocus /opt/aeacus/scoring.dat
	/sbin/shutdown -p now
else
	/usr/local/bin/notify-send -i /opt/aeacus/assets/img/logo.png "Aeacus SE" "Confirmation failed!"
fi
