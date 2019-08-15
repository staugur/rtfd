#!/bin/bash
#Author:      staugur
#Version:     0.1
#Description: 最终调用的核心脚本，此脚本只负责构建，会在docs的项目下，生成不同语言和不同版本的文档
#CreateTime:  2019-08-05
#License:     BSD 3-Clause
#Copyright:   (c) 2019 by staugur.

check_exit_param() {
    local n=$1
    local c=$2
    if [[ -z "${c}" || "x" == "x${c}" || "${c:0:1}" == "-" ]]; then
        echo "Invalid param in ${n}"
        exit 128
    fi
}

check_exit_retcode() {
    local code=$?
    if [[ $code -ne 0 || "${code}" != "0" ]]; then
        echo "Command sending error code $code in $(pwd), the traceback stack:"
        echo "    $(caller 0)"
        echo "    $(caller 1)"
        exit 128
    fi
}

_readINI() {
    #: _readINI [配置文件路径+名称] [节点名] [键值]
    local INIFILE=$1
    local SECTION=$2
    local ITEM=$3
    _readIni=$(awk -F '=' '/['$SECTION']/{a=1}a==1&&$1~/'$ITEM'/{print $2;exit}' $INIFILE)
    echo ${_readIni}
}

_join_path() {
    echo "${1:+$1/}$2" | sed 's#//#/#g'
}

