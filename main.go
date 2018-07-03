package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/stringutil"
	"github.com/bitrise-io/steps-xcode-analyze/utils"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/bitrise-tools/go-xcode/xcodebuild"
	"github.com/bitrise-tools/go-xcode/xcpretty"
)

const (
	bitriseXcodeRawResultTextEnvKey = "BITRISE_XCODE_RAW_RESULT_TEXT_PATH"
	minSupportedXcodeMajorVersion   = 6
)

// Config ...
type Config struct {
	Workdir                  string `env:"workdir"`
	ProjectPath              string `env:"project_path,required"`
	Scheme                   string `env:"scheme,required"`
	IsCleanBuild             bool   `env:"is_clean_build,opt[yes,no]"`
	ForceProvisioningProfile string `env:"force_provisioning_profile"`
	ForceCodeSignIdentity    string `env:"force_code_sign_identity"`
	DisableCodesign          bool   `env:"disable_codesign,opt[yes,no]"`
	OutputTool               string `env:"output_tool,opt[xcpretty,xcodebuild]"`
	OutputDir                string `env:"output_dir,dir"`

	VerboseLog bool `env:"verbose_log,opt[yes,no]"`
}

func main() {
	var conf Config
	if err := stepconf.Parse(&conf); err != nil {
		log.Errorf("Error: %s\n", err)
		os.Exit(1)
	}
	stepconf.Print(conf)
	log.SetEnableDebugLog(conf.VerboseLog)

	fmt.Println()
	log.Infof("Step determined configs:")

	// detect Xcode version
	xcVersion, err := xcodeVersion()
	if err != nil {
		fail("Failed to get Xcode version - %s", err)
	}
	log.Printf(xcVersion)

	// Detect xcpretty version
	if conf.OutputTool == "xcpretty" {
		fmt.Println()
		log.Infof("Checking output tool")
		installed, err := utils.IsXcprettyInstalled()
		if err != nil {
			fail("Failed to check if xcpretty is installed, error: %s", err)
		}

		if !installed {
			log.Warnf(`🚨  xcpretty is not installed`)

			fmt.Println()
			log.Printf("Installing xcpretty")
			if err := utils.InstallXcpretty(); err != nil {
				fail("Failed to install xcpretty, error: %s", err)
			}
		}

		xcprettyVersion, err := utils.XcprettyVersion()
		if err != nil {
			fail("Failed to determin xcpretty version, error: %s", err)
		}
		log.Printf("- xcprettyVersion: %s", xcprettyVersion.String())
	}

	// Output files
	rawXcodebuildOutputLogPath := filepath.Join(conf.OutputDir, "raw-xcodebuild-output.log")

	//
	// Cleanup
	filesToCleanup := []string{
		rawXcodebuildOutputLogPath,
	}

	for _, pth := range filesToCleanup {
		if exist, err := pathutil.IsPathExists(pth); err != nil {
			fail("Failed to check if path (%s) exist, error: %s", pth, err)
		} else if exist {
			if err := os.RemoveAll(pth); err != nil {
				fail("Failed to remove path (%s), error: %s", pth, err)
			}
		}
	}

	//
	// Analyze project with Xcode Command Line tools
	fmt.Println()
	log.Infof("Analyzing the project")

	isWorkspace := false
	ext := filepath.Ext(conf.ProjectPath)
	if ext == ".xcodeproj" {
		isWorkspace = false
	} else if ext == ".xcworkspace" {
		isWorkspace = true
	} else {
		fail("Project file extension should be .xcodeproj or .xcworkspace, but got: %s", ext)
	}

	analyzeCmd := xcodebuild.NewCommandBuilder(conf.ProjectPath, isWorkspace, xcodebuild.AnalyzeAction)
	analyzeCmd.SetScheme(conf.Scheme)

	if conf.DisableCodesign {
		analyzeCmd.SetDisableCodesign(true)
	}

	if conf.OutputTool == "xcpretty" {
		xcprettyCmd := xcpretty.New(analyzeCmd)

		logWithTimestamp(colorstring.Green, "$ %s", xcprettyCmd.PrintableCmd())
		fmt.Println()

		if rawXcodebuildOut, err := xcprettyCmd.Run(); err != nil {
			log.Errorf("\nLast lines of the Xcode's build log:")
			fmt.Println(stringutil.LastNLines(rawXcodebuildOut, 10))

			if err := utils.ExportOutputFileContent(rawXcodebuildOut, rawXcodebuildOutputLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
				log.Warnf("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
			} else {
				log.Warnf(`You can find the last couple of lines of Xcode's build log above, but the full log is also available in the raw-xcodebuild-output.log
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable
(value: %s)`, rawXcodebuildOutputLogPath)
			}

			fail("Analyze failed, error: %s", err)
		}
	} else {
		logWithTimestamp(colorstring.Green, "$ %s", analyzeCmd.PrintableCmd())
		fmt.Println()

		analyzeRootCmd := analyzeCmd.Command()
		analyzeRootCmd.SetStdout(os.Stdout)
		analyzeRootCmd.SetStderr(os.Stderr)

		if err := analyzeRootCmd.Run(); err != nil {
			fail("Analyze failed, error: %s", err)
		}
	}
}

func xcodeVersion() (string, error) {
	// Detect Xcode major version
	xcodebuildVersion, err := utils.XcodeBuildVersion()
	if err != nil {
		return "", fmt.Errorf("failed to determin xcode version, error: %s", err)
	}

	xcodeMajorVersion := xcodebuildVersion.XcodeVersion.Segments()[0]
	if xcodeMajorVersion < minSupportedXcodeMajorVersion {
		return "", fmt.Errorf("invalid xcode major version (%d), should not be less then min supported: %d", xcodeMajorVersion, minSupportedXcodeMajorVersion)
	}

	return fmt.Sprintf("- xcodebuildVersion: %s (%s)", xcodebuildVersion.XcodeVersion.String(), xcodebuildVersion.BuildVersion), nil
}

func fail(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func currentTimestamp() string {
	timeStampFormat := "15:04:05"
	currentTime := time.Now()
	return currentTime.Format(timeStampFormat)
}

// ColoringFunc ...
type ColoringFunc func(...interface{}) string

func logWithTimestamp(coloringFunc ColoringFunc, format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	messageWithTimeStamp := fmt.Sprintf("[%s] %s", currentTimestamp(), coloringFunc(message))
	fmt.Println(messageWithTimeStamp)
}
