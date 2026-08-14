package main

import "os"

func writeFileResult(fileName, fileContent string) error {
	return os.WriteFile(fileName, []byte(fileContent), 0o644)
}
