package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-steputils/v2/ruby"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/stringutil"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/xcodecommand"
	"github.com/bitrise-io/go-xcode/v2/xcpretty"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	cache "github.com/bitrise-io/go-xcode/xcodecache"
	"github.com/bitrise-steplib/steps-xcode-archive/utils"
	"github.com/kballard/go-shellquote"
)

const (
	XcbeautifyTool = "xcbeautify"
	XcodebuildTool = "xcodebuild"
	XcprettyTool   = "xcpretty"

	xcodebuildLogFilename           = "xcodebuild-analyze.log"
	bitriseXcodeRawResultTextEnvKey = "BITRISE_XCODE_RAW_RESULT_TEXT_PATH"
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
		fail(log.NewLogger(), "Failed to parse configs, error: %s", err)
	}

	stepconf.Print(conf)
	logger := log.NewLogger(log.WithDebugLog(true))

	envRepository := env.NewRepository()
	cmdFactory := command.NewFactory(envRepository)
	pathChecker := pathutil.NewPathChecker()
	fileManager := fileutil.NewFileManager()

	xcodeCommandRunner := xcodecommand.Runner(nil)
	switch conf.OutputTool {
	case XcodebuildTool:
		xcodeCommandRunner = xcodecommand.NewRawCommandRunner(logger, cmdFactory)
	case XcbeautifyTool:
		xcodeCommandRunner = xcodecommand.NewXcbeautifyRunner(logger, cmdFactory)
	case XcprettyTool:
		commandLocator := env.NewCommandLocator()
		rubyComamndFactory, err := ruby.NewCommandFactory(cmdFactory, commandLocator)
		if err != nil {
			fail(logger, "failed to install xcpretty: %s", err)
		}
		rubyEnv := ruby.NewEnvironment(rubyComamndFactory, commandLocator, logger)

		xcodeCommandRunner = xcodecommand.NewXcprettyCommandRunner(logger, cmdFactory, pathChecker, fileManager, rubyComamndFactory, rubyEnv)
	default:
		panic(fmt.Sprintf("Unknown log formatter: %s", conf.OutputTool))
	}

	fmt.Println()
	logger.Infof("Step determined configs:")

	absProjectPath, err := filepath.Abs(conf.ProjectPath)
	if err != nil {
		fail(logger, "Failed to expand project path (%s), error: %s", conf.ProjectPath, err)
	}

	xcprettyInstance := xcpretty.NewXcpretty(logger)

	// Detect xcpretty version
	outputTool := conf.OutputTool
	if outputTool == "xcpretty" {
		fmt.Println()
		logger.Infof("Checking if output tool (xcpretty) is installed")

		installed, err := xcprettyInstance.IsInstalled()
		if err != nil {
			logger.Warnf("Failed to check if xcpretty is installed, error: %s", err)
			logger.Printf("Switching to xcodebuild for output tool")
			outputTool = "xcodebuild"
		} else if !installed {
			logger.Warnf(`xcpretty is not installed`)
			fmt.Println()
			logger.Printf("Installing xcpretty")

			if cmds, err := xcprettyInstance.Install(); err != nil {
				logger.Warnf("Failed to create xcpretty install command: %s", err)
				logger.Warnf("Switching to xcodebuild for output tool")
				outputTool = "xcodebuild"
			} else {
				for _, cmd := range cmds {
					if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
						if errorutil.IsExitStatusError(err) {
							logger.Warnf("%s failed: %s", out)
						} else {
							logger.Warnf("%s failed: %s", err)
						}
						logger.Warnf("Switching to xcodebuild for output tool")
						outputTool = "xcodebuild"
					}
				}
			}
		}
	}

	if outputTool == "xcpretty" {
		xcprettyVersion, err := xcprettyInstance.Version()
		if err != nil {
			logger.Warnf("Failed to determine xcpretty version, error: %s", err)
			logger.Printf("Switching to xcodebuild for output tool")
			outputTool = "xcodebuild"
		}
		logger.Printf("- xcprettyVersion: %s", xcprettyVersion.String())
	}

	// Output files
	rawXcodebuildOutputLogPath := filepath.Join(conf.OutputDir, "raw-xcodebuild-output.log")

	tempDir, err := os.MkdirTemp("", "XCOutput")
	if err != nil {
		fail(logger, "Could not create result bundle path directory: %s", err)
	}
	xcresultPath := path.Join(tempDir, "Analyze.xcresult")

	//
	// Cleanup
	filesToCleanup := []string{
		rawXcodebuildOutputLogPath,
	}

	for _, pth := range filesToCleanup {
		if exist, err := pathChecker.IsPathExists(pth); err != nil {
			fail(logger, "Failed to check if path (%s) exist, error: %s", pth, err)
		} else if exist {
			if err := os.RemoveAll(pth); err != nil {
				fail(logger, "Failed to remove path (%s), error: %s", pth, err)
			}
		}
	}

	//
	// Analyze project with Xcode Command Line tools
	fmt.Println()
	logger.Infof("Analyzing the project")

	analyzeCmd := xcodebuild.NewCommandBuilder(absProjectPath, "analyze")

	analyzeCmd.SetScheme(conf.Scheme)

	if conf.DisableCodesign {
		analyzeCmd.SetDisableCodesign(true)
	}

	var customOptions []string
	if conf.XcodebuildOptions != "" {
		if customOptions, err = shellquote.Split(conf.XcodebuildOptions); err != nil {
			fail(logger, "failed to shell split XcodebuildOptions (%s), error: %s", conf.XcodebuildOptions, err)
		}
	}

	if conf.DisableIndexWhileBuilding {
		customOptions = append(customOptions, "COMPILER_INDEX_STORE_ENABLE=NO")
	}

	analyzeCmd.SetCustomOptions(customOptions)

	if !sliceutil.IsStringInSlice("-resultBundlePath", customOptions) {
		analyzeCmd.SetResultBundlePath(xcresultPath)
	}

	swiftPackagesPath, err := cache.SwiftPackagesPath(absProjectPath)
	if err != nil {
		fail(logger, "Failed to get Swift Packages path, error: %s", err)
	}

	rawXcodebuildOut, xcErr := runCommandWithRetry(xcodeCommandRunner, conf.OutputTool, analyzeCmd, swiftPackagesPath, logger)
	if xcErr != nil {
		if outputTool == "xcpretty" {
			logger.Errorf("\nLast lines of the Xcode's build log:")
			fmt.Println(stringutil.LastNLines(rawXcodebuildOut, 10))

			if err := utils.ExportOutputFileContent(rawXcodebuildOut, rawXcodebuildOutputLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
				logger.Warnf("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
			} else {
				logger.Warnf(`You can find the last couple of lines of Xcode's build log above, but the full log is also available in the raw-xcodebuild-output.log
	The log file is stored in $BITRISE_DEPLOY_DIR, and its full path is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable
	(value: %s)`, rawXcodebuildOutputLogPath)
			}
		}
	}

	fmt.Println()
	if xcresultPath != "" {
		// export xcresult bundle
		if err := tools.ExportEnvironmentWithEnvman("BITRISE_XCRESULT_PATH", xcresultPath); err != nil {
			logger.Warnf("Failed to export: BITRISE_XCRESULT_PATH, error: %s", err)
		} else {
			logger.Printf("Exported BITRISE_XCRESULT_PATH: %s", xcresultPath)
		}
	}

	if xcErr != nil {
		fail(logger, "Analyze failed: %s", xcErr)
	}

	// Cache swift PM
	if conf.CacheLevel == "swift_packages" {
		if err := cache.CollectSwiftPackages(absProjectPath); err != nil {
			logger.Warnf("Failed to mark swift packages for caching, error: %s", err)
		}
	}
}

func fail(logger log.Logger, format string, v ...interface{}) {
	logger.Errorf(format, v...)
	os.Exit(1)
}
