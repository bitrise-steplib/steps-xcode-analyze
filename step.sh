#!/usr/bin/env bash

set -e

#=======================================
# Functions
#=======================================

RESTORE='\033[0m'
RED='\033[00;31m'
YELLOW='\033[00;33m'
BLUE='\033[00;34m'
GREEN='\033[00;32m'

function color_echo {
	color=$1
	msg=$2
	echo -e "${color}${msg}${RESTORE}"
}

function echo_fail {
	msg=$1
	echo
	color_echo "${RED}" "${msg}"
	exit 1
}

function echo_warn {
	msg=$1
	color_echo "${YELLOW}" "${msg}"
}

function echo_info {
	msg=$1
	echo
	color_echo "${BLUE}" "${msg}"
}

function echo_details {
	msg=$1
	echo "  ${msg}"
}

function echo_done {
	msg=$1
	color_echo "${GREEN}" "  ${msg}"
}

function validate_required_input {
	key=$1
	value=$2
	if [[ -z "${value}" ]] ; then
		echo_fail "[!] Missing required input: ${key}"
	fi
}

function validate_required_input_with_options {
	key=$1
	value=$2
	options=$3

	validate_required_input "${key}" "${value}"

	found="0"
	for option in "${options[@]}" ; do
		if [[ "${option}" == "${value}" ]] ; then
			found="1"
      break
		fi
	done

	if [[ "${found}" == "0" ]] ; then
		echo_fail "Invalid input: (${key}) value: (${value}), valid options: ($( IFS=$", "; echo "${options[*]}" ))"
	fi
}

#=======================================
# Main
#=======================================

#
# Validate parameters
echo_info "Configs:"
echo_details "* workdir: $workdir"
echo_details "* project_path: $project_path"
echo_details "* scheme: $scheme"
echo_details "* is_clean_build: $is_clean_build"
echo_details "* force_provisioning_profile: $force_provisioning_profile"
echo_details "* force_code_sign_identity: $force_code_sign_identity"
echo_details "* disable_codesign: $disable_codesign"
echo_details "* output_tool: $output_tool"

validate_required_input "project_path" $project_path
validate_required_input "scheme" $scheme
validate_required_input "is_clean_build" $is_clean_build
validate_required_input "output_tool" $output_tool

options=("xcpretty"  "xcodebuild")
validate_required_input_with_options "output_tool" $output_tool "${options[@]}"

echo

# xcodebuild version
out=$(xcodebuild -version)

IFS=$'\n'
xcodebuild_version_split=($out)
unset IFS

xcodebuild_version="${xcodebuild_version_split[0]} (${xcodebuild_version_split[1]})"
echo_details "* xcodebuild_version: $xcodebuild_version"

# Detect xcpretty version
xcpretty_version=""
if [[ "${output_tool}" == "xcpretty" ]] ; then
	xcpretty_version=$(xcpretty --version)
	exit_code=$?
	if [[ $exit_code != 0 || -z "$xcpretty_version" ]] ; then
		echo_fail "xcpretty is not installed
		For xcpretty installation see: 'https://github.com/supermarin/xcpretty',
		or use 'xcodebuild' as 'output_tool'.
		"
	fi

	echo_details "* xcpretty_version: $xcpretty_version"
fi

# Project-or-Workspace flag
if [[ "${project_path}" == *".xcodeproj" ]]; then
	CONFIG_xcode_project_action="-project"
elif [[ "${project_path}" == *".xcworkspace" ]]; then
	CONFIG_xcode_project_action="-workspace"
else
	echo_fail "Failed to get valid project file (invalid project file): ${project_path}"
fi
echo_details "* CONFIG_xcode_project_action: $CONFIG_xcode_project_action"

# work dir
if [[ ! -z "${workdir}" ]] ; then
	echo_info "Switching to working directory: ${workdir}"
	cd "${workdir}"
fi

#
# Main
echo_info "Analyzing the project..."

analyze_cmd="xcodebuild ${CONFIG_xcode_project_action} \"${project_path}\""
analyze_cmd="$analyze_cmd -scheme \"${scheme}\""
if [[ "${is_clean_build}" == "yes" ]] ; then
	analyze_cmd="$analyze_cmd clean"
fi
analyze_cmd="$analyze_cmd analyze"

if [[ -n "${force_code_sign_identity}" ]] ; then
	echo_details "Forcing Code Signing Identity: ${force_code_sign_identity}"

	analyze_cmd="$analyze_cmd CODE_SIGN_IDENTITY=\"${force_code_sign_identity}\""
fi

if [[ -n "${force_provisioning_profile}" ]] ; then
	echo_details "Forcing Provisioning Profile: ${force_provisioning_profile}"

	analyze_cmd="$analyze_cmd PROVISIONING_PROFILE=\"${force_provisioning_profile}\""
fi

if [[ "${disable_codesign}" == "yes" ]] ; then
	echo_details "Disable Code Signing"

	analyze_cmd="$analyze_cmd CODE_SIGN_IDENTITY="" CODE_SIGNING_REQUIRED=NO"
fi

if [[ "${output_tool}" == "xcpretty" ]] ; then
	analyze_cmd="set -o pipefail && $analyze_cmd | xcpretty"
fi

echo_details "$ $analyze_cmd"
echo

eval $analyze_cmd

exit 0
