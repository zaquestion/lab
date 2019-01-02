// +build windows

package cmd

// Add ".exe" to the end of the binary name so running the lab test binary works
// on Windows (the file extension must be in PATHEXT to execute on Windows).
const labBinary = "lab.test.exe"