_env_manager() {
    #: 切换到项目中，创建虚拟环境并构建文档
    local py2_path=$1
    local py3_path=$2
    local project_runtime_dir=$3
    local project_docs_dir=$4
    local branch=$5
    local project_name=$6
    check_exit_param _env_manager_py2_path $py2_path
    check_exit_param _env_manager_py3_path $py3_path
    check_exit_param _env_manager_project_runtime_dir $project_runtime_dir
    check_exit_param _env_manager_project_docs_dir $project_docs_dir
    check_exit_param _env_manager_branch $branch
    check_exit_param _env_manager_project_name $project_name
    cd ${project_runtime_dir}
    check_exit_retcode
    #: 尝试读取项目的文档配置文件
    project_ini=".rtfd.ini"
    if [ -f $project_ini ]; then
        #: 采用ini格式作为配置文件，示例如下：
        #: [project]
        #: latest = master
        #: single = false
        #: [sphinx]
        #: sourcedir = docs
        #: languages = en,zh_CN
        #: [python]
        #: version = 2
        #: requirements = docs/requirements.txt,dev-requirements
        #: install = true
        #: index =  pypi source
        local project_latest=$(_readINI $project_ini project latest)
        local sphinx_sourcedir=$(_readINI $project_ini sphinx sourcedir)
        local sphinx_languages=$(_readINI $project_ini sphinx languages)
        local py_version=$(_readINI $project_ini python version)
        local py_requirements=$(_readINI $project_ini python requirements)
        local py_install_project=$(_readINI $project_ini python install)
        local py_index=$(_readINI $project_ini python index)
    fi
    local project_latest=${project_latest:=master}
    local sphinx_sourcedir=${sphinx_sourcedir:=docs}
    if [[ "${sphinx_sourcedir:0:1}" == "/" || "${sphinx_sourcedir:0:2}" == ".." ]]; then
        echo "In rtfd.ini, sourcedir cannot start with / or .."
        exit 1
    fi
    local sphinx_languages=${sphinx_languages:=en}
    local py_version=${py_version:=2}
    local py_requirements=${py_requirements:=}
    local py_install_project=${py_install_project:=false}
    local py_index=${py_index:=https://pypi.org/simple}
    case $py_version in
    2)
        local py_path=$py2_path
        ;;
    3)
        local py_path=$py3_path
        ;;
    *)
        local py_path=$py2_path
        ;;
    esac
    local vd="venv-${py_version}"
    local venv="${py_path} -m virtualenv -p ${py_path} --no-site-packages"
    #: 创建虚拟环境
    if [ ! -d $vd ]; then
        $venv $vd
        check_exit_retcode
    fi
    #: 激活虚拟环境
    source ${vd}/bin/activate
    check_exit_retcode
    #: 安装依赖
    local venv_py=$(_join_path $project_runtime_dir ${vd}/bin/python)
    local venv_pip_install="${venv_py} -m pip install -i ${py_index} --no-cache-dir"
    $venv_pip_install --upgrade pip sphinx setuptools
    for req in ${py_requirements//,/ }; do
        $venv_pip_install -r $req
        check_exit_retcode
    done
    if [ "${py_install_project}" = "true" ]; then
        $venv_pip_install .
        check_exit_retcode
    fi
    #: 更新conf.py
    local sphinx_conf=$(_join_path $sphinx_sourcedir conf.py)
    if [ ! -f $sphinx_conf ]; then
        echo "Not found docs config.py in $(_join_path $project_runtime_dir $sphinx_sourcedir)"
        exit 1
    fi
    cat >>$sphinx_conf <<EOF
#: Auto-created by rtfd.
if not 'html_js_files' in globals():
    html_js_files = []
html_js_files.append("https://static.saintic.com/rtfd/rtfd.js?name=${project_name}&branch=${branch}")
if not 'html_favicon' in globals():
    html_favicon = 'https://static.saintic.com/rtfd/favicon.png'
EOF
    #: 构建
    local sphinx_build=$(_join_path $project_runtime_dir ${vd}/bin/sphinx-build)
    for lang in ${sphinx_languages//,/ }; do
        local project_docs_lang_dir=$(_join_path ${project_docs_dir} ${lang})
        $sphinx_build -E -T -D language=${lang} -b html $sphinx_sourcedir $(_join_path ${project_docs_lang_dir} ${branch})
        check_exit_retcode
        ln -nsf $(_join_path ${project_docs_lang_dir} ${project_latest}) $(_join_path ${project_docs_lang_dir} latest)
        check_exit_retcode
    done
    #: 退出虚拟环境
    deactivate
    return $?
}

_code_manager() {
    #: 检出代码并切换到相应分支
    local git="git"
    local project_name=$1
    local project_git=$2
    local branch=$3
    local runtime_dir=$4
    check_exit_param _code_manager_project_name $project_name
    check_exit_param _code_manager_project_git $project_git
    check_exit_param _code_manager_branch $branch
    check_exit_param _code_manager_runtime_dir $runtime_dir
    cd $runtime_dir
    check_exit_retcode
    [ -d $project_name ] && rm -rf $project_name
    $git clone --recurse-submodules --tags $project_git $project_name
    check_exit_retcode
    cd $project_name
    check_exit_retcode
    local tags=($($git tag -l))
    local branches=(master ${tags[@]})
    if echo "${branches[@]}" | grep -w "${branch}" &>/dev/null; then
        if [ "$branch" != "master" ]; then
            $git checkout $branch
            check_exit_retcode
            return 0
        fi
    else
        echo "Not found ${branch}, the allowed branches are ${branches[@]}"
        exit 1
    fi
}

usage() {
    printf "
Usage: $0 [options]

Options:

    -h, --help    The help information
    -n, --name    The docs project name
    -u, --url     The docs project git url
    -b, --branch  The docs project branch, default is master.
    -c, --config  The config file, default is ${HOME}/.rtfd.cfg
"
    return $?
}

main() {
    local stime=$SECONDS
    if [ $# -eq 0 ]; then
        usage
        exit 1
    fi
    while [ $# -gt 0 ]; do
        case "$1" in
        -n | --name)
            local project_name="${2,,}"
            check_exit_param project_name $project_name
            shift
            ;;
        -u | --url)
            local project_git="$2"
            check_exit_param project_git $project_git
            shift
            ;;
        -b | --branch)
            local branch="${2}"
            check_exit_param branch $branch
            shift
            ;;
        -c | --config)
            local config="${2}"
            check_exit_param config $config
            shift
            ;;
        -h | --help | \?)
            usage
            exit 0
            ;;
        --)
            shift
            break
            ;;
        *)
            break
            ;;
        esac
        shift
    done
    #: 设置默认配置
    local branch=${branch:=master}
    #: 读取用户级配置文件
    local rtfd_ini="${config:=$HOME/.rtfd.cfg}"
    if [ ! -f $rtfd_ini ]; then
        echo "Not found config file $rtfd_ini"
        exit 1
    fi
    local base_dir=$(_readINI $rtfd_ini g base_dir)
    local py2_path=$(_readINI $rtfd_ini py py2)
    local py3_path=$(_readINI $rtfd_ini py py3)
    #:
    echo -e "$base_dir $py2_path $py3_path\n$project_name $project_git $branch"
    #: 校验参数
    check_exit_param project_name $project_name
    check_exit_param project_git $project_git
    check_exit_param branch $branch
    if [[ -x $py2_path && -x $py3_path && -d $base_dir ]]; then
        local docs_dir=$(_join_path $base_dir docs)
        local runtimes_dir=$(_join_path $base_dir runtimes)
        [ -d $docs_dir ] || mkdir $docs_dir
        [ -d $runtimes_dir ] || mkdir $runtimes_dir
        local runtimes_dir=$(mktemp -d -p $runtimes_dir)
        _code_manager $project_name $project_git $branch $runtimes_dir
        check_exit_retcode
        local project_docs_dir=$(_join_path $docs_dir $project_name)
        local project_runtime_dir=$(_join_path $runtimes_dir $project_name)
        _env_manager $py2_path $py3_path $project_runtime_dir $project_docs_dir $branch $project_name
        check_exit_retcode
        local utime=$(($SECONDS - $stime))
        echo "Build Successfully, $utime seconds passed."
        rm -rf $runtimes_dir
        return 0
    else
        echo "Configuration information is wrong in ${rtfd_ini}"
        exit 1
    fi
}

Clean() {
    echo "The program was terminated, will exit!"
    exit 1
}

trap 'Clean; exit' SIGINT SIGTERM

main "$@"
