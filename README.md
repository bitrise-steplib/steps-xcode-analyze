# Xcode Analyze

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-xcode-analyze?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-xcode-analyze/releases)

Find flaws and potential bugs in the source code of an app with the
static analyzer built into Xcode.

<details>
<summary>Description</summary>

The Step uses the static analyzer built directly into Xcode to analyze your app's source code: the static analyzer tries out thousands of possible code paths in a few seconds, reporting potential bugs that might have remained hidden or bugs that might be nearly impossible to replicate. 

This process also identifies areas in your code that don’t follow recommended API usage, such as Foundation, UIKit, and AppKit idioms. 

### Configuring the Step 

In most cases, you don't need to change the Step's configuration. The default input values work well if you added your iOS app on the website, using automatic configuration. 

To make sure the Step works well for you:

1. Make sure the **Project (or Workspace) path** points to the path of the `.xcodeproj` or `.xcworkspace` file of your app, relative to the app's root directory.
1. Make sure the **Scheme name** input points to a valid shared Xcode scheme. Note that it must be a shared scheme! 
1. Optionally, you can force the Step to use specific code signing identities. To do so, use the **Force code signing with Identity** and **Force code signing with Provisioning Profile** inputs. 

   For detailed instructions on their use, see the inputs themselves.

### Useful links 

* [Running Xcode tests](https://devcenter.bitrise.io/testing/running-xcode-tests/)
* [Device testing for iOS](https://devcenter.bitrise.io/testing/device-testing-for-ios/)

### Related Steps 

* [Xcode build for simulator](https://app.bitrise.io/integrations/steps/xcode-build-for-simulator)
* [Xcode Test for iOS](https://app.bitrise.io/integrations/steps/xcode-test)
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `workdir` | Working directory of the Step. If you leave it empty, the default working directory will be used.  |  | `$BITRISE_SOURCE_DIR` |
| `project_path` | The path to your app's `.xcodeproj` or `.xcworkspace` file, relative to the Step's working directory (if one is specified).  | required | `$BITRISE_PROJECT_PATH` |
| `scheme` | The Xcode scheme to use for the analysis. **IMPORTANT**: The scheme must be marked as shared in Xcode!  | required | `$BITRISE_SCHEME` |
| `is_clean_build` |  | required | `no` |
| `force_code_sign_identity` | Force the `xcodebuild` command to use specified code signing identity. Specify a code signing identity as a full ID (for example, `iPhone Developer: Bitrise Bot (VV2J4SV8V4)`) or specify a code signing group (for example, `iPhone Developer` or `iPhone Distribution`). |  |  |
| `force_provisioning_profile` | Force the `xcodebuild` command to use a specified provisioning profile. You must use the provisioning profile's UUID. The profile's name is NOT accepted by xcodebuild. To get your UUID: - In Xcode select your project -> Build Settings -> Code Signing - Select the desired Provisioning Profile, then scroll down in profile list and click on Other... - The popup will show your profile's UUID. Format example: - c5be4123-1234-4f9d-9843-0d9be985a068 |  |  |
| `disable_codesign` | In order to skip code signing, set this option to `yes`. |  | `yes` |
| `disable_index_while_building` | Add `COMPILER_INDEX_STORE_ENABLE=NO` flag to the `xcodebuild` command which will disable the indexing during the build. Indexing is needed for  * Autocomplete. * Ability to quickly jump to definition. * Get class and method help by alt clicking. None of the above ar needed in a CI environment. **Note:** In Xcode you can turn off the `Index-WhileBuilding` feature  by disabling the `Enable Index-WhileBuilding Functionality` in the `Build Settings`.<br/> In a CI environment you can disable it by adding `COMPILER_INDEX_STORE_ENABLE=NO` flag to the `xcodebuild` command. |  | `yes` |
| `cache_level` | Available options: - `none` : Disable caching. - `swift_packages` : Cache Swift PM packages added to the Xcode project. | required | `swift_packages` |
| `xcodebuild_options` | Options added to the end of the xcodebuild call. You can use multiple options, separated by a space character. Example: `-xcconfig PATH -verbose` |  |  |
| `output_tool` | If the input is set to `xcpretty`, the xcodebuild output will be prettified by xcpretty. If the input is set to `xcodebuild`, the raw xcodebuild output will be printed. | required | `xcpretty` |
| `output_dir` | This directory will contain the generated `raw-xcodebuild-output.log`. | required | `$BITRISE_DEPLOY_DIR` |
| `verbose_log` | Enable verbose logging? | required | `no` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BITRISE_XCRESULT_PATH` | The path of the generated `.xcresult`. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-xcode-analyze/pulls) and [issues](https://github.com/bitrise-steplib/steps-xcode-analyze/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
