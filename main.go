package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/stringutil"
	"github.com/bitrise-io/go-xcode/utility"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	cache "github.com/bitrise-io/go-xcode/xcodecache"
	"github.com/bitrise-io/go-xcode/xcpretty"
	"github.com/bitrise-steplib/steps-xcode-archive/utils"
	"github.com/kballard/go-shellquote"
)

const (
	bitriseXcodeRawResultTextEnvKey = "BITRISE_XCODE_RAW_RESULT_TEXT_PATH"
	minSupportedXcodeMajorVersion   = 6
)

// Config ...
type Config struct {
	Workdir                   string `env:"workdir"`
	ProjectPath               string `env:"project_path,required"`
	Scheme                    string `env:"scheme,required"`
	IsCleanBuild              bool   `env:"is_clean_build,opt[yes,no]"`
	ForceProvisioningProfile  string `env:"force_provisioning_profile"`
	ForceCodeSignIdentity     string `env:"force_code_sign_identity"`
	DisableCodesign           bool   `env:"disable_codesign,opt[yes,no]"`
	DisableIndexWhileBuilding bool   `env:"disable_index_while_building,opt[yes,no]"`
	CacheLevel                string `env:"cache_level,opt[none,swift_packages]"`
	XcodebuildOptions         string `env:"xcodebuild_options"`
	OutputTool                string `env:"output_tool,opt[xcpretty,xcodebuild]"`
	OutputDir                 string `env:"output_dir,dir"`

	VerboseLog bool `env:"verbose_log,opt[yes,no]"`

	DeployDir string `env:"BITRISE_DEPLOY_DIR"`
}

