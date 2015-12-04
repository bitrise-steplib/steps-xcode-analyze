#!/bin/bash

set -e

#
# Required parameters
if [ -z "${project_path}" ] ; then
	echo "[!] Missing required input: project_path"
	exit 1
fi
if [ -z "${scheme}" ] ; then
	echo "[!] Missing required input: scheme"
	exit 1
fi

#
# Project-or-Workspace flag
if [[ "${project_path}" == *".xcodeproj" ]]; then
	export CONFIG_xcode_project_action="-project"
elif [[ "${project_path}" == *".xcworkspace" ]]; then
	export CONFIG_xcode_project_action="-workspace"
else
	echo "Failed to get valid project file (invalid project file): ${project_path}"
	exit 1
fi

if [[ "${output_tool}" != "xcpretty" && "${output_tool}" != "xcodebuild" ]] ; then
	echo "[!] Invalid output_tool: ${output_tool}"
	exit 1
fi

set +e

xcpretty_version=""
if [[ "${output_tool}" == "xcpretty" ]] ; then
	xcpretty_version=$(xcpretty --version)
	exit_code=$?
	if [[ $exit_code != 0 || -z "$xcpretty_version" ]] ; then
		echo
		echo " (!) xcpretty is not installed"
		echo "     For xcpretty installation see: 'https://github.com/supermarin/xcpretty',"
		echo "     or use 'xcodebuild' as 'output_tool'."
		echo
		exit 1
	fi
fi

set -e


#
# Print configs
echo
echo "========== Configs =========="
echo " * output_tool: ${output_tool}"
if [[ "${output_tool}" == "xcpretty" ]] ; then
	echo " * xcpretty version: ${xcpretty_version}"
fi
echo " * xcodebuild version: $(xcodebuild -version)"
echo " * project_path: ${project_path}"
echo " * scheme: ${scheme}"
echo " * workdir: ${workdir}"
echo " * is_clean_build: ${is_clean_build}"
echo " * is_force_code_sign: ${is_force_code_sign}"
echo " * CONFIG_xcode_project_action: ${CONFIG_xcode_project_action}"
echo "============================="
echo


#
# Main
if [ ! -z "${workdir}" ] ; then
	echo
	echo "$ cd ${workdir}"
	cd "${workdir}"
fi

clean_build_param=''
if [[ "${is_clean_build}" == "yes" ]] ; then
	clean_build_param='clean'
fi


analyze_cmd="xcodebuild ${CONFIG_xcode_project_action} \"${project_path}\""
analyze_cmd="$analyze_cmd -scheme \"${scheme}\""
analyze_cmd="$analyze_cmd ${clean_build_param} analyze"

if [[ "${is_force_code_sign}" == "yes" ]] ; then
	echo " (!) Using Force Code Signing mode!"

	analyze_cmd="$analyze_cmd PROVISIONING_PROFILE=\"${BITRISE_PROVISIONING_PROFILE_ID}\""
	analyze_cmd="$analyze_cmd CODE_SIGN_IDENTITY=\"${BITRISE_CODE_SIGN_IDENTITY}\""
fi

if [[ "${output_tool}" == "xcpretty" ]] ; then
	analyze_cmd="set -o pipefail && $analyze_cmd | xcpretty"
fi

echo
echo
echo "=> Analyze command:"
echo '$' $analyze_cmd

echo
eval $analyze_cmd

exit 0
