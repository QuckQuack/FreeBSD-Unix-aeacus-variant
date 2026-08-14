.DEFAULT_GOAL := bsd
.SILENT: release-bsd

bsd:
	CGO_ENABLED=0 GOOS=freebsd go build -ldflags '-s -w' -tags phocus -o ./phocus-freebsd . && \
	CGO_ENABLED=0 GOOS=freebsd go build -ldflags '-s -w' -o ./aeacus-freebsd . && \
	echo "FreeBSD production build successful!"

bsd-dev:
	CGO_ENABLED=0 GOOS=freebsd go build -tags phocus -o ./phocus-freebsd . && \
	CGO_ENABLED=0 GOOS=freebsd go build -o ./aeacus-freebsd . && \
	echo "FreeBSD development build successful!"

release-bsd:
	echo "Building obfuscated binaries..." && \
	sh misc/dev/gen-crypto.sh && \
	CGO_ENABLED=0 GOOS=freebsd go build -ldflags '-s -w' -tags phocus -o ./phocus.freebsd-bin . && \
	CGO_ENABLED=0 GOOS=freebsd go build -ldflags '-s -w' -o ./aeacus.freebsd-bin . && \
	echo "FreeBSD production build successful!" && \
	mv crypto.go.bak crypto.go && \
	echo "Restored crypto.go" && \
	mkdir aeacus-freebsd/ && \
	mv aeacus.freebsd-bin aeacus-freebsd/aeacus && \
	mv phocus.freebsd-bin aeacus-freebsd/phocus && \
	cp -Rf assets/ aeacus-freebsd/ && \
	cp -Rf misc/ aeacus-freebsd/ && \
	cp -Rf LICENSE aeacus-freebsd/ && \
	zip -r aeacus-freebsd.zip aeacus-freebsd/ > /dev/null && \
	echo "Successfully compressed aeacus-freebsd!" && \
	rm -rf aeacus-freebsd/