func main() {
	var conf Config
	if err := stepconf.Parse(&conf); err != nil {
		fail("Failed to parse configs, error: %s", err)
	}

	stepconf.Print(conf)
	log.SetEnableDebugLog(conf.VerboseLog)

	fmt.Println()
	log.Infof("Step determined configs:")

	absProjectPath, err := filepath.Abs(conf.ProjectPath)
	if err != nil {
		fail("Failed to expand project path (%s), error: %s", conf.ProjectPath, err)
	}

	// Detect Xcode major version
	xcodebuildVersion, err := utility.GetXcodeVersion()
	if err != nil {
		fail("Failed to determine xcode version, error: %s", err)
	}
	log.Printf("- xcodebuildVersion: %s (%s)", xcodebuildVersion.Version, xcodebuildVersion.BuildVersion)

	xcodeMajorVersion := xcodebuildVersion.MajorVersion
	if xcodeMajorVersion < minSupportedXcodeMajorVersion {
		fail("Invalid xcode major version (%d), should not be less then min supported: %d", xcodeMajorVersion, minSupportedXcodeMajorVersion)
	}

	// Detect xcpretty version
	outputTool := conf.OutputTool
	if outputTool == "xcpretty" {
		fmt.Println()
		log.Infof("Checking if output tool (xcpretty) is installed")

		installed, err := xcpretty.IsInstalled()
		if err != nil {
			log.Warnf("Failed to check if xcpretty is installed, error: %s", err)
			log.Printf("Switching to xcodebuild for output tool")
			outputTool = "xcodebuild"
		} else if !installed {
			log.Warnf(`xcpretty is not installed`)
			fmt.Println()
			log.Printf("Installing xcpretty")

			if cmds, err := xcpretty.Install(); err != nil {
				log.Warnf("Failed to create xcpretty install command: %s", err)
				log.Warnf("Switching to xcodebuild for output tool")
				outputTool = "xcodebuild"
			} else {
				for _, cmd := range cmds {
					if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
						if errorutil.IsExitStatusError(err) {
							log.Warnf("%s failed: %s", out)
						} else {
							log.Warnf("%s failed: %s", err)
						}
						log.Warnf("Switching to xcodebuild for output tool")
						outputTool = "xcodebuild"
					}
				}
			}
		}
	}

	if outputTool == "xcpretty" {
		xcprettyVersion, err := xcpretty.Version()
		if err != nil {
			log.Warnf("Failed to determin xcpretty version, error: %s", err)
			log.Printf("Switching to xcodebuild for output tool")
			outputTool = "xcodebuild"
		}
		log.Printf("- xcprettyVersion: %s", xcprettyVersion.String())
	}

	// Output files
	rawXcodebuildOutputLogPath := filepath.Join(conf.OutputDir, "raw-xcodebuild-output.log")

	tempDir, err := ioutil.TempDir("", "XCOutput")
	if err != nil {
		fail("Could not create result bundle path directory: %s", err)
	}
	xcresultPath := path.Join(tempDir, "Analyze.xcresult")

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
	ext := filepath.Ext(absProjectPath)
	if ext == ".xcodeproj" {
		isWorkspace = false
	} else if ext == ".xcworkspace" {
		isWorkspace = true
	} else {
		fail("Project file extension should be .xcodeproj or .xcworkspace, but got: %s", ext)
	}

	analyzeCmd := xcodebuild.NewCommandBuilder(absProjectPath, isWorkspace, xcodebuild.AnalyzeAction)

	analyzeCmd.SetDisableIndexWhileBuilding(conf.DisableIndexWhileBuilding)
	analyzeCmd.SetScheme(conf.Scheme)

	if conf.DisableCodesign {
		analyzeCmd.SetDisableCodesign(true)
	}

	var customOptions []string
	if conf.XcodebuildOptions != "" {
		if customOptions, err = shellquote.Split(conf.XcodebuildOptions); err != nil {
			fail("failed to shell split XcodebuildOptions (%s), error: %s", conf.XcodebuildOptions, err)
		}

		analyzeCmd.SetCustomOptions(customOptions)
	}

	if !sliceutil.IsStringInSlice("-resultBundlePath", customOptions) {
		analyzeCmd.SetResultBundlePath(xcresultPath)
	}

	var swiftPackagesPath string
	if xcodeMajorVersion >= 11 {
		var err error
		if swiftPackagesPath, err = cache.SwiftPackagesPath(absProjectPath); err != nil {
			fail("Failed to get Swift Packages path, error: %s", err)
		}
	}

	rawXcodebuildOut, xcErr := runCommandWithRetry(analyzeCmd, outputTool == "xcpretty", swiftPackagesPath)
	if xcErr != nil {
		if outputTool == "xcpretty" {
			log.Errorf("\nLast lines of the Xcode's build log:")
			fmt.Println(stringutil.LastNLines(rawXcodebuildOut, 10))

			if err := utils.ExportOutputFileContent(rawXcodebuildOut, rawXcodebuildOutputLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
				log.Warnf("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
			} else {
				log.Warnf(`You can find the last couple of lines of Xcode's build log above, but the full log is also available in the raw-xcodebuild-output.log
	The log file is stored in $BITRISE_DEPLOY_DIR, and its full path is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable
	(value: %s)`, rawXcodebuildOutputLogPath)
			}
		}
	}

	fmt.Println()
	if xcresultPath != "" {
		// export xcresult bundle
		if err := tools.ExportEnvironmentWithEnvman("BITRISE_XCRESULT_PATH", xcresultPath); err != nil {
			log.Warnf("Failed to export: BITRISE_XCRESULT_PATH, error: %s", err)
		} else {
			log.Printf("Exported BITRISE_XCRESULT_PATH: %s", xcresultPath)
		}
	}

	if xcErr != nil {
		fail("Analyze failed: %s", xcErr)
	}

	// Cache swift PM
	if xcodeMajorVersion >= 11 && conf.CacheLevel == "swift_packages" {
		if err := cache.CollectSwiftPackages(absProjectPath); err != nil {
			log.Warnf("Failed to mark swift packages for caching, error: %s", err)
		}
	}
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
