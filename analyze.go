package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	cache "github.com/bitrise-io/go-xcode/xcodecache"
	"github.com/bitrise-io/go-xcode/xcpretty"
)

func runCommandWithRetry(cmd *xcodebuild.CommandBuilder, useXcpretty bool, swiftPackagesPath string) (string, error) {
	output, err := runCommand(cmd, useXcpretty)
	if err != nil && swiftPackagesPath != "" && strings.Contains(output, cache.SwiftPackagesStateInvalid) {
		log.Warnf("Build failed, swift packages cache is in an invalid state, error: %s", err)
		log.RWarnf("xcode-analyze", "swift-packages-cache-invalid", nil, "swift packages cache is in an invalid state")
		if err := os.RemoveAll(swiftPackagesPath); err != nil {
			return output, fmt.Errorf("failed to remove invalid Swift package caches, error: %s", err)
		}
		return runCommand(cmd, useXcpretty)
	}
	return output, err
}

func prepareCommand(xcodeCmd *xcodebuild.CommandBuilder, useXcpretty bool, output *bytes.Buffer) (*command.Model, *xcpretty.CommandModel) {
	if useXcpretty {
		return nil, xcpretty.New(xcodeCmd)
	}

	buildRootCmd := xcodeCmd.Command()
	buildRootCmd.SetStdout(io.MultiWriter(os.Stdout, output))
	buildRootCmd.SetStderr(io.MultiWriter(os.Stderr, output))
	return buildRootCmd, nil
}

func runCommand(buildCmd *xcodebuild.CommandBuilder, useXcpretty bool) (string, error) {
	var output bytes.Buffer
	xcodebuildCmd, xcprettyCmd := prepareCommand(buildCmd, useXcpretty, &output)

	if xcprettyCmd != nil {
		logWithTimestamp(colorstring.Green, "$ %s", xcprettyCmd.PrintableCmd())
		fmt.Println()
		return xcprettyCmd.Run()
	}
	logWithTimestamp(colorstring.Green, "$ %s", xcodebuildCmd.PrintableCommandArgs())
	fmt.Println()
	return output.String(), xcodebuildCmd.Run()
}
