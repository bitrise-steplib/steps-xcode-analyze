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

#
# Print configs
echo
echo "========== Configs =========="
echo " * project_path: ${project_path}"
echo " * scheme: ${scheme}"
echo " * workdir: ${workdir}"
echo " * is_clean_build: ${is_clean_build}"
echo " * CONFIG_xcode_project_action: ${CONFIG_xcode_project_action}"


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

set -v

xcodebuild ${CONFIG_xcode_project_action} "${project_path}" \
	-scheme "${scheme}" \
	${clean_build_param} analyze \
	-verbose

exit 0
